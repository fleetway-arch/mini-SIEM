package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	// RequireTLS refuses to send when the server does not offer STARTTLS.
	// Only disable it for a local test server such as Mailpit.
	RequireTLS bool
	Timeout    time.Duration
}

type SMTPMailer struct {
	config SMTPConfig
	from   string
}

func NewSMTPMailer(config SMTPConfig) (*SMTPMailer, error) {
	from, err := ParseAddress(config.From)
	if err != nil {
		return nil, fmt.Errorf("MAIL_FROM: %w", err)
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}
	return &SMTPMailer{config: config, from: from.Address}, nil
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, textBody, htmlBody string) error {
	recipient, err := ParseAddress(to)
	if err != nil {
		return err
	}
	message, err := BuildMessage(m.config.From, recipient.String(), subject, textBody, htmlBody)
	if err != nil {
		return err
	}

	dialer := net.Dialer{Timeout: m.config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(m.config.Host, m.config.Port))
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	deadline := time.Now().Add(m.config.Timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("start SMTP session: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	} else if m.config.RequireTLS {
		return errors.New("SMTP server does not support STARTTLS and SMTP_REQUIRE_TLS is enabled")
	}
	if m.config.User != "" {
		// PlainAuth itself refuses to send credentials over an unencrypted
		// connection to a remote host.
		if err := client.Auth(smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish message: %w", err)
	}
	return client.Quit()
}

// BuildMessage renders a multipart/alternative message. Every header value is
// sanitized and the subject is MIME-encoded, so user data cannot add headers.
func BuildMessage(from, to, subject, textBody, htmlBody string) ([]byte, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	headers := [][2]string{
		{"From", SanitizeHeader(from)},
		{"To", SanitizeHeader(to)},
		{"Subject", mime.QEncoding.Encode("utf-8", SanitizeHeader(subject))},
		{"Date", time.Now().UTC().Format(time.RFC1123Z)},
		{"Message-ID", "<" + boundary + "@mini-siem>"},
		{"MIME-Version", "1.0"},
		{"Content-Type", `multipart/alternative; boundary="` + boundary + `"`},
	}
	for _, header := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", header[0], header[1])
	}
	buf.WriteString("\r\n")
	for _, part := range []struct{ contentType, body string }{
		{"text/plain; charset=utf-8", textBody},
		{"text/html; charset=utf-8", htmlBody},
	} {
		fmt.Fprintf(&buf, "--%s\r\nContent-Type: %s\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", boundary, part.contentType)
		qp := quotedprintable.NewWriter(&buf)
		if _, err := qp.Write([]byte(strings.ReplaceAll(part.body, "\n", "\r\n"))); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}

func randomBoundary() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate MIME boundary: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
