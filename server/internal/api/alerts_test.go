package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"siem-dashboard/server/internal/models"
)

func TestListAlertsFilters(t *testing.T) {
	env := newTestEnv(t)
	a := env.signup("Company A", "owner@a.test")
	b := env.signup("Company B", "owner@b.test")
	now := time.Now().UTC().Format(time.RFC3339)
	alert := func(id, companyID int, ip, status string) bson.M {
		return bson.M{"id": id, "companyId": companyID, "createdAt": now, "rule": "brute-force", "severity": "critical",
			"ip": ip, "description": "d", "eventCount": 1, "status": status}
	}
	if _, err := env.db.Collection("alerts").InsertMany(context.Background(), []any{
		alert(1, a.user.CompanyID, "203.0.113.5", "open"),
		alert(2, a.user.CompanyID, "203.0.113.5", "resolved"),
		alert(3, a.user.CompanyID, "198.51.100.1", "open"),
		alert(4, b.user.CompanyID, "203.0.113.5", "open"),
	}); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		query string
		want  []int
	}{
		{"", []int{1, 2, 3}},
		{"?status=open", []int{1, 3}},
		{"?address=203.0.113.5", []int{1, 2}},
		{"?status=open&address=203.0.113.5", []int{1}},
		{"?address=203.0.113", nil},
	} {
		t.Run("GET /api/alerts"+tt.query, func(t *testing.T) {
			r := env.send(request{method: "GET", path: "/api/alerts" + tt.query, token: a.token})
			expectStatus(t, r, http.StatusOK)
			var alerts []models.Alert
			r.decode(t, &alerts)
			got := map[int]bool{}
			for _, alert := range alerts {
				got[alert.ID] = true
			}
			if len(alerts) != len(tt.want) {
				t.Fatalf("got %d alerts %+v, want ids %v", len(alerts), alerts, tt.want)
			}
			for _, id := range tt.want {
				if !got[id] {
					t.Fatalf("missing alert %d in %+v", id, alerts)
				}
			}
		})
	}
}
