package mail

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSanitizeHeaderStripsControlCharacters(t *testing.T) {
	got := SanitizeHeader("Hello\r\nBcc: victim@example.com\x00\t!")
	if strings.ContainsAny(got, "\r\n\x00\t") {
		t.Fatalf("SanitizeHeader() kept control characters: %q", got)
	}
	if got != "HelloBcc: victim@example.com!" {
		t.Fatalf("SanitizeHeader() = %q", got)
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"alice@example.com", true},
		{"Mini SIEM <no-reply@example.com>", true},
		{"alice@example.com\r\nBcc: victim@example.com", false},
		{"alice@example.com\nCc: x@y.z", false},
		{"not an address", false},
		{"", false},
	}
	for _, tt := range tests {
		_, err := ParseAddress(tt.input)
		if (err == nil) != tt.valid {
			t.Errorf("ParseAddress(%q) error = %v, want valid=%v", tt.input, err, tt.valid)
		}
	}
}

func TestBuildMessageCannotBeHeaderInjected(t *testing.T) {
	raw, err := BuildMessage("Mini SIEM <no-reply@example.com>", "bob@example.com",
		"Hi\r\nBcc: victim@example.com", "text body", "<p>html</p>")
	if err != nil {
		t.Fatal(err)
	}
	headers, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Fatalf("injected header found: %q", line)
		}
	}
	if !strings.Contains(headers, "To: bob@example.com") || !strings.Contains(headers, "multipart/alternative") {
		t.Fatalf("unexpected headers:\n%s", headers)
	}
}

func TestBuildMessageEncodesNonASCIISubject(t *testing.T) {
	raw, _ := BuildMessage("a@example.com", "b@example.com", "Réinitialisation", "t", "h")
	if !strings.Contains(string(raw), "Subject: =?utf-8?q?R=C3=A9initialisation?=") {
		t.Fatalf("subject is not MIME encoded:\n%s", raw)
	}
}

func TestInvitationEmailEscapesHTMLAndContainsLink(t *testing.T) {
	link := "https://app.example.com/invite/abc_DEF-123"
	message, err := InvitationEmail("bob@example.com", InvitationData{
		CompanyName: `Acme <script>alert(1)</script>`, InviterName: `Eve "<b>"`, Role: "viewer", Link: link,
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.To != "bob@example.com" || !strings.Contains(message.Subject, "Acme") {
		t.Fatalf("recipient/subject = %q / %q", message.To, message.Subject)
	}
	if strings.Contains(message.HTML, "<script>") || strings.Contains(message.HTML, `"<b>"`) {
		t.Fatal("HTML body contains unescaped user input")
	}
	if !strings.Contains(message.HTML, "&lt;script&gt;") {
		t.Fatal("HTML body does not contain the escaped company name")
	}
	if !strings.Contains(message.HTML, `href="`+link+`"`) || !strings.Contains(message.Text, link) {
		t.Fatal("link missing from the HTML or text body")
	}
}

func TestEmailLinkRejectsJavascriptURL(t *testing.T) {
	message, _ := PasswordResetEmail("a@example.com", "javascript:alert(1)")
	if strings.Contains(message.HTML, `href="javascript:`) {
		t.Fatal("html/template should neutralise javascript: URLs")
	}
}

// fakeMailer fails the first failures calls, then records messages.
type fakeMailer struct {
	mu       sync.Mutex
	failures int
	calls    int
	sent     []Message
}

func (f *fakeMailer) Send(_ context.Context, to, subject, text, html string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failures {
		return errors.New("smtp unavailable")
	}
	f.sent = append(f.sent, Message{To: to, Subject: subject, Text: text, HTML: html})
	return nil
}

func (f *fakeMailer) snapshot() (int, []Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, append([]Message{}, f.sent...)
}

func TestQueueRetriesThenDelivers(t *testing.T) {
	mailer := &fakeMailer{failures: 2}
	queue := NewQueue(mailer, 4, time.Millisecond)
	if !queue.Enqueue(Message{To: "bob@example.com", Subject: "hello"}) {
		t.Fatal("Enqueue() refused a message")
	}
	if err := queue.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls, sent := mailer.snapshot()
	if calls != 3 || len(sent) != 1 || sent[0].To != "bob@example.com" {
		t.Fatalf("calls=%d sent=%+v, want 3 calls and one delivery", calls, sent)
	}
}

func TestQueueGivesUpAfterThreeRetries(t *testing.T) {
	mailer := &fakeMailer{failures: 100}
	queue := NewQueue(mailer, 4, time.Millisecond)
	queue.Enqueue(Message{To: "bob@example.com", Subject: "hello"})
	_ = queue.Close(context.Background())
	if calls, sent := mailer.snapshot(); calls != 4 || len(sent) != 0 {
		t.Fatalf("calls=%d sent=%d, want 1 attempt + 3 retries", calls, len(sent))
	}
}

func TestQueueEnqueueNeverBlocks(t *testing.T) {
	blocker := make(chan struct{})
	queue := NewQueue(MailerFunc(func() error { <-blocker; return nil }), 1, time.Millisecond)
	defer func() { close(blocker); _ = queue.Close(context.Background()) }()
	accepted := 0
	for i := 0; i < 5; i++ {
		if queue.Enqueue(Message{To: "x@example.com"}) {
			accepted++
		}
	}
	if accepted >= 5 {
		t.Fatal("a full queue should drop messages instead of blocking")
	}
}

func TestQueueRejectsAfterClose(t *testing.T) {
	queue := NewQueue(&fakeMailer{}, 1, time.Millisecond)
	_ = queue.Close(context.Background())
	if queue.Enqueue(Message{To: "x@example.com"}) {
		t.Fatal("Enqueue() after Close() should return false")
	}
}

type MailerFunc func() error

func (f MailerFunc) Send(context.Context, string, string, string, string) error { return f() }

func TestMaskAddress(t *testing.T) {
	if got := MaskAddress("alice@example.com"); got != "a***@example.com" {
		t.Fatalf("MaskAddress() = %q", got)
	}
	if got := MaskAddress("broken"); got != "***" {
		t.Fatalf("MaskAddress() = %q", got)
	}
}
