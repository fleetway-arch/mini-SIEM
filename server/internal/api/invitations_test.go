package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"siem-dashboard/server/internal/tokens"
)

func TestInvitationEmailContent(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	token := env.invite(owner, "new@acme.test", "admin")

	message := env.mail.to("new@acme.test")[0]
	if !strings.Contains(message.Subject, "Acme") {
		t.Errorf("subject = %q, want the company name", message.Subject)
	}
	link := testBaseURL + "/invite/" + token
	for _, body := range []string{message.Text, message.HTML} {
		if !strings.Contains(body, link) || !strings.Contains(body, "Owner of Acme") || !strings.Contains(body, "admin") {
			t.Errorf("email body misses the link, inviter or role:\n%s", body)
		}
	}
	ctx := context.Background()
	byHash, _ := env.db.Collection("invitations").CountDocuments(ctx, bson.M{"tokenHash": tokens.Hash(token)})
	byPlain, _ := env.db.Collection("invitations").CountDocuments(ctx, bson.M{"$or": bson.A{bson.M{"tokenHash": token}, bson.M{"token": token}}})
	if byHash != 1 || byPlain != 0 {
		t.Fatalf("invitation stored by hash=%d, by plain token=%d; only the hash must be stored", byHash, byPlain)
	}
}

func TestInvitationTokens(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	accept := func(token string) result {
		return env.send(request{method: "POST", path: "/api/invitations/" + token + "/accept", body: map[string]string{"fullName": "New Member", "password": strongPassword}})
	}
	get := func(token string) result {
		return env.send(request{method: "GET", path: "/api/invitations/" + token})
	}

	t.Run("valid", func(t *testing.T) {
		token := env.invite(owner, "valid@acme.test", "viewer")
		r := get(token)
		expectStatus(t, r, http.StatusOK)
		var details struct{ CompanyName, Email, Role string }
		r.decode(t, &details)
		if details.Email != "valid@acme.test" || details.Role != "viewer" || details.CompanyName != "Acme" {
			t.Fatalf("invitation details = %+v", details)
		}
	})

	t.Run("tampered", func(t *testing.T) {
		token := env.invite(owner, "tampered@acme.test", "viewer")
		replacement := "A"
		if strings.HasSuffix(token, "A") {
			replacement = "B"
		}
		altered := token[:len(token)-1] + replacement
		expectStatus(t, get(altered), http.StatusNotFound)
		expectStatus(t, accept(altered), http.StatusNotFound)
		expectStatus(t, get(token+"x"), http.StatusNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		token := env.invite(owner, "expired@acme.test", "viewer")
		_, err := env.db.Collection("invitations").UpdateOne(context.Background(),
			bson.M{"tokenHash": tokens.Hash(token)}, bson.M{"$set": bson.M{"expiresAt": time.Now().Add(-time.Minute)}})
		if err != nil {
			t.Fatal(err)
		}
		expectStatus(t, get(token), http.StatusNotFound)
		expectStatus(t, accept(token), http.StatusNotFound)
	})

	t.Run("already used", func(t *testing.T) {
		token := env.invite(owner, "used@acme.test", "viewer")
		expectStatus(t, accept(token), http.StatusOK)
		expectStatus(t, get(token), http.StatusNotFound)
		expectStatus(t, accept(token), http.StatusNotFound)
	})

	t.Run("weak password keeps the invitation usable", func(t *testing.T) {
		token := env.invite(owner, "weak@acme.test", "viewer")
		r := env.send(request{method: "POST", path: "/api/invitations/" + token + "/accept", body: map[string]string{"fullName": "Weak", "password": "password"}})
		expectStatus(t, r, http.StatusBadRequest)
		expectStatus(t, get(token), http.StatusOK)
	})
}

func TestResendRotatesInvitationLink(t *testing.T) {
	env := newTestEnv(t)
	owner := env.signup("Acme", "owner@acme.test")
	first := env.invite(owner, "late@acme.test", "viewer")

	var team []struct {
		ID     int
		Email  string
		Status string
	}
	env.send(request{method: "GET", path: "/api/team", token: owner.token}).decode(t, &team)
	id := 0
	for _, member := range team {
		if member.Email == "late@acme.test" && member.Status == "invited" {
			id = member.ID
		}
	}
	if id == 0 {
		t.Fatalf("pending invitation missing from the team list: %+v", team)
	}

	expectStatus(t, env.send(request{method: "POST", path: "/api/team/" + itoa(id) + "/resend", token: owner.token}), http.StatusOK)
	second := tokenFromEmail(t, env.mail.waitFor(t, "late@acme.test", 2))
	if second == first {
		t.Fatal("resend reused the previous token")
	}
	expectStatus(t, env.send(request{method: "GET", path: "/api/invitations/" + first}), http.StatusNotFound)
	expectStatus(t, env.send(request{method: "GET", path: "/api/invitations/" + second}), http.StatusOK)

	// Cancelling the invitation kills the link.
	expectStatus(t, env.send(request{method: "DELETE", path: "/api/team/" + itoa(id), token: owner.token}), http.StatusNoContent)
	expectStatus(t, env.send(request{method: "GET", path: "/api/invitations/" + second}), http.StatusNotFound)
}
