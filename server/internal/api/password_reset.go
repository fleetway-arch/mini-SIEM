package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"siem-dashboard/server/internal/mail"
	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/repository"
	"siem-dashboard/server/internal/tokens"
)

const (
	forgotPasswordMessage = "If an account exists for this email, a reset link has been sent."
	invalidResetLink      = "invalid or expired reset link"
	backgroundTimeout     = 15 * time.Second
)

// forgotPassword always answers 200 with the same body. The request path only
// performs the user lookup in both cases; creating the token and queuing the
// email run in the background so timing does not reveal whether the email
// exists.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(w, r, "forgot-password") {
		return
	}
	var input models.ForgotPasswordRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if !s.allowEmail(w, "forgot-password", email) {
		return
	}
	user, err := s.repository.FindUserByEmail(r.Context(), email)
	if err != nil && !repository.IsNotFound(err) {
		log.Printf("forgot-password: find user: %v", err)
	}
	if err == nil && user.IsActive() {
		go s.issuePasswordReset(user)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": forgotPasswordMessage})
}

func (s *Server) issuePasswordReset(user models.UserDocument) {
	ctx, cancel := context.WithTimeout(context.Background(), backgroundTimeout)
	defer cancel()
	plain, hash, err := tokens.New()
	if err != nil {
		log.Printf("forgot-password: generate token: %v", err)
		return
	}
	if err := s.repository.CreatePasswordReset(ctx, user, hash); err != nil {
		log.Printf("forgot-password: store token for user %d: %v", user.ID, err)
		return
	}
	message, err := mail.PasswordResetEmail(user.Email, s.options.AppBaseURL+"/reset-password/"+url.PathEscape(plain))
	if err != nil {
		log.Printf("forgot-password: render email: %v", err)
		return
	}
	s.mail.Enqueue(message)
}

// resetPassword is limited per IP and per token: the account (and so the
// email) is unknown until the token has been validated.
func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(w, r, "reset-password") {
		return
	}
	var input models.ResetPasswordRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Token == "" {
		writeError(w, http.StatusBadRequest, invalidResetLink)
		return
	}
	tokenHash := tokens.Hash(input.Token)
	if !allow(w, s.emailLimiter, "reset-password:"+tokenHash) {
		return
	}
	if err := password.ValidateStrength(input.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	newHash, err := password.Hash(input.NewPassword)
	if err != nil {
		internalError(w, "could not reset password", err)
		return
	}
	user, err := s.repository.ResetPassword(r.Context(), tokenHash, newHash)
	if err != nil {
		if !errors.Is(err, repository.ErrInvalidResetToken) {
			log.Printf("reset-password: %v", err)
		}
		writeError(w, http.StatusBadRequest, invalidResetLink)
		return
	}
	if message, err := mail.PasswordChangedEmail(user.Email, time.Now()); err == nil {
		s.mail.Enqueue(message)
	} else {
		log.Printf("reset-password: render notification: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
