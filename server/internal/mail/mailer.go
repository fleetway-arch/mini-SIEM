// Package mail sends transactional emails (invitations, password reset).
package mail

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, textBody, htmlBody string) error
}

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

var ErrInvalidAddress = errors.New("invalid email address")

// SanitizeHeader removes CR, LF and other control characters so a value
// cannot inject extra headers (for example "x\r\nBcc: victim@example.com").
func SanitizeHeader(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

// ParseAddress validates a single address and rejects any control character
// instead of silently stripping it, because a recipient with CR/LF is never
// legitimate.
func ParseAddress(value string) (*mail.Address, error) {
	if SanitizeHeader(value) != value {
		return nil, ErrInvalidAddress
	}
	address, err := mail.ParseAddress(value)
	if err != nil {
		return nil, ErrInvalidAddress
	}
	return address, nil
}

// MaskAddress keeps logs useful without writing full addresses.
func MaskAddress(value string) string {
	local, domain, found := strings.Cut(value, "@")
	if !found || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}

// LogMailer is used when SMTP is not configured. In development it prints the
// text body (which contains the link) so flows can be tested without SMTP;
// otherwise it only records that a message was dropped.
type LogMailer struct {
	ShowContent bool
}

func (m LogMailer) Send(_ context.Context, to, subject, textBody, _ string) error {
	if m.ShowContent {
		log.Printf("mail (SMTP not configured, development only) to=%s subject=%q\n%s", to, subject, textBody)
		return nil
	}
	log.Printf("mail not sent: SMTP is not configured (to=%s subject=%q)", MaskAddress(to), subject)
	return nil
}
