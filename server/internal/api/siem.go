package api

import (
	"net/http"
	"strconv"

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

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	events, err := s.repository.ListEvents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": events, "total": len(events)})
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
