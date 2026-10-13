package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"siem-dashboard/server/internal/models"
)

const PasswordResetTTL = 30 * time.Minute

var ErrInvalidResetToken = errors.New("password reset token is invalid, expired or already used")

func (m *Mongo) ensurePasswordResetIndexes(ctx context.Context) error {
	_, err := m.database.Collection("password_resets").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetName("password_resets_hash_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "used", Value: 1}}, Options: options.Index().SetName("password_resets_user")},
		// Expired tokens are deleted automatically.
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetName("password_resets_expiry").SetExpireAfterSeconds(0)},
	})
	return err
}

// CreatePasswordReset stores a new reset token hash and removes any older
// unused token of the same user, so only the latest link works.
func (m *Mongo) CreatePasswordReset(ctx context.Context, user models.UserDocument, tokenHash string) error {
	collection := m.database.Collection("password_resets")
	if _, err := collection.DeleteMany(ctx, bson.M{"userId": user.ID, "used": false}); err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err := collection.InsertOne(ctx, models.PasswordResetDocument{
		TokenHash: tokenHash, UserID: user.ID, CompanyID: user.CompanyID,
		CreatedAt: now, ExpiresAt: now.Add(PasswordResetTTL),
	})
	return err
}

// ResetPassword consumes the token atomically (single use), stores the new
// hash, increments tokenVersion so existing access tokens die, and revokes
// every refresh token of the user. Any failure looks the same to callers.
func (m *Mongo) ResetPassword(ctx context.Context, tokenHash, newPasswordHash string) (models.UserDocument, error) {
	now := time.Now().UTC()
	var reset models.PasswordResetDocument
	err := m.database.Collection("password_resets").FindOneAndUpdate(ctx,
		bson.M{"tokenHash": tokenHash, "used": false, "expiresAt": bson.M{"$gt": now}},
		bson.M{"$set": bson.M{"used": true, "usedAt": now}},
	).Decode(&reset)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.UserDocument{}, ErrInvalidResetToken
	}
	if err != nil {
		return models.UserDocument{}, err
	}
	var user models.UserDocument
	err = m.database.Collection("users").FindOneAndUpdate(ctx,
		bson.M{"id": reset.UserID, "companyId": reset.CompanyID, "status": bson.M{"$ne": "revoked"}},
		bson.M{"$set": bson.M{"passwordHash": newPasswordHash}, "$inc": bson.M{"tokenVersion": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.UserDocument{}, ErrInvalidResetToken
	}
	if err != nil {
		return models.UserDocument{}, err
	}
	if err := m.RevokeUserRefreshTokens(ctx, user.ID); err != nil {
		return models.UserDocument{}, err
	}
	return user, nil
}
