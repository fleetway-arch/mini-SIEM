package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

type contextKey string

const (
	authenticatedEmailKey     contextKey = "authenticatedEmail"
	authenticatedCompanyIDKey contextKey = "authenticatedCompanyID"
)

var ErrMissingCompanyContext = errors.New("authenticated company is missing from context")

func WithAuthenticatedUser(ctx context.Context, email string, companyID int) context.Context {
	ctx = context.WithValue(ctx, authenticatedEmailKey, email)
	return context.WithValue(ctx, authenticatedCompanyIDKey, companyID)
}

func AuthenticatedEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(authenticatedEmailKey).(string)
	return email, ok && email != ""
}

func CompanyFilter(ctx context.Context) (bson.D, error) {
	companyID, ok := ctx.Value(authenticatedCompanyIDKey).(int)
	if !ok || companyID == 0 {
		return nil, ErrMissingCompanyContext
	}
	return bson.D{{Key: "companyId", Value: companyID}}, nil
}
