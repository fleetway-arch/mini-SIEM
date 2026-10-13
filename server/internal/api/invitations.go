package api

import (
	"net/http"

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
	if input.FullName == "" || len(input.Password) < 8 {
		writeError(w, 400, "full name and a password of at least 8 characters are required")
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
	writeJSON(w, 200, map[string]any{"token": tokenForEmail(user.Email), "user": user})
}
