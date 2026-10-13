package api

import (
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"siem-dashboard/server/internal/middleware"
	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/repository"
	"siem-dashboard/server/internal/tokens"
)

const (
	refreshCookieName = "siem_refresh"
	refreshCookiePath = "/api/auth"

	roleOwner  = "owner"
	roleAdmin  = "admin"
	roleViewer = "viewer"

	invalidCredentials = "invalid credentials"
	invalidSession     = "invalid or expired session"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(w, r, "login") {
		return
	}
	var input models.LoginRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if !s.allowEmail(w, "login", input.Email) {
		return
	}
	user, err := s.repository.FindUserByEmail(r.Context(), input.Email)
	if err != nil {
		if !repository.IsNotFound(err) {
			internalError(w, "could not log in", err)
			return
		}
		// Hash anyway so an unknown email takes as long as a wrong password.
		password.VerifyDummy(input.Password)
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}
	if err := password.Verify(input.Password, user.PasswordHash); err != nil {
		if !errors.Is(err, password.ErrMismatch) {
			log.Printf("login: stored hash for user %d is unreadable: %v", user.ID, err)
		}
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}
	if !user.IsActive() {
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}
	s.startSession(w, r, user, http.StatusOK)
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(w, r, "signup") {
		return
	}
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
	if err := password.ValidateStrength(input.Admin.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.allowEmail(w, "signup", input.Admin.Email) {
		return
	}
	// Same message whether the email or the company name is taken.
	if _, err := s.repository.FindUserByEmail(r.Context(), input.Admin.Email); err == nil {
		writeError(w, http.StatusConflict, "could not create account")
		return
	}
	user, err := s.repository.CreateAccount(r.Context(), input)
	if err != nil {
		log.Printf("signup: create account: %v", err)
		writeError(w, http.StatusConflict, "could not create account")
		return
	}
	s.startSession(w, r, models.UserDocument{User: user, Status: "active"}, http.StatusCreated)
}

// refresh rotates the refresh cookie and returns a new access token.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(w, r, "refresh") {
		return
	}
	if !middleware.OriginAllowed(r, s.options.AllowedOrigins) {
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, invalidSession)
		return
	}
	stored, err := s.repository.ConsumeRefreshToken(r.Context(), tokens.Hash(cookie.Value))
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenReused) {
			log.Printf("refresh: reuse of a rotated refresh token detected, sessions revoked")
		} else if !repository.IsNotFound(err) {
			log.Printf("refresh: consume token: %v", err)
		}
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, invalidSession)
		return
	}
	user, err := s.repository.FindUserByID(r.Context(), stored.UserID, stored.CompanyID)
	if err != nil || !user.IsActive() || user.TokenVersion != stored.TokenVersion {
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, invalidSession)
		return
	}
	s.startSession(w, r, user, http.StatusOK)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !middleware.OriginAllowed(r, s.options.AllowedOrigins) {
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		if err := s.repository.RevokeRefreshToken(r.Context(), tokens.Hash(cookie.Value)); err != nil {
			log.Printf("logout: revoke refresh token: %v", err)
		}
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	id, _ := repository.AuthenticatedUserID(r.Context())
	user, err := s.repository.GetMember(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user.User)
}

// startSession issues an access token plus a rotated refresh cookie.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, user models.UserDocument, status int) {
	accessToken, err := s.tokens.Issue(user.ID, user.CompanyID, user.Role, user.TokenVersion)
	if err != nil {
		internalError(w, "could not start session", err)
		return
	}
	plain, hash, err := tokens.New()
	if err != nil {
		internalError(w, "could not start session", err)
		return
	}
	now := time.Now().UTC()
	expiresAt := now.Add(s.options.RefreshTTL)
	if err := s.repository.CreateRefreshToken(r.Context(), models.RefreshTokenDocument{
		TokenHash: hash, UserID: user.ID, CompanyID: user.CompanyID,
		TokenVersion: user.TokenVersion, CreatedAt: now, ExpiresAt: expiresAt,
	}); err != nil {
		internalError(w, "could not start session", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookieName, Value: plain, Path: refreshCookiePath,
		Expires: expiresAt, MaxAge: int(s.options.RefreshTTL.Seconds()),
		HttpOnly: true, Secure: s.options.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, status, models.AuthResponse{
		Token: accessToken, ExpiresIn: int(s.tokens.TTL().Seconds()), User: user.User,
	})
}

func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookieName, Value: "", Path: refreshCookiePath, MaxAge: -1,
		HttpOnly: true, Secure: s.options.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

// requireAuth validates the Bearer access token and loads the user, so a
// revoked account or a stale tokenVersion is rejected immediately.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		raw, found := strings.CutPrefix(header, "Bearer ")
		if !found || raw == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		claims, err := s.tokens.Parse(raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		user, err := s.repository.FindUserByID(r.Context(), claims.UserID, claims.CompanyID)
		if err != nil || !user.IsActive() || user.TokenVersion != claims.TokenVersion {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		ctx := repository.WithAuthenticatedUser(r.Context(), user.ID, user.CompanyID, user.Role)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := repository.AuthenticatedRole(r.Context())
		if !slices.Contains(roles, role) {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		next(w, r)
	}
}
