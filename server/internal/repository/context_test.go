package repository

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestCompanyFilterUsesAuthenticatedCompany(t *testing.T) {
	ctx := WithAuthenticatedUser(context.Background(), "a@example.com", 101)

	filter, err := CompanyFilter(ctx)
	if err != nil {
		t.Fatalf("CompanyFilter() returned error: %v", err)
	}

	expected := bson.D{{Key: "companyId", Value: 101}}
	if len(filter) != 1 || filter[0] != expected[0] {
		t.Fatalf("CompanyFilter() = %#v, want %#v", filter, expected)
	}
}

func TestCompanyFilterRejectsMissingAuthentication(t *testing.T) {
	if _, err := CompanyFilter(context.Background()); err != ErrMissingCompanyContext {
		t.Fatalf("CompanyFilter() error = %v, want %v", err, ErrMissingCompanyContext)
	}
}
