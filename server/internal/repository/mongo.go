package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
)

const refreshReuseGrace = 10 * time.Second

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

// flattenLegacyUsers fixes user documents written before UserDocument was
// inlined, where every field except the hash was nested under "user".
func (m *Mongo) flattenLegacyUsers(ctx context.Context) error {
	result, err := m.database.Collection("users").UpdateMany(ctx,
		bson.M{"user": bson.M{"$type": "object"}},
		mongo.Pipeline{
			{{Key: "$replaceWith", Value: bson.D{{Key: "$mergeObjects", Value: bson.A{"$$ROOT", "$user"}}}}},
			{{Key: "$unset", Value: "user"}},
		},
	)
	if err != nil {
		return fmt.Errorf("flatten legacy user documents: %w", err)
	}
	if result.ModifiedCount > 0 {
		log.Printf("migrated %d legacy user document(s)", result.ModifiedCount)
	}
	return nil
}

func (m *Mongo) EnsureIndexes(ctx context.Context) error {
	if err := m.flattenLegacyUsers(ctx); err != nil {
		return err
	}
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
	if _, err := m.database.Collection("events").Indexes().CreateMany(ctx, indexes); err != nil {
		return err
	}
	if _, err := m.database.Collection("users").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetName("users_email_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetName("users_id_unique").SetUnique(true)},
	}); err != nil {
		return err
	}
	invitations := m.database.Collection("invitations")
	// The first schema stored raw tokens with a unique index on "token"; new
	// documents have no such field, so that index would reject them.
	if _, err := invitations.Indexes().DropOne(ctx, "invitations_token_unique"); err != nil && !isIndexNotFound(err) {
		return err
	}
	if _, err := invitations.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetName("invitations_token_hash_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "id", Value: 1}}, Options: options.Index().SetName("invitations_company_id")},
		{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "email", Value: 1}}, Options: options.Index().SetName("invitations_company_email")},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetName("invitations_expiry").SetExpireAfterSeconds(0)},
	}); err != nil {
		return err
	}
	_, err := m.database.Collection("refresh_tokens").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetName("refresh_tokens_hash_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetName("refresh_tokens_user")},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetName("refresh_tokens_expiry").SetExpireAfterSeconds(0)},
	})
	return err
}

func (m *Mongo) FindUserByEmail(ctx context.Context, email string) (models.UserDocument, error) {
	var user models.UserDocument
	err := m.database.Collection("users").FindOne(ctx, bson.M{"email": strings.ToLower(email)}).Decode(&user)
	return user, err
}

// FindUserByID is used by the auth middleware; the company is checked
// against the token claims so a token can never cross tenants.
func (m *Mongo) FindUserByID(ctx context.Context, id, companyID int) (models.UserDocument, error) {
	var user models.UserDocument
	err := m.database.Collection("users").FindOne(ctx, bson.M{"id": id, "companyId": companyID}).Decode(&user)
	return user, err
}

// GetMember loads a member of the authenticated company.
func (m *Mongo) GetMember(ctx context.Context, id int) (models.UserDocument, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.UserDocument{}, err
	}
	filter = append(filter, bson.E{Key: "id", Value: id})
	var user models.UserDocument
	err = m.database.Collection("users").FindOne(ctx, filter).Decode(&user)
	return user, err
}

func memberStatus(user models.UserDocument) string {
	if user.Status == "" {
		return "active"
	}
	return user.Status
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
		result = append(result, models.TeamMember{ID: user.ID, Email: user.Email, FullName: user.FullName, Role: user.Role, Status: memberStatus(user)})
	}
	pending := pendingInvitationFilter(time.Now().UTC())
	pending["companyId"] = filter[0].Value
	invitationCursor, err := m.database.Collection("invitations").Find(ctx, pending, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer invitationCursor.Close(ctx)
	var invitations []models.InvitationDocument
	if err := invitationCursor.All(ctx, &invitations); err != nil {
		return nil, err
	}
	for _, invitation := range invitations {
		result = append(result, invitationMember(invitation))
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
	if len(update) == 0 {
		return models.TeamMember{}, ErrNoChanges
	}
	// A role or status change invalidates the member's existing sessions.
	var user models.UserDocument
	err = m.database.Collection("users").FindOneAndUpdate(ctx, filter,
		bson.M{"$set": update, "$inc": bson.M{"tokenVersion": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err != nil {
		return models.TeamMember{}, err
	}
	if err := m.RevokeUserRefreshTokens(ctx, user.ID); err != nil {
		return models.TeamMember{}, err
	}
	return models.TeamMember{ID: user.ID, Email: user.Email, FullName: user.FullName, Role: user.Role, Status: memberStatus(user)}, nil
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
	return m.RevokeUserRefreshTokens(ctx, id)
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
		Status:       "active",
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

func (m *Mongo) CreateRefreshToken(ctx context.Context, token models.RefreshTokenDocument) error {
	_, err := m.database.Collection("refresh_tokens").InsertOne(ctx, token)
	return err
}

// ConsumeRefreshToken atomically revokes an active refresh token and returns
// it, so each token can be rotated exactly once. Presenting a token that was
// already rotated means it leaked: every session of that user is revoked.
func (m *Mongo) ConsumeRefreshToken(ctx context.Context, tokenHash string) (models.RefreshTokenDocument, error) {
	now := time.Now().UTC()
	collection := m.database.Collection("refresh_tokens")
	var token models.RefreshTokenDocument
	err := collection.FindOneAndUpdate(ctx,
		bson.M{"tokenHash": tokenHash, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": now}},
	).Decode(&token)
	if errors.Is(err, mongo.ErrNoDocuments) {
		var revoked models.RefreshTokenDocument
		if findErr := collection.FindOne(ctx, bson.M{"tokenHash": tokenHash}).Decode(&revoked); findErr == nil {
			// Two tabs refreshing at the same moment is not an attack: reject
			// the slower one without revoking the whole session family.
			if revoked.RevokedAt != nil && now.Sub(*revoked.RevokedAt) < refreshReuseGrace {
				return models.RefreshTokenDocument{}, mongo.ErrNoDocuments
			}
			if revokeErr := m.RevokeUserRefreshTokens(ctx, revoked.UserID); revokeErr != nil {
				return models.RefreshTokenDocument{}, revokeErr
			}
			return models.RefreshTokenDocument{}, ErrRefreshTokenReused
		}
		return models.RefreshTokenDocument{}, mongo.ErrNoDocuments
	}
	if err != nil {
		return models.RefreshTokenDocument{}, err
	}
	if !token.ExpiresAt.After(now) {
		return models.RefreshTokenDocument{}, mongo.ErrNoDocuments
	}
	return token, nil
}

func (m *Mongo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := m.database.Collection("refresh_tokens").UpdateOne(ctx,
		bson.M{"tokenHash": tokenHash, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": time.Now().UTC()}},
	)
	return err
}

func (m *Mongo) RevokeUserRefreshTokens(ctx context.Context, userID int) error {
	_, err := m.database.Collection("refresh_tokens").UpdateMany(ctx,
		bson.M{"userId": userID, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": time.Now().UTC()}},
	)
	return err
}

// isIndexNotFound covers both a missing index (27) and a collection that does
// not exist yet (26, fresh database).
func isIndexNotFound(err error) bool {
	var commandErr mongo.CommandError
	return errors.As(err, &commandErr) && (commandErr.Code == 26 || commandErr.Code == 27)
}

func IsNotFound(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}
