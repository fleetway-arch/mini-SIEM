package middleware

import (
	"net/http"
	"slices"
)

// CORS only answers for origins listed in allowedOrigins. Credentials are
// allowed so the browser sends the httpOnly refresh cookie.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && slices.Contains(allowedOrigins, origin)
		w.Header().Add("Vary", "Origin")
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "Retry-After")
		}
		if r.Method == http.MethodOptions {
			if !allowed {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// OriginAllowed reports whether a cookie-authenticated request may proceed.
// Browsers always send Origin on cross-origin POSTs, so a foreign Origin is
// rejected; requests without Origin come from non-browser clients, which
// cannot be driven by CSRF.
func OriginAllowed(r *http.Request, allowedOrigins []string) bool {
	origin := r.Header.Get("Origin")
	return origin == "" || slices.Contains(allowedOrigins, origin)
}
