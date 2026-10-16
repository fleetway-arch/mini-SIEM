package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"siem-dashboard/server/internal/models"
)

func (e *testEnv) createIngestKey(by session, name string) models.CreatedIngestKey {
	e.t.Helper()
	r := e.send(request{method: "POST", path: "/api/ingest-keys", token: by.token, body: map[string]string{"name": name}})
	expectStatus(e.t, r, http.StatusCreated)
	var key models.CreatedIngestKey
	r.decode(e.t, &key)
	return key
}

func (e *testEnv) ingest(key string, events ...map[string]string) result {
	e.t.Helper()
	return e.send(request{method: "POST", path: "/api/ingest/events", token: key, body: map[string]any{"events": events}})
}

func (e *testEnv) ingestOK(key string, events ...map[string]string) models.IngestResponse {
	e.t.Helper()
	r := e.ingest(key, events...)
	expectStatus(e.t, r, http.StatusAccepted)
	var response models.IngestResponse
	r.decode(e.t, &response)
	return response
}

func failedLogin(ip string, at time.Time) map[string]string {
	return map[string]string{"timestamp": at.Format(time.RFC3339), "source": "sshd", "severity": "warning", "ip": ip, "message": "Failed password for root"}
}

func TestIngestKeys(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Company A", "owner@a.test")
	viewer := env.addMember(owner, "viewer@a.test", "viewer")

	expectStatus(t, env.send(request{method: "POST", path: "/api/ingest-keys", token: viewer.token, body: map[string]string{"name": "x"}}), http.StatusForbidden)
	expectStatus(t, env.send(request{method: "GET", path: "/api/ingest-keys", token: viewer.token}), http.StatusForbidden)
	expectStatus(t, env.send(request{method: "POST", path: "/api/ingest-keys", token: owner.token, body: map[string]string{"name": "  "}}), http.StatusBadRequest)

	key := env.createIngestKey(owner, "Web server")
	if !strings.HasPrefix(key.Key, "siem_") || len(key.Key) < 40 || !strings.HasPrefix(key.Key, key.Prefix) || key.Prefix == key.Key {
		t.Fatalf("unexpected key %+v", key)
	}
	var stored bson.M
	if err := env.db.Collection("ingest_keys").FindOne(context.Background(), bson.M{"id": key.ID}).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(stored) == "" || strings.Contains(fmt.Sprint(stored), key.Key) {
		t.Fatal("the plain key is stored")
	}

	list := env.send(request{method: "GET", path: "/api/ingest-keys", token: owner.token})
	expectStatus(t, list, http.StatusOK)
	if strings.Contains(string(list.body), key.Key) || strings.Contains(string(list.body), "keyHash") || !strings.Contains(string(list.body), "Web server") {
		t.Fatalf("list exposes the key or misses it: %s", list.body)
	}

	event := map[string]string{"source": "sshd", "message": "hello"}
	expectStatus(t, env.ingest(key.Key, event), http.StatusAccepted)
	for name, token := range map[string]string{
		"no key": "", "unknown key": "siem_unknown", "access token instead of key": owner.token,
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			expectStatus(t, env.ingest(token, event), http.StatusUnauthorized)
		})
	}

	expectStatus(t, env.send(request{method: "DELETE", path: "/api/ingest-keys/" + itoa(key.ID), token: owner.token}), http.StatusNoContent)
	expectStatus(t, env.ingest(key.Key, event), http.StatusUnauthorized)
	expectStatus(t, env.send(request{method: "DELETE", path: "/api/ingest-keys/" + itoa(key.ID), token: owner.token}), http.StatusNotFound)

	other := env.signup("Company B", "owner@b.test")
	otherKey := env.createIngestKey(other, "B key")
	expectStatus(t, env.send(request{method: "DELETE", path: "/api/ingest-keys/" + itoa(otherKey.ID), token: owner.token}), http.StatusNotFound)
	expectStatus(t, env.ingest(otherKey.Key, event), http.StatusAccepted)
}

func TestIngestStoresEventsForTheKeysCompany(t *testing.T) {
	env := newTestEnv(t)
	a := env.signup("Company A", "owner@a.test")
	b := env.signup("Company B", "owner@b.test")
	key := env.createIngestKey(a, "collector")

	response := env.ingestOK(key.Key,
		map[string]string{"timestamp": "2026-10-16T10:00:00+02:00", "source": "Apache", "severity": "info", "ip": "::ffff:198.51.100.4", "message": "GET / 200"},
		map[string]string{"source": "dns", "rawLog": "query example.com\nsecond line"},
	)
	if response.Accepted != 2 || len(response.Alerts) != 0 {
		t.Fatalf("response = %+v", response)
	}

	page := env.events(a.token, "")
	if page.Total != 2 {
		t.Fatalf("company A sees %d events, want 2", page.Total)
	}
	byMessage := map[string]int{}
	for i, item := range page.Items {
		byMessage[item.Message] = i
	}
	web := page.Items[byMessage["GET / 200"]]
	if web.Timestamp != "2026-10-16T08:00:00Z" || web.Source != "apache" || web.IP != "198.51.100.4" {
		t.Fatalf("event not normalized: %+v", web)
	}
	dns, ok := byMessage["query example.com"]
	if !ok || page.Items[dns].Severity != "info" || page.Items[dns].ID == web.ID {
		t.Fatalf("rawLog event not stored as expected: %+v", page.Items)
	}
	if total := env.events(b.token, "").Total; total != 0 {
		t.Fatalf("company B sees %d of A's events", total)
	}

	invalid := []map[string]string{
		{"message": "no source"},
		{"source": "sshd"},
		{"source": "sshd", "message": "m", "severity": "fatal"},
		{"source": "sshd", "message": "m", "ip": "999.1.1.1"},
		{"source": "sshd", "message": "m", "timestamp": "16/10/2026"},
		{"source": "sshd", "message": "m", "timestamp": time.Now().Add(time.Hour).Format(time.RFC3339)},
		{"source": "ssh d", "message": "m"},
		{"source": "sshd", "message": strings.Repeat("x", 1025)},
	}
	for i, bad := range invalid {
		t.Run(fmt.Sprintf("rejects invalid event %d", i), func(t *testing.T) {
			// The valid first event must not be stored either.
			r := env.ingest(key.Key, map[string]string{"source": "sshd", "message": "valid"}, bad)
			expectStatus(t, r, http.StatusBadRequest)
			if !strings.Contains(string(r.body), "events[1]") {
				t.Fatalf("error does not point at the event: %s", r.body)
			}
		})
	}
	expectStatus(t, env.send(request{method: "POST", path: "/api/ingest/events", token: key.Key, body: map[string]any{"events": []any{}}}), http.StatusBadRequest)
	if total := env.events(a.token, "").Total; total != 2 {
		t.Fatalf("rejected batches stored events: total %d", total)
	}
}

func TestIngestDetection(t *testing.T) {
	env := newTestEnv(t) // brute-force threshold 5 within 5 minutes
	a := env.signup("Company A", "owner@a.test")
	key := env.createIngestKey(a, "collector")
	attacker := "203.0.113.9"
	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	alerts := func(status string) []models.Alert {
		var list []models.Alert
		env.send(request{method: "GET", path: "/api/alerts?status=" + status, token: a.token}).decode(t, &list)
		return list
	}

	// Four failures, plus older failures outside the window: no alert.
	batch := []map[string]string{failedLogin(attacker, start.Add(-10*time.Minute)), failedLogin(attacker, start.Add(-9*time.Minute))}
	for i := range 4 {
		batch = append(batch, failedLogin(attacker, start.Add(time.Duration(i)*time.Minute)))
	}
	if response := env.ingestOK(key.Key, batch...); len(response.Alerts) != 0 {
		t.Fatalf("alert below threshold: %+v", response.Alerts)
	}

	// The fifth failure within 5 minutes opens one alert.
	response := env.ingestOK(key.Key, failedLogin(attacker, start.Add(4*time.Minute)))
	if len(response.Alerts) != 1 {
		t.Fatalf("alerts = %+v, want one brute-force", response.Alerts)
	}
	alert := response.Alerts[0]
	if alert.Rule != "brute-force" || alert.IP != attacker || alert.EventCount != 5 || alert.Status != "open" || alert.Severity != "critical" {
		t.Fatalf("alert = %+v", alert)
	}

	// More failures add to the same alert, even once acknowledged.
	expectStatus(t, env.send(request{method: "PATCH", path: "/api/alerts/" + itoa(alert.ID), token: a.token, body: map[string]string{"status": "acknowledged"}}), http.StatusOK)
	response = env.ingestOK(key.Key, failedLogin(attacker, start.Add(5*time.Minute)), failedLogin(attacker, start.Add(5*time.Minute+time.Second)))
	if len(response.Alerts) != 1 || response.Alerts[0].ID != alert.ID || response.Alerts[0].EventCount != 7 {
		t.Fatalf("alert not updated in place: %+v", response.Alerts)
	}
	if list := alerts(""); len(list) != 1 || list[0].Status != "acknowledged" {
		t.Fatalf("alerts = %+v", list)
	}

	// Once resolved, a new attack opens a new alert.
	expectStatus(t, env.send(request{method: "PATCH", path: "/api/alerts/" + itoa(alert.ID), token: a.token, body: map[string]string{"status": "resolved"}}), http.StatusOK)
	response = env.ingestOK(key.Key, failedLogin(attacker, start.Add(6*time.Minute)))
	if len(response.Alerts) != 1 || response.Alerts[0].ID == alert.ID {
		t.Fatalf("no new alert after resolve: %+v", response.Alerts)
	}

	// A critical event opens a critical-event alert; summary counts it.
	response = env.ingestOK(key.Key, map[string]string{"source": "vsftpd", "severity": "critical", "ip": "198.51.100.20", "message": "Backdoor login"})
	if len(response.Alerts) != 1 || response.Alerts[0].Rule != "critical-event" || response.Alerts[0].IP != "198.51.100.20" {
		t.Fatalf("alerts = %+v", response.Alerts)
	}
	var summary models.StatsSummary
	env.send(request{method: "GET", path: "/api/stats/summary", token: a.token}).decode(t, &summary)
	if summary.OpenAlerts != 2 || summary.TotalEvents != 11 {
		t.Fatalf("summary = %+v", summary)
	}

	// Another company's failures from the same IP do not count for A.
	b := env.signup("Company B", "owner@b.test")
	bKey := env.createIngestKey(b, "collector")
	if response := env.ingestOK(bKey.Key, failedLogin(attacker, start.Add(6*time.Minute))); len(response.Alerts) != 0 {
		t.Fatalf("B got an alert from A's events: %+v", response.Alerts)
	}
}

func TestIngestRateLimitPerKey(t *testing.T) {
	env := newTestEnv(t, func(o *Options) { o.RateLimitIngestPerKey = 2 })
	a := env.signup("Company A", "owner@a.test")
	first := env.createIngestKey(a, "first")
	second := env.createIngestKey(a, "second")
	event := map[string]string{"source": "sshd", "message": "hello"}

	expectStatus(t, env.ingest(first.Key, event), http.StatusAccepted)
	expectStatus(t, env.ingest(first.Key, event), http.StatusAccepted)
	limited := env.ingest(first.Key, event)
	expectStatus(t, limited, http.StatusTooManyRequests)
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	expectStatus(t, env.ingest(second.Key, event), http.StatusAccepted)
}
