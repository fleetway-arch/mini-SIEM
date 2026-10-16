package api

import (
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"siem-dashboard/server/internal/detection"
	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/repository"
	"siem-dashboard/server/internal/tokens"
)

const (
	ingestKeyPrefix     = "siem_"
	ingestKeyShown      = len(ingestKeyPrefix) + 6
	maxIngestKeyName    = 64
	maxIngestBatch      = 500
	maxMessageLength    = 1024
	maxRawLogLength     = 8192
	maxFutureClockSkew  = 5 * time.Minute
	ingestAuthFailedMsg = "missing or invalid ingestion key"
)

var sourcePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,31}$`)

func (s *Server) listIngestKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.repository.ListIngestKeys(r.Context())
	if err != nil {
		internalError(w, "could not list ingestion keys", err)
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// createIngestKey returns the plain key once; only its hash is stored.
func (s *Server) createIngestKey(w http.ResponseWriter, r *http.Request) {
	var input models.IngestKeyRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > maxIngestKeyName {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("name is required and limited to %d characters", maxIngestKeyName))
		return
	}
	plain, _, err := tokens.New()
	if err != nil {
		internalError(w, "could not create ingestion key", err)
		return
	}
	plain = ingestKeyPrefix + plain
	key, err := s.repository.CreateIngestKey(r.Context(), name, tokens.Hash(plain), plain[:ingestKeyShown])
	if err != nil {
		internalError(w, "could not create ingestion key", err)
		return
	}
	writeJSON(w, http.StatusCreated, models.CreatedIngestKey{IngestKey: key, Key: plain})
}

func (s *Server) revokeIngestKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ingestion key id")
		return
	}
	if err := s.repository.RevokeIngestKey(r.Context(), id); err != nil {
		if repository.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "ingestion key not found")
			return
		}
		internalError(w, "could not revoke ingestion key", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireIngestKey authenticates a collector by its ingestion key and scopes
// the request to the key's company.
func (s *Server) requireIngestKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || !strings.HasPrefix(raw, ingestKeyPrefix) {
			writeError(w, http.StatusUnauthorized, ingestAuthFailedMsg)
			return
		}
		key, err := s.repository.UseIngestKey(r.Context(), tokens.Hash(raw))
		if err != nil {
			if !repository.IsNotFound(err) {
				internalError(w, "could not verify ingestion key", err)
				return
			}
			writeError(w, http.StatusUnauthorized, ingestAuthFailedMsg)
			return
		}
		if !allow(w, s.ingestLimiter, strconv.Itoa(key.ID)) {
			return
		}
		next(w, r.WithContext(repository.WithIngestCompany(r.Context(), key.CompanyID)))
	}
}

// normalizeEvent validates one ingested event and converts it to the stored
// form: UTC RFC 3339 timestamp, canonical IP, message defaulting to rawLog.
func normalizeEvent(input models.IngestEvent, now time.Time) (models.Event, error) {
	event := models.Event{
		Source: strings.ToLower(strings.TrimSpace(input.Source)), Severity: strings.TrimSpace(input.Severity),
		Message: strings.TrimSpace(input.Message), RawLog: input.RawLog,
	}
	if !sourcePattern.MatchString(event.Source) {
		return event, fmt.Errorf("source is required: up to 32 lowercase letters, digits, '.', '_' or '-'")
	}
	if event.Severity == "" {
		event.Severity = "info"
	}
	if !slices.Contains(severities, event.Severity) {
		return event, fmt.Errorf("severity must be info, warning or critical")
	}
	if ip := strings.TrimSpace(input.IP); ip != "" {
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			return event, fmt.Errorf("ip %q is not a valid IP address", ip)
		}
		event.IP = addr.Unmap().String()
	}
	if event.Message == "" {
		event.Message = strings.TrimSpace(strings.SplitN(event.RawLog, "\n", 2)[0])
	}
	if event.Message == "" {
		return event, fmt.Errorf("message or rawLog is required")
	}
	if len(event.Message) > maxMessageLength || len(event.RawLog) > maxRawLogLength {
		return event, fmt.Errorf("message is limited to %d bytes and rawLog to %d bytes", maxMessageLength, maxRawLogLength)
	}
	timestamp := now
	if input.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, input.Timestamp)
		if err != nil {
			return event, fmt.Errorf("timestamp must be RFC 3339, for example 2026-10-16T08:30:00Z")
		}
		if parsed.After(now.Add(maxFutureClockSkew)) {
			return event, fmt.Errorf("timestamp is in the future")
		}
		timestamp = parsed
	}
	event.Timestamp = timestamp.UTC().Format(time.RFC3339)
	return event, nil
}

// ingestEvents stores a batch of events, all or none, then runs detection on
// it. The response lists the alerts the batch opened or added to.
func (s *Server) ingestEvents(w http.ResponseWriter, r *http.Request) {
	var input models.IngestRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if len(input.Events) == 0 || len(input.Events) > maxIngestBatch {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("events must contain between 1 and %d events", maxIngestBatch))
		return
	}
	now := time.Now().UTC()
	events := make([]models.Event, len(input.Events))
	for i, raw := range input.Events {
		event, err := normalizeEvent(raw, now)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("events[%d]: %v", i, err))
			return
		}
		events[i] = event
	}
	ctx := r.Context()
	stored, err := s.repository.InsertEvents(ctx, events)
	if err != nil {
		internalError(w, "could not store events", err)
		return
	}
	// The events are stored: a detection error is only logged, because a 5xx
	// would make the collector resend the batch and duplicate it.
	findings, err := detection.Evaluate(s.options.Detection, stored, func(ip, since string) (int, error) {
		return s.repository.CountFailedEvents(ctx, ip, since)
	})
	if err != nil {
		log.Printf("detection failed on an ingested batch: %v", err)
	}
	alerts := make([]models.Alert, 0, len(findings))
	for _, finding := range findings {
		alert, err := s.repository.RecordAlert(ctx, finding)
		if err != nil {
			log.Printf("could not record %s alert: %v", finding.Rule, err)
			continue
		}
		alerts = append(alerts, alert)
	}
	writeJSON(w, http.StatusAccepted, models.IngestResponse{Accepted: len(stored), Alerts: alerts})
}
