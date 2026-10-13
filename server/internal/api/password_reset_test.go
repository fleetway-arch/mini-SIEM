package api

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"siem-dashboard/server/internal/tokens"
)

func (e *testEnv) forgot(email string) result {
	return e.send(request{method: "POST", path: "/api/auth/forgot-password", body: map[string]string{"email": email}})
}

func (e *testEnv) reset(token, newPassword string) result {
	return e.send(request{method: "POST", path: "/api/auth/reset-password", body: map[string]string{"token": token, "newPassword": newPassword}})
}

// requestReset triggers forgot-password and returns the token of the n-th
// reset email received by email.
func (e *testEnv) requestReset(email string, n int) string {
	e.t.Helper()
	expectStatus(e.t, e.forgot(email), http.StatusOK)
	return tokenFromEmail(e.t, e.mail.waitFor(e.t, email, n))
}

func TestForgotPasswordDoesNotRevealAccounts(t *testing.T) {
	env := newTestEnv(t)
	env.signup("Acme", "owner@acme.test")

	known := env.forgot("owner@acme.test")
	unknown := env.forgot("nobody@acme.test")
	expectStatus(t, known, http.StatusOK)
	expectStatus(t, unknown, http.StatusOK)
	if !bytes.Equal(known.body, unknown.body) {
		t.Fatalf("responses differ: %s vs %s", known.body, unknown.body)
	}

	message := env.mail.waitFor(t, "owner@acme.test", 1)
	if !strings.Contains(message.Text, testBaseURL+"/reset-password/") {
		t.Fatalf("reset email has no link: %s", message.Text)
	}
	time.Sleep(100 * time.Millisecond)
	if sent := env.mail.to("nobody@acme.test"); len(sent) != 0 {
		t.Fatalf("an email was sent to an unknown address: %+v", sent)
	}
}

func TestPasswordReset(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	const newPassword = "N3w!Passw0rd-x"
	invalidBody := env.reset("never-issued", newPassword).body

	t.Run("weak password is refused and keeps the token", func(t *testing.T) {
		token := env.requestReset("owner@acme.test", 1)
		expectStatus(t, env.reset(token, "alllowercase-1"), http.StatusBadRequest)
		expectStatus(t, env.reset(token, "Sh0rt!"), http.StatusBadRequest)
		var stillUsable int64
		stillUsable, _ = env.db.Collection("password_resets").CountDocuments(context.Background(), bson.M{"tokenHash": tokens.Hash(token), "used": false})
		if stillUsable != 1 {
			t.Fatal("a rejected weak password consumed the token")
		}
	})

	t.Run("a new request invalidates the previous link", func(t *testing.T) {
		older := env.requestReset("owner@acme.test", 2)
		newer := env.requestReset("owner@acme.test", 3)
		r := env.reset(older, newPassword)
		expectStatus(t, r, http.StatusBadRequest)
		if !bytes.Equal(r.body, invalidBody) {
			t.Fatalf("superseded token error %s differs from the generic one %s", r.body, invalidBody)
		}
		_ = newer
	})

	t.Run("expired token", func(t *testing.T) {
		token := env.requestReset("owner@acme.test", 4)
		_, err := env.db.Collection("password_resets").UpdateOne(context.Background(),
			bson.M{"tokenHash": tokens.Hash(token)}, bson.M{"$set": bson.M{"expiresAt": time.Now().Add(-time.Second)}})
		if err != nil {
			t.Fatal(err)
		}
		r := env.reset(token, newPassword)
		expectStatus(t, r, http.StatusBadRequest)
		if !bytes.Equal(r.body, invalidBody) {
			t.Fatalf("expired token error %s differs from the generic one %s", r.body, invalidBody)
		}
	})

	t.Run("valid reset invalidates sessions and the token", func(t *testing.T) {
		token := env.requestReset("owner@acme.test", 5)
		expectStatus(t, env.reset(token, newPassword), http.StatusOK)

		expectStatus(t, env.send(request{method: "GET", path: "/api/auth/me", token: owner.token}), http.StatusUnauthorized)
		expectStatus(t, env.send(request{method: "POST", path: "/api/auth/refresh", cookies: []*http.Cookie{owner.refresh}}), http.StatusUnauthorized)
		expectStatus(t, env.login("owner@acme.test", strongPassword), http.StatusUnauthorized)
		expectStatus(t, env.login("owner@acme.test", newPassword), http.StatusOK)

		reused := env.reset(token, "An0ther!Passw0rd")
		expectStatus(t, reused, http.StatusBadRequest)
		if !bytes.Equal(reused.body, invalidBody) {
			t.Fatalf("reused token error %s differs from the generic one %s", reused.body, invalidBody)
		}
		notification := env.mail.waitFor(t, "owner@acme.test", 6)
		if !strings.Contains(notification.Subject, "modifié") {
			t.Fatalf("expected a password-changed notification, got %q", notification.Subject)
		}
	})

	t.Run("revoked users cannot reset", func(t *testing.T) {
		member := env.addMember(env.sessionFrom(env.login("owner@acme.test", newPassword)), "gone@acme.test", "viewer")
		token := env.requestReset("gone@acme.test", 1)
		admin := env.sessionFrom(env.login("owner@acme.test", newPassword))
		expectStatus(t, env.send(request{method: "PATCH", path: "/api/team/" + itoa(member.user.ID), token: admin.token, body: map[string]string{"status": "revoked"}}), http.StatusOK)
		expectStatus(t, env.reset(token, newPassword), http.StatusBadRequest)
	})
}

func TestPasswordResetRateLimit(t *testing.T) {
	env := newTestEnv(t, func(o *Options) { o.RateLimitPerEmail = 2 })
	for i := 0; i < 2; i++ {
		expectStatus(t, env.forgot("someone@acme.test"), http.StatusOK)
	}
	r := env.forgot("someone@acme.test")
	expectStatus(t, r, http.StatusTooManyRequests)
	if r.header.Get("Retry-After") == "" {
		t.Fatal("429 without Retry-After")
	}
	for i := 0; i < 2; i++ {
		expectStatus(t, env.reset("guess", "N3w!Passw0rd-x"), http.StatusBadRequest)
	}
	expectStatus(t, env.reset("guess", "N3w!Passw0rd-x"), http.StatusTooManyRequests)
}
