package api

// Integration tests: they drive the real routes (JWT middleware, handlers,
// MongoDB repository) through httptest. They need a MongoDB server and are
// skipped unless MONGODB_TEST_URI is set, for example:
//
//	docker run -d --rm -p 27018:27017 mongo:7
//	MONGODB_TEST_URI=mongodb://localhost:27018 go test ./...

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"siem-dashboard/server/internal/auth"
	"siem-dashboard/server/internal/mail"
	"siem-dashboard/server/internal/models"
	"siem-dashboard/server/internal/password"
	"siem-dashboard/server/internal/repository"
)

const (
	testSecret     = "test-secret-0123456789abcdef0123456789"
	testOrigin     = "http://localhost:5173"
	testBaseURL    = "http://app.test"
	strongPassword = "Str0ng!Passw0rd"
)

func TestMain(m *testing.M) {
	// Cheap Argon2id parameters: the cost itself is covered by the password
	// package tests.
	password.DefaultParams = password.Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	os.Exit(m.Run())
}

// fakeMail records queued emails instead of sending them.
type fakeMail struct {
	mu       sync.Mutex
	messages []mail.Message
}

func (f *fakeMail) Enqueue(message mail.Message) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, message)
	return true
}

func (f *fakeMail) to(address string) []mail.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []mail.Message
	for _, message := range f.messages {
		if message.To == address {
			result = append(result, message)
		}
	}
	return result
}

// waitFor returns the n-th email (1-based) sent to address; some emails are
// queued from a background goroutine.
func (f *fakeMail) waitFor(t *testing.T, address string, n int) mail.Message {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if messages := f.to(address); len(messages) >= n {
			return messages[n-1]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no email #%d was sent to %s", n, address)
	return mail.Message{}
}

type testEnv struct {
	t       *testing.T
	handler http.Handler
	db      *mongo.Database
	mail    *fakeMail
}

func newTestEnv(t *testing.T, configure ...func(*Options)) *testEnv {
	t.Helper()
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI is not set; skipping MongoDB integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repo, err := repository.NewMongo(ctx, uri, fmt.Sprintf("siem_test_%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("connect to test MongoDB: %v", err)
	}
	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = repo.Database().Drop(ctx)
		_ = repo.Close(ctx)
	})
	mailbox := &fakeMail{}
	options := Options{
		Tokens:            auth.NewTokenManager(testSecret, 15*time.Minute),
		RefreshTTL:        time.Hour,
		CookieSecure:      true,
		AllowedOrigins:    []string{testOrigin},
		RateLimitWindow:   time.Minute,
		RateLimitPerIP:    10000,
		RateLimitPerEmail: 10000,
		AppBaseURL:        testBaseURL,
		Mail:              mailbox,
	}
	for _, apply := range configure {
		apply(&options)
	}
	return &testEnv{t: t, handler: New(repo, options).Routes(), db: repo.Database(), mail: mailbox}
}

type request struct {
	method  string
	path    string
	token   string
	body    any
	cookies []*http.Cookie
	origin  string
	ip      string
	header  map[string]string
}

type result struct {
	status  int
	body    []byte
	header  http.Header
	cookies []*http.Cookie
}

func (e *testEnv) send(req request) result {
	e.t.Helper()
	var reader io.Reader
	if req.body != nil {
		raw, err := json.Marshal(req.body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	httpReq := httptest.NewRequest(req.method, req.path, reader)
	httpReq.RemoteAddr = "192.0.2.10:1234"
	if req.ip != "" {
		httpReq.RemoteAddr = req.ip + ":1234"
	}
	if req.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.token)
	}
	if req.origin != "" {
		httpReq.Header.Set("Origin", req.origin)
	}
	for key, value := range req.header {
		httpReq.Header.Set(key, value)
	}
	for _, cookie := range req.cookies {
		httpReq.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	e.handler.ServeHTTP(recorder, httpReq)
	return result{status: recorder.Code, body: recorder.Body.Bytes(), header: recorder.Header(), cookies: recorder.Result().Cookies()}
}

func (r result) decode(t *testing.T, target any) {
	t.Helper()
	if err := json.Unmarshal(r.body, target); err != nil {
		t.Fatalf("decode response %q: %v", r.body, err)
	}
}

func (r result) cookie(name string) *http.Cookie {
	for _, cookie := range r.cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func expectStatus(t *testing.T, r result, want int) {
	t.Helper()
	if r.status != want {
		t.Fatalf("status = %d, want %d (body %s)", r.status, want, r.body)
	}
}

type session struct {
	token   string
	refresh *http.Cookie
	user    models.User
}

func (e *testEnv) sessionFrom(r result) session {
	e.t.Helper()
	var response models.AuthResponse
	r.decode(e.t, &response)
	refresh := r.cookie(refreshCookieName)
	if response.Token == "" || refresh == nil {
		e.t.Fatalf("response did not start a session: %s", r.body)
	}
	return session{token: response.Token, refresh: refresh, user: response.User}
}

func (e *testEnv) signup(company, email string) session {
	e.t.Helper()
	r := e.send(request{method: "POST", path: "/api/auth/signup", body: map[string]any{
		"company": map[string]string{"name": company, "industry": "Technology", "size": "1-10", "country": "France", "securityContact": email},
		"admin":   map[string]string{"fullName": "Owner of " + company, "email": email, "password": strongPassword},
	}})
	expectStatus(e.t, r, http.StatusCreated)
	return e.sessionFrom(r)
}

func (e *testEnv) login(email, plainPassword string) result {
	e.t.Helper()
	return e.send(request{method: "POST", path: "/api/auth/login", body: map[string]string{"email": email, "password": plainPassword}})
}

var linkToken = regexp.MustCompile(`/(?:invite|reset-password)/([A-Za-z0-9_-]+)`)

func tokenFromEmail(t *testing.T, message mail.Message) string {
	t.Helper()
	match := linkToken.FindStringSubmatch(message.Text)
	if match == nil {
		t.Fatalf("no link in email: %s", message.Text)
	}
	return match[1]
}

// invite creates an invitation and returns the token from the email.
func (e *testEnv) invite(by session, email, role string) string {
	e.t.Helper()
	before := len(e.mail.to(email))
	r := e.send(request{method: "POST", path: "/api/team/invitations", token: by.token, body: map[string]any{"emails": []string{email}, "role": role}})
	expectStatus(e.t, r, http.StatusOK)
	return tokenFromEmail(e.t, e.mail.waitFor(e.t, email, before+1))
}

func (e *testEnv) addMember(by session, email, role string) session {
	e.t.Helper()
	token := e.invite(by, email, role)
	r := e.send(request{method: "POST", path: "/api/invitations/" + token + "/accept", body: map[string]string{"fullName": strings.Split(email, "@")[0], "password": strongPassword}})
	expectStatus(e.t, r, http.StatusOK)
	return e.sessionFrom(r)
}
