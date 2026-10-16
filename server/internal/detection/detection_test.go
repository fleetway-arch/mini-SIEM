package detection

import (
	"errors"
	"testing"
	"time"

	"siem-dashboard/server/internal/models"
)

var config = Config{BruteForceThreshold: 3, BruteForceWindow: 5 * time.Minute}

func event(ts, severity, ip, message string) models.Event {
	return models.Event{Timestamp: ts, Source: "sshd", Severity: severity, IP: ip, Message: message}
}

func TestIsFailed(t *testing.T) {
	for message, want := range map[string]bool{
		"Failed password for root":   true,
		"PAM authentication FAILURE": true,
		"authentication denied":      true,
		"Accepted publickey for bob": false,
	} {
		if got := IsFailed(models.Event{Message: message}); got != want {
			t.Errorf("IsFailed(%q) = %v, want %v", message, got, want)
		}
	}
	if !IsFailed(models.Event{Message: "sshd", RawLog: "login failed"}) {
		t.Error("IsFailed ignores rawLog")
	}
}

func TestEvaluateBruteForce(t *testing.T) {
	batch := []models.Event{
		event("2026-10-16T10:00:00Z", "warning", "203.0.113.5", "Failed password for root"),
		event("2026-10-16T10:02:00Z", "warning", "203.0.113.5", "Failed password for admin"),
		event("2026-10-16T10:01:00Z", "info", "203.0.113.5", "Accepted password for bob"),
		event("2026-10-16T10:01:00Z", "warning", "198.51.100.7", "Failed password for root"),
	}
	var calls []string
	counts := map[string]int{"203.0.113.5": 3, "198.51.100.7": 1}
	findings, err := Evaluate(config, batch, func(ip, since string) (int, error) {
		calls = append(calls, ip+"@"+since)
		return counts[ip], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The window ends at the latest failed event of each IP.
	wantCalls := []string{"198.51.100.7@2026-10-16T09:56:00Z", "203.0.113.5@2026-10-16T09:57:00Z"}
	if len(calls) != 2 || calls[0] != wantCalls[0] || calls[1] != wantCalls[1] {
		t.Fatalf("counter calls = %v, want %v", calls, wantCalls)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want one brute-force", findings)
	}
	f := findings[0]
	if f.Rule != RuleBruteForce || f.IP != "203.0.113.5" || f.EventCount != 3 || f.NewEvents != 2 || f.LastSeen != "2026-10-16T10:02:00Z" {
		t.Fatalf("finding = %+v", f)
	}
}

func TestEvaluateIgnoresFailuresWithoutIP(t *testing.T) {
	findings, err := Evaluate(config, []models.Event{event("2026-10-16T10:00:00Z", "warning", "", "Failed password")},
		func(string, string) (int, error) { t.Fatal("counter called for an event without IP"); return 0, nil })
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings = %+v, err = %v", findings, err)
	}
}

func TestEvaluateCriticalEvents(t *testing.T) {
	batch := []models.Event{
		event("2026-10-16T10:00:00Z", "critical", "203.0.113.5", "Root shell spawned"),
		event("2026-10-16T10:03:00Z", "critical", "203.0.113.5", "Root shell spawned again"),
		event("2026-10-16T10:01:00Z", "critical", "", "Disk wiped"),
		event("2026-10-16T10:01:00Z", "info", "203.0.113.5", "Session opened"),
	}
	findings, err := Evaluate(config, batch, func(string, string) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %+v, want one per IP", findings)
	}
	if f := findings[0]; f.Rule != RuleCriticalEvent || f.IP != "203.0.113.5" || f.EventCount != 2 || f.LastSeen != "2026-10-16T10:03:00Z" ||
		f.Description != "Critical sshd event from 203.0.113.5: Root shell spawned" {
		t.Fatalf("first finding = %+v", f)
	}
	if f := findings[1]; f.IP != "" || f.Description != "Critical sshd event from an unknown source: Disk wiped" {
		t.Fatalf("second finding = %+v", f)
	}
}

func TestEvaluateCounterError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Evaluate(config, []models.Event{event("2026-10-16T10:00:00Z", "warning", "203.0.113.5", "Failed password")},
		func(string, string) (int, error) { return 0, boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
