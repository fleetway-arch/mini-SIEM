package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

type contextKey string

const (
	authenticatedUserIDKey    contextKey = "authenticatedUserID"
	authenticatedCompanyIDKey contextKey = "authenticatedCompanyID"
	authenticatedRoleKey      contextKey = "authenticatedRole"
)

var ErrMissingCompanyContext = errors.New("authenticated company is missing from context")

// WithAuthenticatedUser stores the identity verified by the auth middleware.
// It is the only source CompanyFilter reads the tenant from.
func WithAuthenticatedUser(ctx context.Context, userID, companyID int, role string) context.Context {
	ctx = context.WithValue(ctx, authenticatedUserIDKey, userID)
	ctx = context.WithValue(ctx, authenticatedCompanyIDKey, companyID)
	return context.WithValue(ctx, authenticatedRoleKey, role)
}

// WithIngestCompany stores the company of a verified ingestion key. There is
// no user, so AuthenticatedUserID and AuthenticatedRole report nothing.
func WithIngestCompany(ctx context.Context, companyID int) context.Context {
	return context.WithValue(ctx, authenticatedCompanyIDKey, companyID)
}

func AuthenticatedUserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(authenticatedUserIDKey).(int)
	return id, ok && id != 0
}

func AuthenticatedRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(authenticatedRoleKey).(string)
	return role, ok && role != ""
}

func CompanyFilter(ctx context.Context) (bson.D, error) {
	companyID, ok := ctx.Value(authenticatedCompanyIDKey).(int)
	if !ok || companyID == 0 {
		return nil, ErrMissingCompanyContext
	}
	return bson.D{{Key: "companyId", Value: companyID}}, nil
}

var (
	ErrNoChanges          = errors.New("no supported field to update")
	ErrRefreshTokenReused = errors.New("refresh token reuse detected")
)
