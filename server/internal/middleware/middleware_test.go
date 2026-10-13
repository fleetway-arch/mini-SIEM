package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var origins = []string{"http://localhost:5173"}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		origin      string
		wantStatus  int
		wantAllowed bool
	}{
		{"allowed origin", http.MethodGet, "http://localhost:5173", 200, true},
		{"foreign origin gets no CORS headers", http.MethodGet, "https://evil.example", 200, false},
		{"allowed preflight", http.MethodOptions, "http://localhost:5173", 204, true},
		{"foreign preflight is rejected", http.MethodOptions, "https://evil.example", 403, false},
		{"no origin", http.MethodGet, "", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/auth/login", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			CORS(origins, okHandler()).ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllowed && (allowOrigin != tt.origin || rec.Header().Get("Access-Control-Allow-Credentials") != "true") {
				t.Fatalf("missing CORS headers: %v", rec.Header())
			}
			if !tt.wantAllowed && allowOrigin != "" {
				t.Fatalf("Access-Control-Allow-Origin = %q, want none", allowOrigin)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Cache-Control":          "no-store",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:4321"
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")
	if got := ClientIP(req, false); got != "10.0.0.5" {
		t.Fatalf("ClientIP(untrusted) = %q, X-Forwarded-For must be ignored", got)
	}
	if got := ClientIP(req, true); got != "203.0.113.9" {
		t.Fatalf("ClientIP(trusted) = %q, want the right-most (proxy-added) entry", got)
	}
}

func TestOriginAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if !OriginAllowed(req, origins) {
		t.Fatal("requests without Origin (non-browser) should pass")
	}
	req.Header.Set("Origin", "https://evil.example")
	if OriginAllowed(req, origins) {
		t.Fatal("foreign Origin should be rejected")
	}
}
