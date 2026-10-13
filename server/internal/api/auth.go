package api

import (
	"fmt"
	"net/http"
	"strings"

	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/repository"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input models.LoginRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	user, err := s.repository.FindUserByEmail(r.Context(), input.Email)
	if err != nil || password.Verify(input.Password, user.PasswordHash) != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tokenForEmail(user.Email), "user": user.User})
}
func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var input models.SignupRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Admin.Email = strings.ToLower(strings.TrimSpace(input.Admin.Email))
	input.Admin.FullName = strings.TrimSpace(input.Admin.FullName)
	input.Company.Name = strings.TrimSpace(input.Company.Name)
	if input.Company.Name == "" || input.Admin.FullName == "" || input.Admin.Email == "" || input.Admin.Password == "" {
		writeError(w, http.StatusBadRequest, "company name, full name, email, and password are required")
		return
	}

	if len(input.Admin.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if _, err := s.repository.FindUserByEmail(r.Context(), input.Admin.Email); err == nil {
		writeError(w, http.StatusConflict, "email already used")
		return
	}
	user, err := s.repository.CreateAccount(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create account")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": tokenForEmail(user.Email), "user": user})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	email, _ := repository.AuthenticatedEmail(r.Context())
	user, err := s.repository.FindUserByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user.User)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer demo-token-"
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, prefix) {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		email := strings.TrimPrefix(token, prefix)
		user, err := s.repository.FindUserByEmail(r.Context(), email)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid session")
			return
		}
		next(w, r.WithContext(repository.WithAuthenticatedUser(r.Context(), email, user.CompanyID)))
	}
}

func tokenForEmail(email string) string {
	return fmt.Sprintf("demo-token-%s", strings.ToLower(email))
}
