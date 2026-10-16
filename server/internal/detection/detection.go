// Package detection turns freshly ingested events into alert findings.
// It holds no state: the repository counts past events and records alerts.
package detection

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"siem-dashboard/server/internal/models"
)

const (
	RuleBruteForce    = "brute-force"
	RuleCriticalEvent = "critical-event"
)

// FailedPattern marks an event as a failed authentication. MongoDB
// aggregations use the same pattern, so stats and detection agree.
const FailedPattern = "failed|failure|authentication denied|login failed"

var failedRegexp = regexp.MustCompile("(?i)" + FailedPattern)

func IsFailed(event models.Event) bool {
	return failedRegexp.MatchString(event.Message) || failedRegexp.MatchString(event.RawLog)
}

type Config struct {
	// BruteForceThreshold failed events from one IP within BruteForceWindow
	// raise a brute-force alert.
	BruteForceThreshold int
	BruteForceWindow    time.Duration
}

// Finding is an alert to create, or to add to when the same rule already has
// an unresolved alert for the same IP.
type Finding struct {
	Rule        string
	Severity    string
	IP          string
	Description string
	// EventCount is the count for a new alert; NewEvents is what this batch
	// adds to an existing one.
	EventCount int
	NewEvents  int
	LastSeen   string
}

// FailedCounter returns how many failed events of the company came from ip
// with a timestamp at or after since (RFC 3339, UTC).
type FailedCounter func(ip, since string) (int, error)

type group struct {
	events []models.Event
	latest string
}

// groupByIP keeps the order of first appearance stable for deterministic
// output; timestamps are normalized RFC 3339 UTC, so they compare as strings.
func groupByIP(events []models.Event, keep func(models.Event) bool) ([]string, map[string]*group) {
	var order []string
	groups := map[string]*group{}
	for _, event := range events {
		if !keep(event) {
			continue
		}
		g, ok := groups[event.IP]
		if !ok {
			g = &group{}
			groups[event.IP] = g
			order = append(order, event.IP)
		}
		g.events = append(g.events, event)
		if event.Timestamp > g.latest {
			g.latest = event.Timestamp
		}
	}
	return order, groups
}

// Evaluate applies the rules to a batch that is already stored, so count
// includes the batch itself.
func Evaluate(config Config, batch []models.Event, count FailedCounter) ([]Finding, error) {
	var findings []Finding

	order, critical := groupByIP(batch, func(e models.Event) bool { return e.Severity == "critical" })
	for _, ip := range order {
		g := critical[ip]
		source := ip
		if source == "" {
			source = "an unknown source"
		}
		findings = append(findings, Finding{
			Rule: RuleCriticalEvent, Severity: "critical", IP: ip,
			Description: fmt.Sprintf("Critical %s event from %s: %s", g.events[0].Source, source, g.events[0].Message),
			EventCount:  len(g.events), NewEvents: len(g.events), LastSeen: g.latest,
		})
	}

	order, failed := groupByIP(batch, func(e models.Event) bool { return e.IP != "" && IsFailed(e) })
	sort.Strings(order)
	for _, ip := range order {
		g := failed[ip]
		latest, err := time.Parse(time.RFC3339, g.latest)
		if err != nil {
			return nil, fmt.Errorf("event timestamp %q: %w", g.latest, err)
		}
		since := latest.Add(-config.BruteForceWindow).UTC().Format(time.RFC3339)
		total, err := count(ip, since)
		if err != nil {
			return nil, err
		}
		if total < config.BruteForceThreshold {
			continue
		}
		findings = append(findings, Finding{
			Rule: RuleBruteForce, Severity: "critical", IP: ip,
			Description: fmt.Sprintf("%d failed authentications from %s within %s", total, ip, config.BruteForceWindow),
			EventCount:  total, NewEvents: len(g.events), LastSeen: g.latest,
		})
	}
	return findings, nil
}
