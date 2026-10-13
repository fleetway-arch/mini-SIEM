package api

import (
	"encoding/json"
	"net/http"

	"siem-dashboard/server/internal/repository"
)

type Server struct {
	repository repository.Repository
}

func New(data repository.Repository) *Server {
	return &Server{repository: data}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/signup", s.signup)
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.me))
	mux.HandleFunc("GET /api/team", s.requireAuth(s.listTeam))
	mux.HandleFunc("POST /api/team/invitations", s.requireAuth(s.createInvitations))
	mux.HandleFunc("PATCH /api/team/{id}", s.requireAuth(s.updateMember))
	mux.HandleFunc("DELETE /api/team/{id}", s.requireAuth(s.deleteMember))
	mux.HandleFunc("GET /api/invitations/{token}", s.getInvitation)
	mux.HandleFunc("POST /api/invitations/{token}/accept", s.acceptInvitation)
	mux.HandleFunc("GET /api/stats/summary", s.requireAuth(s.summary))
	mux.HandleFunc("GET /api/stats/timeline", s.requireAuth(s.timeline))
	mux.HandleFunc("GET /api/stats/top-ips", s.requireAuth(s.topIPs))
	mux.HandleFunc("GET /api/events", s.requireAuth(s.events))
	mux.HandleFunc("GET /api/alerts", s.requireAuth(s.listAlerts))
	mux.HandleFunc("PATCH /api/alerts/{id}", s.requireAuth(s.updateAlert))
	mux.HandleFunc("GET /api/addresses", s.requireAuth(s.listAddresses))
	mux.HandleFunc("POST /api/addresses", s.requireAuth(s.createAddress))
	mux.HandleFunc("DELETE /api/addresses/{id}", s.requireAuth(s.deleteAddress))
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
