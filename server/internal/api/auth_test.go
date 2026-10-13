package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"

	"siem-dashboard/server/internal/auth"
	"siem-dashboard/server/internal/tokens"
)

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	revoked := env.addMember(owner, "revoked@acme.test", "viewer")
	expectStatus(t, env.send(request{method: "PATCH", path: fmt.Sprintf("/api/team/%d", revoked.user.ID), token: owner.token, body: map[string]string{"status": "revoked"}}), http.StatusOK)

	cases := map[string]result{
		"wrong password": env.login("owner@acme.test", "Wr0ng!Password"),
		"unknown email":  env.login("nobody@acme.test", strongPassword),
		"revoked user":   env.login("revoked@acme.test", strongPassword),
	}
	reference := cases["wrong password"]
	for name, r := range cases {
		if r.status != http.StatusUnauthorized || !bytes.Equal(r.body, reference.body) {
			t.Errorf("%s: status %d body %s, want 401 %s", name, r.status, r.body, reference.body)
		}
		if r.cookie(refreshCookieName) != nil {
			t.Errorf("%s: a refresh cookie was set on a failed login", name)
		}
	}
}

func TestLoginSetsHardenedRefreshCookie(t *testing.T) {
	env := newTestEnv(t)
	env.signup("Acme", "owner@acme.test")
	r := env.login("OWNER@acme.test ", strongPassword)
	expectStatus(t, r, http.StatusOK)
	cookie := r.cookie(refreshCookieName)
	if cookie == nil || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api/auth" {
		t.Fatalf("refresh cookie = %+v, want HttpOnly, Secure, SameSite=Strict, Path=/api/auth", cookie)
	}
	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(raw) != 32 {
		t.Fatalf("refresh token is %d bytes, want 32 random bytes", len(raw))
	}
	count, _ := env.db.Collection("refresh_tokens").CountDocuments(context.Background(), bson.M{"tokenHash": tokens.Hash(cookie.Value)})
	plain, _ := env.db.Collection("refresh_tokens").CountDocuments(context.Background(), bson.M{"tokenHash": cookie.Value})
	if count != 1 || plain != 0 {
		t.Fatalf("stored refresh tokens: by hash=%d, by plain value=%d; only the hash must be stored", count, plain)
	}
}

func TestAccessTokenValidation(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	other := env.signup("Other", "owner@other.test")
	u := owner.user

	issue := func(manager *auth.TokenManager, companyID, version int) string {
		token, err := manager.Issue(u.ID, companyID, u.Role, version)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	signed := func(method jwt.SigningMethod, key any) string {
		token, err := jwt.NewWithClaims(method, jwt.MapClaims{
			"uid": u.ID, "cid": u.CompanyID, "role": u.Role, "ver": 0, "iss": "mini-siem",
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		}).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer " + owner.token, http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"not a bearer token", "Basic " + owner.token, http.StatusUnauthorized},
		{"legacy demo token", "Bearer demo-token-owner@acme.test", http.StatusUnauthorized},
		{"expired", "Bearer " + issue(auth.NewTokenManager(testSecret, -time.Minute), u.CompanyID, 0), http.StatusUnauthorized},
		{"wrong signature", "Bearer " + issue(auth.NewTokenManager("another-secret-0123456789abcdef0123", time.Hour), u.CompanyID, 0), http.StatusUnauthorized},
		{"wrong algorithm", "Bearer " + signed(jwt.SigningMethodHS512, []byte(testSecret)), http.StatusUnauthorized},
		{"none algorithm", "Bearer " + signed(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType), http.StatusUnauthorized},
		{"stale token version", "Bearer " + issue(auth.NewTokenManager(testSecret, time.Hour), u.CompanyID, 5), http.StatusUnauthorized},
		{"company claim of another tenant", "Bearer " + issue(auth.NewTokenManager(testSecret, time.Hour), other.user.CompanyID, 0), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := request{method: "GET", path: "/api/auth/me"}
			if tt.header != "" {
				req.header = map[string]string{"Authorization": tt.header}
			}
			expectStatus(t, env.send(req), tt.want)
		})
	}
}

func TestRoleChangeInvalidatesSessions(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	member := env.addMember(owner, "member@acme.test", "viewer")
	expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: member.token}), http.StatusOK)

	expectStatus(t, env.send(request{method: "PATCH", path: fmt.Sprintf("/api/team/%d", member.user.ID), token: owner.token, body: map[string]string{"role": "admin"}}), http.StatusOK)

	expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: member.token}), http.StatusUnauthorized)
	expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{member.refresh}}), http.StatusUnauthorized)
	// The member can log in again and gets the new role.
	fresh := env.sessionFrom(env.login("member@acme.test", strongPassword))
	if fresh.user.Role != "admin" {
		t.Fatalf("role after re-login = %q, want admin", fresh.user.Role)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	env := newTestEnv(t)
	first := env.signup("Acme", "owner@acme.test")

	r := env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{first.refresh}, origin: testOrigin})
	expectStatus(t, r, http.StatusOK)
	second := env.sessionFrom(r)
	if second.refresh.Value == first.refresh.Value {
		t.Fatal("refresh did not rotate the refresh token")
	}
	expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: second.token}), http.StatusOK)

	t.Run("rotated token cannot be reused", func(t *testing.T) {
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{first.refresh}}), http.StatusUnauthorized)
		// Inside the grace window (two tabs racing) the newest session survives.
		expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: second.token}), http.StatusOK)
	})

	t.Run("late reuse revokes the whole session family", func(t *testing.T) {
		_, err := env.db.Collection("refresh_tokens").UpdateOne(context.Background(),
			bson.M{"tokenHash": tokens.Hash(first.refresh.Value)},
			bson.M{"$set": bson.M{"revokedAt": time.Now().Add(-time.Minute)}})
		if err != nil {
			t.Fatal(err)
		}
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{first.refresh}}), http.StatusUnauthorized)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{second.refresh}}), http.StatusUnauthorized)
	})

	t.Run("missing or unknown cookie", func(t *testing.T) {
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh"}), http.StatusUnauthorized)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{{Name: refreshCookieName, Value: "forged"}}}), http.StatusUnauthorized)
	})
}

func TestRefreshRejectsForeignOrigin(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{owner.refresh}, origin: "https://evil.example"}), http.StatusForbidden)
	expectStatus(t, env.send(request{method: "POST", path: "/api/auth/logout", cookies: []*http.Cookie{owner.refresh}, origin: "https://evil.example"}), http.StatusForbidden)
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	r := env.send(request{method: "POST", path: "/api/auth/logout", cookies: []*http.Cookie{owner.refresh}, origin: testOrigin})
	expectStatus(t, r, http.StatusNoContent)
	if cleared := r.cookie(refreshCookieName); cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("logout did not clear the cookie: %+v", cleared)
	}
	expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{owner.refresh}}), http.StatusUnauthorized)
}

func TestAuthRateLimits(t *testing.T) {
	t.Run("per email", func(t *testing.T) {
		env := newTestEnv(t, func(o *Options) { o.RateLimitPerEmail = 2 })
		for i := 0; i < 2; i++ {
			expectStatus(t, env.login("victim@acme.test", "Wr0ng!Password"), http.StatusUnauthorized)
		}
		r := env.send(request{method: "POST", path: "/api/auth/login", ip: "198.51.100.7", body: map[string]string{"email": "victim@acme.test", "password": "x"}})
		expectStatus(t, r, http.StatusTooManyRequests)
		if seconds, err := strconv.Atoi(r.header.Get("Retry-After")); err != nil || seconds < 1 || seconds > 60 {
			t.Fatalf("Retry-After = %q, want 1-60 seconds", r.header.Get("Retry-After"))
		}
	})
	t.Run("per IP across emails", func(t *testing.T) {
		env := newTestEnv(t, func(o *Options) { o.RateLimitPerIP = 3 })
		for i := 0; i < 3; i++ {
			expectStatus(t, env.login(fmt.Sprintf("user%d@acme.test", i), "x"), http.StatusUnauthorized)
		}
		expectStatus(t, env.login("user9@acme.test", "x"), http.StatusTooManyRequests)
		// Another client is not affected.
		other := env.send(request{method: "POST", path: "/api/auth/login", ip: "198.51.100.8", body: map[string]string{"email": "user9@acme.test", "password": "x"}})
		expectStatus(t, other, http.StatusUnauthorized)
	})
	t.Run("refresh and signup are limited", func(t *testing.T) {
		env := newTestEnv(t, func(o *Options) { o.RateLimitPerIP = 1 })
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh"}), http.StatusUnauthorized)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh"}), http.StatusTooManyRequests)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/signup", body: map[string]any{}}), http.StatusBadRequest)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/signup", body: map[string]any{}}), http.StatusTooManyRequests)
	})
}

func TestTeamRoleEnforcement(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	admin := env.addMember(owner, "admin@acme.test", "admin")
	otherAdmin := env.addMember(owner, "admin2@acme.test", "admin")
	viewer := env.addMember(owner, "viewer@acme.test", "viewer")
	member := func(id int) string { return fmt.Sprintf("/api/team/%d", id) }

	tests := []struct {
		name string
		req  request
		want int
	}{
		{"viewer cannot list the team", request{method: "GET", path: "/api/team", token: viewer.token}, http.StatusForbidden},
		{"viewer cannot promote themselves", request{method: "PATCH", path: member(viewer.user.ID), token: viewer.token, body: map[string]string{"role": "admin"}}, http.StatusForbidden},
		{"viewer cannot invite", request{method: "POST", path: "/api/team/invitations", token: viewer.token, body: map[string]any{"emails": []string{"x@acme.test"}, "role": "viewer"}}, http.StatusForbidden},
		{"nobody can invite an owner", request{method: "POST", path: "/api/team/invitations", token: owner.token, body: map[string]any{"emails": []string{"x@acme.test"}, "role": "owner"}}, http.StatusBadRequest},
		{"owner role cannot be assigned", request{method: "PATCH", path: member(admin.user.ID), token: owner.token, body: map[string]string{"role": "owner"}}, http.StatusBadRequest},
		{"admin cannot change another admin", request{method: "PATCH", path: member(otherAdmin.user.ID), token: admin.token, body: map[string]string{"role": "viewer"}}, http.StatusForbidden},
		{"admin cannot delete the owner", request{method: "DELETE", path: member(owner.user.ID), token: admin.token}, http.StatusForbidden},
		{"owner cannot modify themselves", request{method: "PATCH", path: member(owner.user.ID), token: owner.token, body: map[string]string{"status": "revoked"}}, http.StatusForbidden},
		{"admin can revoke a viewer", request{method: "PATCH", path: member(viewer.user.ID), token: admin.token, body: map[string]string{"status": "revoked"}}, http.StatusOK},
		{"owner can delete an admin", request{method: "DELETE", path: member(otherAdmin.user.ID), token: owner.token}, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectStatus(t, env.send(tt.req), tt.want)
		})
	}
	// Revocation applies immediately to existing sessions.
	expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: viewer.token}), http.StatusUnauthorized)
}
