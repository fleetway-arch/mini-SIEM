package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/repository"
)

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.repository.GetSummary(r.Context(), r.URL.Query().Get("address"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

const (
	defaultEventLimit = 20
	maxEventLimit     = 100
	maxSearchLength   = 200
)

var severities = []string{"info", "warning", "critical"}

// parseEventQuery validates the filters of GET /api/events. from and to are
// dates (YYYY-MM-DD, the whole "to" day included) or RFC 3339 timestamps
// (from inclusive, to exclusive).
func parseEventQuery(values url.Values) (models.EventQuery, error) {
	query := models.EventQuery{
		Page: 1, Limit: defaultEventLimit,
		Source: strings.TrimSpace(values.Get("source")), Severity: strings.TrimSpace(values.Get("severity")),
		IP: strings.TrimSpace(values.Get("ip")), Q: strings.TrimSpace(values.Get("q")),
		Address: strings.TrimSpace(values.Get("address")),
	}
	if raw := values.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			return query, errors.New("page must be a positive integer")
		}
		query.Page = page
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxEventLimit {
			return query, fmt.Errorf("limit must be between 1 and %d", maxEventLimit)
		}
		query.Limit = limit
	}
	if query.Severity != "" && !slices.Contains(severities, query.Severity) {
		return query, errors.New("severity must be info, warning or critical")
	}
	if len(query.Q) > maxSearchLength || len(query.IP) > maxSearchLength {
		return query, fmt.Errorf("q and ip are limited to %d characters", maxSearchLength)
	}
	var err error
	if query.From, err = parseBound(values.Get("from"), false); err != nil {
		return query, fmt.Errorf("from: %w", err)
	}
	if query.To, err = parseBound(values.Get("to"), true); err != nil {
		return query, fmt.Errorf("to: %w", err)
	}
	return query, nil
}

// parseBound returns the bound in the stored timestamp format. A date used
// as an upper bound becomes the start of the next day.
func parseBound(raw string, upper bool) (string, error) {
	if raw == "" {
		return "", nil
	}
	if day, err := time.Parse(time.DateOnly, raw); err == nil {
		if upper {
			day = day.AddDate(0, 0, 1)
		}
		return day.Format(time.RFC3339), nil
	}
	instant, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "", errors.New("expected YYYY-MM-DD or an RFC 3339 timestamp")
	}
	return instant.UTC().Format(time.RFC3339), nil
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	query, err := parseEventQuery(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := s.repository.ListEvents(r.Context(), query)
	if err != nil {
		internalError(w, "could not list events", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	points, err := s.repository.GetTimeline(r.Context(), r.URL.Query().Get("range"), r.URL.Query().Get("address"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load timeline")
		return
	}
	writeJSON(w, http.StatusOK, points)
}

func (s *Server) topIPs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	points, err := s.repository.GetTopIPs(r.Context(), limit, r.URL.Query().Get("address"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load top IPs")
		return
	}
	writeJSON(w, http.StatusOK, points)
}

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.repository.ListAlerts(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list alerts")
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) updateAlert(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid alert id")
		return
	}
	var input models.AlertUpdateRequest
	if !decodeJSON(w, r, &input) || input.Status == "" {
		return
	}
	alert, err := s.repository.UpdateAlertStatus(r.Context(), id, input.Status)
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "alert not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update alert")
		return
	}
	if alert.ID == 0 {
		writeError(w, http.StatusNotFound, "alert not found")
		return
	}
	writeJSON(w, http.StatusOK, alert)
}
