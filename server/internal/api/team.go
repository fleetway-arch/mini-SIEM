package api

import (
	"net/http"
	"strconv"

	"siem-dashboard/server/internal/repository"
)

func (s *Server) listTeam(w http.ResponseWriter, r *http.Request) {
	team, err := s.repository.ListTeam(r.Context())
	if err != nil {
		writeError(w, 500, "could not list team")
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (s *Server) createInvitations(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Emails []string `json:"emails"`
		Role   string   `json:"role"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.repository.CreateInvitations(r.Context(), input.Emails, input.Role)
	if err != nil {
		writeError(w, 500, "could not create invitations")
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) updateMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "invalid member id")
		return
	}
	var changes map[string]string
	if !decodeJSON(w, r, &changes) {
		return
	}
	member, err := s.repository.UpdateMember(r.Context(), id, changes)
	if err != nil {
		if repository.IsNotFound(err) {
			writeError(w, 404, "member not found")
			return
		}
		writeError(w, 500, "could not update member")
		return
	}
	writeJSON(w, 200, member)
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "invalid member id")
		return
	}
	if err := s.repository.DeleteMember(r.Context(), id); err != nil {
		writeError(w, 404, "member not found")
		return
	}
	w.WriteHeader(204)
}
