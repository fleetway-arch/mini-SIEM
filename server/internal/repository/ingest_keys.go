package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/models"
)

func (m *Mongo) ensureIngestKeyIndexes(ctx context.Context) error {
	_, err := m.database.Collection("ingest_keys").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "keyHash", Value: 1}}, Options: options.Index().SetName("ingest_keys_hash_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "companyId", Value: 1}, {Key: "id", Value: 1}}, Options: options.Index().SetName("ingest_keys_company_id")},
	})
	return err
}

// CreateIngestKey stores a key hash for the authenticated company.
func (m *Mongo) CreateIngestKey(ctx context.Context, name, keyHash, prefix string) (models.IngestKey, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return models.IngestKey{}, err
	}
	userID, _ := AuthenticatedUserID(ctx)
	id, err := m.nextIDs(ctx, "ingest_keys", 1)
	if err != nil {
		return models.IngestKey{}, err
	}
	key := models.IngestKey{ID: id, Name: name, Prefix: prefix, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	_, err = m.database.Collection("ingest_keys").InsertOne(ctx, models.IngestKeyDocument{
		IngestKey: key, CompanyID: filter[0].Value.(int), KeyHash: keyHash, CreatedBy: userID,
	})
	return key, err
}

func (m *Mongo) ListIngestKeys(ctx context.Context) ([]models.IngestKey, error) {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return nil, err
	}
	filter = append(filter, bson.E{Key: "revokedAt", Value: nil})
	cursor, err := m.database.Collection("ingest_keys").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	keys := make([]models.IngestKey, 0)
	if err := cursor.All(ctx, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (m *Mongo) RevokeIngestKey(ctx context.Context, id int) error {
	filter, err := CompanyFilter(ctx)
	if err != nil {
		return err
	}
	filter = append(filter, bson.E{Key: "id", Value: id}, bson.E{Key: "revokedAt", Value: nil})
	result, err := m.database.Collection("ingest_keys").UpdateOne(ctx, filter, bson.M{"$set": bson.M{"revokedAt": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UseIngestKey finds an active key by hash and records its use. It runs
// before any company is known, like FindUserByEmail.
func (m *Mongo) UseIngestKey(ctx context.Context, keyHash string) (models.IngestKeyDocument, error) {
	var key models.IngestKeyDocument
	err := m.database.Collection("ingest_keys").FindOneAndUpdate(ctx,
		bson.M{"keyHash": keyHash, "revokedAt": nil},
		bson.M{"$set": bson.M{"lastUsedAt": time.Now().UTC().Format(time.RFC3339)}},
	).Decode(&key)
	return key, err
}
