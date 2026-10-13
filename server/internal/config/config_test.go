package config

import (
	"strings"
	"testing"
)

const secret = "0123456789abcdef0123456789abcdef"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"valid development config", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development"}, ""},
		{"APP_ENV defaults to production and fails fast on http links", map[string]string{"JWT_SECRET": secret}, "APP_BASE_URL"},
		{"valid production config", map[string]string{"JWT_SECRET": secret, "APP_ENV": "production", "APP_BASE_URL": "https://siem.example.com"}, ""},
		{"missing JWT secret", map[string]string{"APP_ENV": "development"}, "JWT_SECRET"},
		{"short JWT secret", map[string]string{"JWT_SECRET": "too-short", "APP_ENV": "development"}, "JWT_SECRET"},
		{"malformed duration", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development", "JWT_TTL": "fifteen"}, "JWT_TTL"},
		{"refresh shorter than access", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development", "JWT_TTL": "2h", "REFRESH_TTL": "1h"}, "REFRESH_TTL"},
		{"wildcard origin", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development", "CORS_ALLOWED_ORIGINS": "*"}, "CORS_ALLOWED_ORIGINS"},
		{"insecure cookie in production", map[string]string{"JWT_SECRET": secret, "APP_ENV": "production", "COOKIE_SECURE": "false", "APP_BASE_URL": "https://siem.example.com"}, "COOKIE_SECURE"},
		{"http base URL in production", map[string]string{"JWT_SECRET": secret, "APP_ENV": "production", "APP_BASE_URL": "http://siem.example.com"}, "APP_BASE_URL"},
		{"SMTP without sender", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development", "SMTP_HOST": "smtp.example.com"}, "MAIL_FROM"},
		{"SMTP user without password", map[string]string{"JWT_SECRET": secret, "APP_ENV": "development", "SMTP_HOST": "smtp.example.com", "MAIL_FROM": "a@example.com", "SMTP_USER": "u"}, "SMTP_PASSWORD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"JWT_SECRET", "JWT_TTL", "REFRESH_TTL", "CORS_ALLOWED_ORIGINS", "APP_ENV", "COOKIE_SECURE", "APP_BASE_URL", "SMTP_HOST", "MAIL_FROM", "SMTP_USER", "SMTP_PASSWORD"} {
				t.Setenv(key, "")
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			err := Validate(Load())
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want it to mention %s", err, tt.wantErr)
			}
		})
	}
}
