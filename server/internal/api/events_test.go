package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type eventPage struct {
	Items []struct {
		ID        int
		Timestamp string
		Source    string
		Severity  string
		IP        string
		Message   string
	}
	Total int
}

func (e *testEnv) events(token, query string) eventPage {
	e.t.Helper()
	r := e.send(request{method: "GET", path: "/api/events?" + query, token: token})
	expectStatus(e.t, r, http.StatusOK)
	var page eventPage
	r.decode(e.t, &page)
	return page
}

// TestEventsFiltersAndPagination checks that GET /api/events filters and
// paginates in MongoDB instead of returning every event.
func TestEventsFiltersAndPagination(t *testing.T) {
	env := newTestEnv(t)
	a := env.signup("Company A", "owner@a.test")
	b := env.signup("Company B", "owner@b.test")
	ctx := context.Background()

	// 30 events for A, one per hour from 2026-10-01T00:00Z.
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	var documents []any
	for i := range 30 {
		source, severity, ip, message := "sshd", "info", "10.0.0.1", "Accepted password"
		if i%3 == 0 {
			source, severity, ip, message = "apache", "warning", "203.0.113.7", "GET /admin 403"
		}
		if i == 29 {
			severity, ip, message = "critical", "203.0.113.70", "Exploit (CVE-2026-1) detected"
		}
		documents = append(documents, bson.M{
			"id": i + 1, "companyId": a.user.CompanyID, "timestamp": start.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			"source": source, "severity": severity, "ip": ip, "message": message, "rawLog": fmt.Sprintf("raw line %d", i),
		})
	}
	documents = append(documents, bson.M{"id": 100, "companyId": b.user.CompanyID, "timestamp": start.Format(time.RFC3339),
		"source": "sshd", "severity": "info", "ip": "10.0.0.1", "message": "B event", "rawLog": "raw line 0"})
	if _, err := env.db.Collection("events").InsertMany(ctx, documents); err != nil {
		t.Fatal(err)
	}

	t.Run("default page is 20 newest events", func(t *testing.T) {
		page := env.events(a.token, "")
		if page.Total != 30 || len(page.Items) != 20 || page.Items[0].ID != 30 || page.Items[19].ID != 11 {
			t.Fatalf("total=%d items=%d first=%d", page.Total, len(page.Items), page.Items[0].ID)
		}
	})
	t.Run("second page", func(t *testing.T) {
		page := env.events(a.token, "page=2&limit=20")
		if page.Total != 30 || len(page.Items) != 10 || page.Items[0].ID != 10 || page.Items[9].ID != 1 {
			t.Fatalf("total=%d items=%d", page.Total, len(page.Items))
		}
	})

	filters := []struct {
		name  string
		query url.Values
		total int
	}{
		{"source", url.Values{"source": {"apache"}}, 10},
		{"severity", url.Values{"severity": {"critical"}}, 1},
		{"ip prefix", url.Values{"ip": {"203.0.113.7"}}, 11},
		{"address is exact", url.Values{"address": {"203.0.113.7"}}, 10},
		{"address and ip prefix together", url.Values{"address": {"10.0.0.1"}, "ip": {"203."}}, 0},
		{"q in message, case-insensitive", url.Values{"q": {"accepted"}}, 19},
		{"q in rawLog", url.Values{"q": {"raw line 2"}}, 11},
		{"q is literal text, not a regex", url.Values{"q": {"(CVE-2026-1)"}}, 1},
		{"q regex metacharacters match nothing", url.Values{"q": {".*"}}, 0},
		{"from date", url.Values{"from": {"2026-10-02"}}, 6},
		{"to date includes the whole day", url.Values{"to": {"2026-10-01"}}, 24},
		{"RFC 3339 range, to exclusive", url.Values{"from": {"2026-10-01T02:00:00Z"}, "to": {"2026-10-01T05:00:00+00:00"}}, 3},
		{"filters combine", url.Values{"source": {"sshd"}, "from": {"2026-10-01T12:00:00Z"}}, 12},
	}
	for _, tt := range filters {
		t.Run(tt.name, func(t *testing.T) {
			page := env.events(a.token, tt.query.Encode())
			if page.Total != tt.total || len(page.Items) != min(tt.total, 20) {
				t.Fatalf("total=%d items=%d, want %d", page.Total, len(page.Items), tt.total)
			}
			for _, item := range page.Items {
				if item.Message == "B event" {
					t.Fatal("company B's event leaked")
				}
			}
		})
	}

	for _, query := range []string{"page=0", "page=x", "limit=0", "limit=101", "severity=fatal", "from=yesterday", "to=2026-13-01"} {
		t.Run("rejects "+query, func(t *testing.T) {
			expectStatus(t, env.send(request{method: "GET", path: "/api/events?" + query, token: a.token}), http.StatusBadRequest)
		})
	}
}
