package api

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"siem-dashboard/server/internal/auth"
	"siem-dashboard/server/internal/mail"
	"siem-dashboard/server/internal/middleware"
	"siem-dashboard/server/internal/ratelimit"
	"siem-dashboard/server/internal/repository"
)

const maxBodyBytes = 1 << 20

type Options struct {
	Tokens            *auth.TokenManager
	RefreshTTL        time.Duration
	CookieSecure      bool
	AllowedOrigins    []string
	TrustProxy        bool
	RateLimitWindow   time.Duration
	RateLimitPerIP    int
	RateLimitPerEmail int
	AppBaseURL        string
	Mail              MailQueue
}

// MailQueue is satisfied by *mail.Queue; tests use a fake.
type MailQueue interface {
	Enqueue(mail.Message) bool
}

type Server struct {
	repository   repository.Repository
	tokens       *auth.TokenManager
	mail         MailQueue
	options      Options
	ipLimiter    *ratelimit.Limiter
	emailLimiter *ratelimit.Limiter
}

func New(data repository.Repository, options Options) *Server {
	return &Server{
		repository:   data,
		tokens:       options.Tokens,
		mail:         options.Mail,
		options:      options,
		ipLimiter:    ratelimit.New(options.RateLimitPerIP, options.RateLimitWindow),
		emailLimiter: ratelimit.New(options.RateLimitPerEmail, options.RateLimitWindow),
	}
}

var managers = []string{roleOwner, roleAdmin}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/signup", s.signup)
	mux.HandleFunc("POST /api/auth/refresh", s.refresh)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.me))
	mux.HandleFunc("GET /api/team", s.requireAuth(s.requireRole(managers, s.listTeam)))
	mux.HandleFunc("POST /api/team/invitations", s.requireAuth(s.requireRole(managers, s.createInvitations)))
	mux.HandleFunc("POST /api/team/{id}/resend", s.requireAuth(s.requireRole(managers, s.resendInvitation)))
	mux.HandleFunc("PATCH /api/team/{id}", s.requireAuth(s.requireRole(managers, s.updateMember)))
	mux.HandleFunc("DELETE /api/team/{id}", s.requireAuth(s.requireRole(managers, s.deleteMember)))
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
	return middleware.SecurityHeaders(middleware.CORS(s.options.AllowedOrigins, mux))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
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

// internalError logs the cause server-side and returns a generic message.
func internalError(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	writeError(w, http.StatusInternalServerError, message)
}

// allow applies a rate limiter and writes 429 with Retry-After when exceeded.
func allow(w http.ResponseWriter, limiter *ratelimit.Limiter, key string) bool {
	ok, retryAfter := limiter.Allow(key)
	if ok {
		return true
	}
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests, "too many requests, please try again later")
	return false
}

func (s *Server) allowIP(w http.ResponseWriter, r *http.Request, route string) bool {
	return allow(w, s.ipLimiter, route+":"+middleware.ClientIP(r, s.options.TrustProxy))
}

func (s *Server) allowEmail(w http.ResponseWriter, route, email string) bool {
	return allow(w, s.emailLimiter, route+":"+email)
}
