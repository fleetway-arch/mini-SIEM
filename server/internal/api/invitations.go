package api

import (
	"net/http"
	"strings"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/repository"
)

func (s *Server) getInvitation(w http.ResponseWriter, r *http.Request) {
	invitation, err := s.repository.GetInvitation(r.Context(), r.PathValue("token"))
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "invitation not found")
			return
		}
		writeError(w, 500, "could not load invitation")
		return
	}
	writeJSON(w, 200, invitation)
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FullName string `json:"fullName"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if input.FullName == "" {
		writeError(w, 400, "full name is required")
		return
	}
	if err := password.ValidateStrength(input.Password); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	user, err := s.repository.AcceptInvitation(r.Context(), r.PathValue("token"), input.FullName, input.Password)
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "invitation not found")
			return
		}
		writeError(w, 409, "could not accept invitation")
		return
	}
	s.startSession(w, r, models.UserDocument{User: user, Status: "active"}, http.StatusOK)
}
