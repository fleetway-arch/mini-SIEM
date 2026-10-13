package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func itoa(value int) string { return strconv.Itoa(value) }

// TestTenantIsolation proves that a user of company A cannot read, update or
// delete company B's data through the real JWT-authenticated routes.
func TestTenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	a := env.signup("Company A", "owner@a.test")
	b := env.signup("Company B", "owner@b.test")
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// Data owned by B.
	if _, err := env.db.Collection("alerts").InsertOne(ctx, bson.M{
		"id": 900, "companyId": b.user.CompanyID, "createdAt": now, "rule": "brute-force",
		"severity": "critical", "ip": "203.0.113.50", "description": "B only", "eventCount": 12, "status": "open",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.db.Collection("events").InsertOne(ctx, bson.M{
		"id": 901, "companyId": b.user.CompanyID, "timestamp": now, "source": "sshd",
		"severity": "warning", "ip": "203.0.113.50", "message": "login failed for root", "rawLog": "B secret log",
	}); err != nil {
		t.Fatal(err)
	}
	var address struct{ ID int }
	created := env.send(request{method: "POST", path: "/api/addresses", token: b.token, body: map[string]string{"value": "203.0.113.50", "label": "B server"}})
	expectStatus(t, created, http.StatusCreated)
	created.decode(t, &address)
	bViewer := env.addMember(b, "viewer@b.test", "viewer")
	env.invite(b, "pending@b.test", "viewer")
	var bTeam []struct {
		ID     int
		Status string
	}
	env.send(request{method: "GET", path: "/api/team", token: b.token}).decode(t, &bTeam)
	pendingID := 0
	for _, member := range bTeam {
		if member.Status == "invited" {
			pendingID = member.ID
		}
	}

	reads := []string{
		"/api/alerts", "/api/events", "/api/addresses", "/api/team",
		"/api/stats/summary", "/api/stats/top-ips", "/api/stats/timeline?range=7d",
	}
	for _, path := range reads {
		t.Run("A cannot read B via GET "+path, func(t *testing.T) {
			r := env.send(request{method: "GET", path: path, token: a.token})
			expectStatus(t, r, http.StatusOK)
			body := string(r.body)
			for _, leak := range []string{"203.0.113.50", "B only", "B secret log", "B server", "@b.test"} {
				if strings.Contains(body, leak) {
					t.Fatalf("response leaks %q from company B: %s", leak, body)
				}
			}
			if path == "/api/stats/summary" {
				var summary struct{ TotalEvents, OpenAlerts int }
				_ = json.Unmarshal(r.body, &summary)
				if summary.TotalEvents != 0 || summary.OpenAlerts != 0 {
					t.Fatalf("summary counts B's data: %+v", summary)
				}
			}
		})
	}

	writes := []struct {
		name string
		req  request
	}{
		{"update B's alert", request{method: "PATCH", path: "/api/alerts/900", body: map[string]string{"status": "resolved"}}},
		{"delete B's address", request{method: "DELETE", path: "/api/addresses/" + itoa(address.ID)}},
		{"change B's member role", request{method: "PATCH", path: "/api/team/" + itoa(bViewer.user.ID), body: map[string]string{"role": "admin"}}},
		{"revoke B's owner", request{method: "PATCH", path: "/api/team/" + itoa(b.user.ID), body: map[string]string{"status": "revoked"}}},
		{"delete B's member", request{method: "DELETE", path: "/api/team/" + itoa(bViewer.user.ID)}},
		{"resend B's invitation", request{method: "POST", path: "/api/team/" + itoa(pendingID) + "/resend"}},
		{"cancel B's invitation", request{method: "DELETE", path: "/api/team/" + itoa(pendingID)}},
	}
	for _, tt := range writes {
		t.Run("A cannot "+tt.name, func(t *testing.T) {
			tt.req.token = a.token
			expectStatus(t, env.send(tt.req), http.StatusNotFound)
		})
	}

	// Nothing of B changed.
	var alert struct{ Status string }
	if err := env.db.Collection("alerts").FindOne(ctx, bson.M{"id": 900}).Decode(&alert); err != nil || alert.Status != "open" {
		t.Fatalf("B's alert was modified: %+v (%v)", alert, err)
	}
	for collection, filter := range map[string]bson.M{
		"addresses":   {"companyId": b.user.CompanyID},
		"invitations": {"companyId": b.user.CompanyID, "used": false},
		"users":       {"companyId": b.user.CompanyID, "role": "viewer", "status": "active"},
	} {
		if count, _ := env.db.Collection(collection).CountDocuments(ctx, filter); count != 1 {
			t.Errorf("B's %s changed: %d matching documents, want 1", collection, count)
		}
	}
	expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: bViewer.token}), http.StatusOK)
}
