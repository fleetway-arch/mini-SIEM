package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/detection"
	"siem-dashboard/server/internal/models"
)

// Collections whose documents get small sequential ids from "counters". The
// ids reach the browser, so they must stay below 2^53.
var countedCollections = []string{"events", "alerts", "ingest_keys"}

// seedCounters starts each counter after the highest id already stored, so
// data inserted before counters existed keeps unique ids.
func (m *Mongo) seedCounters(ctx context.Context) error {
	for _, name := range countedCollections {
		var last struct {
			ID int `bson:"id"`
		}
		err := m.database.Collection(name).FindOne(ctx, bson.M{},
			options.FindOne().SetSort(bson.D{{Key: "id", Value: -1}}).SetProjection(bson.M{"id": 1})).Decode(&last)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
		if _, err := m.database.Collection("counters").UpdateOne(ctx, bson.M{"_id": name},
			bson.M{"$max": bson.M{"seq": last.ID}}, options.Update().SetUpsert(true)); err != nil {
			return err
		}
	}
	return nil
}

// nextIDs reserves n consecutive ids and returns the first one.
func (m *Mongo) nextIDs(ctx context.Context, name string, n int) (int, error) {
	var counter struct {
		Seq int `bson:"seq"`
	}
	err := m.database.Collection("counters").FindOneAndUpdate(ctx, bson.M{"_id": name},
		bson.M{"$inc": bson.M{"seq": n}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&counter)
	return counter.Seq - n + 1, err
}

func eventFilter(ctx context.Context, query models.EventQuery) (bson.D, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	if query.Source != "" {
		filter = append(filter, bson.E{Key: "source", Value: query.Source})
	}
	if query.Severity != "" {
		filter = append(filter, bson.E{Key: "severity", Value: query.Severity})
	}
	// The selected address is an exact IP; the ip filter is a prefix typed by
	// the user. Both apply when both are set.
	ip := bson.D{}
	if query.Address != "" {
		ip = append(ip, bson.E{Key: "$eq", Value: query.Address})
	}
	if query.IP != "" {
		ip = append(ip, bson.E{Key: "$regex", Value: "^" + regexp.QuoteMeta(query.IP)})
	}
	if len(ip) > 0 {
		filter = append(filter, bson.E{Key: "ip", Value: ip})
	}
	timestamp := bson.D{}
	if query.From != "" {
		timestamp = append(timestamp, bson.E{Key: "$gte", Value: query.From})
	}
	if query.To != "" {
		timestamp = append(timestamp, bson.E{Key: "$lt", Value: query.To})
	}
	if len(timestamp) > 0 {
		filter = append(filter, bson.E{Key: "timestamp", Value: timestamp})
	}
	if query.Q != "" {
		text := bson.D{{Key: "$regex", Value: regexp.QuoteMeta(query.Q)}, {Key: "$options", Value: "i"}}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "message", Value: text}},
			bson.D{{Key: "rawLog", Value: text}},
		}})
	}
	return filter, nil
}

// ListEvents returns one page of the company's events, newest first, and
// the number of events matching the filters.
func (m *Mongo) ListEvents(ctx context.Context, query models.EventQuery) (models.EventPage, error) {
	filter, err := eventFilter(ctx, query)
	if err != nil {
		return models.EventPage{}, err
	}
	collection := m.database.Collection("events")
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return models.EventPage{}, err
	}
	cursor, err := collection.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}, {Key: "id", Value: -1}}).
		SetSkip(int64((query.Page-1)*query.Limit)).
		SetLimit(int64(query.Limit)))
	if err != nil {
		return models.EventPage{}, err
	}
	defer cursor.Close(ctx)
	events := make([]models.Event, 0, query.Limit)
	if err := cursor.All(ctx, &events); err != nil {
		return models.EventPage{}, err
	}
	return models.EventPage{Items: events, Total: total}, nil
}

// InsertEvents stores events for the authenticated company and returns them
// with their ids.
func (m *Mongo) InsertEvents(ctx context.Context, events []models.Event) ([]models.Event, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return events, nil
	}
	companyID := filter[0].Value.(int)
	first, err := m.nextIDs(ctx, "events", len(events))
	if err != nil {
		return nil, err
	}
	stored := make([]models.Event, len(events))
	documents := make([]any, len(events))
	for i, event := range events {
		event.ID = first + i
		stored[i] = event
		documents[i] = models.EventDocument{Event: event, CompanyID: companyID}
	}
	if _, err := m.database.Collection("events").InsertMany(ctx, documents); err != nil {
		return nil, err
	}
	return stored, nil
}

// CountFailedEvents counts the company's failed authentications from ip
// since the given RFC 3339 UTC timestamp.
func (m *Mongo) CountFailedEvents(ctx context.Context, ip, since string) (int, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return 0, err
	}
	failed := bson.D{{Key: "$regex", Value: detection.FailedPattern}, {Key: "$options", Value: "i"}}
	filter = append(filter,
		bson.E{Key: "ip", Value: ip},
		bson.E{Key: "timestamp", Value: bson.D{{Key: "$gte", Value: since}}},
		bson.E{Key: "$or", Value: bson.A{bson.D{{Key: "message", Value: failed}}, bson.D{{Key: "rawLog", Value: failed}}}},
	)
	count, err := m.database.Collection("events").CountDocuments(ctx, filter)
	return int(count), err
}

// RecordAlert adds the finding to the unresolved alert of the same rule and
// IP, or opens a new alert. Two batches racing on the same IP can open two
// alerts; analysts resolve the duplicate.
func (m *Mongo) RecordAlert(ctx context.Context, finding detection.Finding) (models.Alert, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.Alert{}, err
	}
	companyID := filter[0].Value.(int)
	filter = append(filter,
		bson.E{Key: "rule", Value: finding.Rule},
		bson.E{Key: "ip", Value: finding.IP},
		bson.E{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{"open", "acknowledged"}}}},
	)
	alerts := m.database.Collection("alerts")
	var alert models.Alert
	err = alerts.FindOneAndUpdate(ctx, filter,
		bson.M{"$inc": bson.M{"eventCount": finding.NewEvents}, "$max": bson.M{"lastSeenAt": finding.LastSeen}},
		options.FindOneAndUpdate().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetReturnDocument(options.After),
	).Decode(&alert)
	if err == nil || !errors.Is(err, mongo.ErrNoDocuments) {
		return alert, err
	}
	id, err := m.nextIDs(ctx, "alerts", 1)
	if err != nil {
		return models.Alert{}, err
	}
	alert = models.Alert{
		ID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339), Rule: finding.Rule,
		Severity: finding.Severity, IP: finding.IP, Description: finding.Description,
		EventCount: finding.EventCount, Status: "open", LastSeenAt: finding.LastSeen,
	}
	document := struct {
		models.Alert `bson:",inline"`
		CompanyID    int `bson:"companyId"`
	}{alert, companyID}
	_, err = alerts.InsertOne(ctx, document)
	return alert, err
}
