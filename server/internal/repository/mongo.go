package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
)

type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongo(ctx context.Context, uri, databaseName string) (*Mongo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}
	return &Mongo{client: client, database: client.Database(databaseName)}, nil
}

func (m *Mongo) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *Mongo) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "companyId", Value: 1}, {Key: "timestamp", Value: -1}},
			Options: options.Index().SetName("events_company_timestamp"),
		},
		{
			Keys:    bson.D{{Key: "companyId", Value: 1}, {Key: "ip", Value: 1}},
			Options: options.Index().SetName("events_company_ip"),
		},
	}
	_, err := m.database.Collection("events").Indexes().CreateMany(ctx, indexes)
	return err
}

func (m *Mongo) FindUserByEmail(ctx context.Context, email string) (models.UserDocument, error) {
	var user models.UserDocument
	err := m.database.Collection("users").FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&user)
	return user, err
}

func (m *Mongo) ListTeam(ctx context.Context) ([]models.TeamMember, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	cursor, err := m.database.Collection("users").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []models.UserDocument
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	result := make([]models.TeamMember, 0, len(users))
	for _, user := range users {
		result = append(result, models.TeamMember{ID: user.ID, Email: user.Email, FullName: user.FullName, Role: user.Role, Status: "active"})
	}
	return result, nil
}

func (m *Mongo) CreateInvitations(ctx context.Context, emails []string, role string) (models.BulkInvitationResult, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.BulkInvitationResult{}, err
	}
	companyID := filter[0].Value.(int)
	email, ok := AuthenticatedEmail(ctx)
	if !ok {
		return models.BulkInvitationResult{}, ErrMissingCompanyContext
	}
	user, err := m.FindUserByEmail(ctx, email)
	if err != nil {
		return models.BulkInvitationResult{}, err
	}
	result := models.BulkInvitationResult{Created: []models.TeamMember{}, Skipped: []models.SkippedInvitation{}}
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}
		userFilter := append(bson.D{}, filter...)
		userFilter = append(userFilter, bson.E{Key: "email", Value: email})
		count, err := m.database.Collection("users").CountDocuments(ctx, userFilter)
		if err != nil {
			return result, err
		}
		if count > 0 {
			result.Skipped = append(result.Skipped, models.SkippedInvitation{Email: email, Reason: "already exists"})
			continue
		}
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			return result, err
		}
		token := hex.EncodeToString(tokenBytes)
		invitation := models.InvitationDocument{Token: token, CompanyID: companyID, CompanyName: user.CompanyName, Email: email, Role: role, ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
		if _, err := m.database.Collection("invitations").InsertOne(ctx, invitation); err != nil {
			return result, err
		}
		result.Created = append(result.Created, models.TeamMember{ID: len(result.Created) + 1, Email: email, Role: role, Status: "invited", InvitedAt: time.Now().UTC().Format(time.RFC3339)})
	}
	return result, nil
}

func (m *Mongo) UpdateMember(ctx context.Context, id int, changes map[string]string) (models.TeamMember, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.TeamMember{}, err
	}
	filter = append(filter, bson.E{Key: "id", Value: id})
	update := bson.M{}
	for key, value := range changes {
		if key == "role" || key == "status" {
			update[key] = value
		}
	}
	var user models.UserDocument
	err = m.database.Collection("users").FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&user)
	if err != nil {
		return models.TeamMember{}, err
	}
	return models.TeamMember{ID: user.ID, Email: user.Email, FullName: user.FullName, Role: user.Role, Status: "active"}, nil
}

func (m *Mongo) DeleteMember(ctx context.Context, id int) error {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return err
	}
	filter = append(filter, bson.E{Key: "id", Value: id})
	result, err := m.database.Collection("users").DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (m *Mongo) GetInvitation(ctx context.Context, token string) (models.InvitationDetails, error) {
	var invitation models.InvitationDocument
	err := m.database.Collection("invitations").FindOne(ctx, bson.M{"token": token, "used": false}).Decode(&invitation)
	return models.InvitationDetails{CompanyName: invitation.CompanyName, Email: invitation.Email, Role: invitation.Role}, err
}

func (m *Mongo) AcceptInvitation(ctx context.Context, token, fullName, plainPassword string) (models.User, error) {
	var invitation models.InvitationDocument
	if err := m.database.Collection("invitations").FindOne(ctx, bson.M{"token": token, "used": false}).Decode(&invitation); err != nil {
		return models.User{}, err
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return models.User{}, err
	}
	user := models.UserDocument{User: models.User{ID: int(time.Now().UnixNano()), Email: invitation.Email, FullName: fullName, CompanyID: invitation.CompanyID, CompanyName: invitation.CompanyName, Role: invitation.Role}, PasswordHash: string(hash)}
	if _, err := m.database.Collection("users").InsertOne(ctx, user); err != nil {
		return models.User{}, err
	}
	_, err = m.database.Collection("invitations").UpdateOne(ctx, bson.M{"token": token}, bson.M{"$set": bson.M{"used": true}})
	return user.User, err
}

func (m *Mongo) CreateAccount(ctx context.Context, input models.SignupRequest) (models.User, error) {
	passwordHash, err := password.Hash(input.Admin.Password)
	if err != nil {
		return models.User{}, fmt.Errorf("hash signup password: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	companyID := int(time.Now().UnixNano())
	userID := companyID + 1
	company := models.CompanyDocument{
		ID: companyID, Name: input.Company.Name, Industry: input.Company.Industry,
		Size: input.Company.Size, Country: input.Company.Country, Website: input.Company.Website,
		SecurityContact: input.Company.SecurityContact, CreatedAt: now,
	}
	user := models.UserDocument{
		User: models.User{
			ID: userID, Email: strings.ToLower(input.Admin.Email), FullName: input.Admin.FullName,
			CompanyID: companyID, CompanyName: input.Company.Name, Role: "owner",
		},
		PasswordHash: string(passwordHash),
	}

	if _, err := m.database.Collection("companies").InsertOne(ctx, company); err != nil {
		return models.User{}, err
	}
	if _, err := m.database.Collection("users").InsertOne(ctx, user); err != nil {
		_, _ = m.database.Collection("companies").DeleteOne(ctx, bson.M{"id": companyID})
		return models.User{}, err
	}
	return user.User, nil
}

func (m *Mongo) GetSummary(ctx context.Context, address string) (models.StatsSummary, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.StatsSummary{}, err
	}
	if address != "" {
		filter = append(filter, bson.E{Key: "ip", Value: address})
	} else {
		filter = append(filter, bson.E{Key: "ip", Value: bson.D{{Key: "$nin", Value: bson.A{"", nil}}}})
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalEvents", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "failedLogins", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: failedEventExpression()}}}}},
			{Key: "ips", Value: bson.D{{Key: "$addToSet", Value: "$ip"}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "totalEvents", Value: 1},
			{Key: "failedLogins", Value: 1},
			{Key: "uniqueIps", Value: bson.D{{Key: "$size", Value: bson.D{{Key: "$setDifference", Value: bson.A{"$ips", bson.A{"", nil}}}}}}},
		}}},
	}
	var summary struct {
		TotalEvents  int64 `bson:"totalEvents"`
		FailedLogins int64 `bson:"failedLogins"`
		UniqueIPs    int64 `bson:"uniqueIps"`
	}
	cursor, err := m.database.Collection("events").Aggregate(ctx, pipeline)
	if err != nil {
		return models.StatsSummary{}, err
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		if err := cursor.Decode(&summary); err != nil {
			return models.StatsSummary{}, err
		}
	} else if err := cursor.Err(); err != nil {
		return models.StatsSummary{}, err
	}
	openAlertsFilter, err := CompanyFilter(ctx)
	if err != nil {
		return models.StatsSummary{}, err
	}
	openAlertsFilter = append(openAlertsFilter, bson.E{Key: "status", Value: "open"})
	if address != "" {
		openAlertsFilter = append(openAlertsFilter, bson.E{Key: "ip", Value: address})
	}
	openAlerts, err := m.database.Collection("alerts").CountDocuments(ctx, openAlertsFilter)
	if err != nil {
		return models.StatsSummary{}, err
	}
	return models.StatsSummary{
		TotalEvents: summary.TotalEvents, FailedLogins: summary.FailedLogins,
		UniqueIPs: summary.UniqueIPs, OpenAlerts: openAlerts,
	}, nil
}

func (m *Mongo) ListEvents(ctx context.Context) ([]models.Event, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	cursor, err := m.database.Collection("events").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := make([]models.Event, 0)
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (m *Mongo) GetTimeline(ctx context.Context, rangeName, address string) ([]models.TimelinePoint, error) {
	unit, binSize, duration := timelineBucket(rangeName)
	start := time.Now().UTC().Add(-duration)
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	if address != "" {
		filter = append(filter, bson.E{Key: "ip", Value: address})
	} else {
		filter = append(filter, bson.E{Key: "ip", Value: bson.D{{Key: "$nin", Value: bson.A{"", nil}}}})
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$set", Value: bson.D{{Key: "eventDate", Value: bson.D{{Key: "$convert", Value: bson.D{
			{Key: "input", Value: "$timestamp"}, {Key: "to", Value: "date"}, {Key: "onError", Value: nil}, {Key: "onNull", Value: nil},
		}}}}}}},
		{{Key: "$match", Value: bson.D{{Key: "eventDate", Value: bson.D{{Key: "$gte", Value: start}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateTrunc", Value: bson.D{
				{Key: "date", Value: "$eventDate"}, {Key: "unit", Value: unit}, {Key: "binSize", Value: binSize}, {Key: "timezone", Value: "UTC"},
			}}}},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "failed", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: failedEventExpression()}}}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "time", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "date", Value: "$_id"}, {Key: "format", Value: "%Y-%m-%dT%H:%M:%SZ"}, {Key: "timezone", Value: "UTC"},
			}}}},
			{Key: "total", Value: 1},
			{Key: "failed", Value: 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "time", Value: 1}}}},
	}
	cursor, err := m.database.Collection("events").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	points := make([]models.TimelinePoint, 0)
	if err := cursor.All(ctx, &points); err != nil {
		return nil, err
	}
	return points, nil
}

func (m *Mongo) GetTopIPs(ctx context.Context, limit int, address string) ([]models.TopIP, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	if address != "" {
		filter = append(filter, bson.E{Key: "ip", Value: address})
	} else {
		filter = append(filter, bson.E{Key: "ip", Value: bson.D{{Key: "$nin", Value: bson.A{"", nil}}}})
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$ip"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "failedCount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: failedEventExpression()}}}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$project", Value: bson.D{{Key: "_id", Value: 0}, {Key: "ip", Value: "$_id"}, {Key: "count", Value: 1}, {Key: "failedCount", Value: 1}}}},
	}
	cursor, err := m.database.Collection("events").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	points := make([]models.TopIP, 0)
	if err := cursor.All(ctx, &points); err != nil {
		return nil, err
	}
	return points, nil
}

func timelineBucket(rangeName string) (string, int, time.Duration) {
	switch rangeName {
	case "1h":
		return "minute", 5, time.Hour
	case "7d":
		return "day", 1, 7 * 24 * time.Hour
	default:
		return "hour", 1, 24 * time.Hour
	}
}

func failedEventExpression() bson.A {
	return bson.A{
		bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "$regexMatch", Value: bson.D{{Key: "input", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$message", ""}}}}, {Key: "regex", Value: "failed|failure|authentication denied|login failed"}, {Key: "options", Value: "i"}}}},
			bson.D{{Key: "$regexMatch", Value: bson.D{{Key: "input", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$rawLog", ""}}}}, {Key: "regex", Value: "failed|failure|authentication denied|login failed"}, {Key: "options", Value: "i"}}}},
		}}},
		1,
		0,
	}
}

func (m *Mongo) ListAlerts(ctx context.Context, status string) ([]models.Alert, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}
	cursor, err := m.database.Collection("alerts").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	alerts := make([]models.Alert, 0)
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (m *Mongo) UpdateAlertStatus(ctx context.Context, id int, status string) (models.Alert, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.Alert{}, err
	}
	filter = append(filter, bson.E{Key: "id", Value: id})
	var alert models.Alert
	err = m.database.Collection("alerts").FindOneAndUpdate(
		ctx, filter, bson.M{"$set": bson.M{"status": status}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&alert)
	return alert, err
}

func (m *Mongo) ListAddresses(ctx context.Context) ([]models.Address, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	cursor, err := m.database.Collection("addresses").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	addresses := make([]models.Address, 0)
	if err := cursor.All(ctx, &addresses); err != nil {
		return nil, err
	}
	return addresses, nil
}

func (m *Mongo) CreateAddress(ctx context.Context, input models.AddressRequest) (models.Address, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.Address{}, err
	}
	companyID := filter[0].Value.(int)
	id, err := m.database.Collection("addresses").CountDocuments(ctx, filter)
	if err != nil {
		return models.Address{}, err
	}
	address := models.Address{ID: int(id) + 1, CompanyID: companyID, Value: input.Value, Label: input.Label, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	_, err = m.database.Collection("addresses").InsertOne(ctx, address)
	return address, err
}

func (m *Mongo) DeleteAddress(ctx context.Context, id int) error {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return err
	}
	filter = append(filter, bson.E{Key: "id", Value: id})
	result, err := m.database.Collection("addresses").DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}
