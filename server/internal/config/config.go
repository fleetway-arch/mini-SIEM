package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const minJWTSecretLength = 32

type Config struct {
	Env                 string
	Port                string
	MongoURI            string
	Database            string
	AllowedOrigins      []string
	MongoConnectRetries int

	JWTSecret    string
	JWTTTL       time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool
	TrustProxy   bool

	RateLimitWindow   time.Duration
	RateLimitPerIP    int
	RateLimitPerEmail int
	// RateLimitIngestPerKey caps ingestion requests per key and window.
	RateLimitIngestPerKey int

	BruteForceThreshold int
	BruteForceWindow    time.Duration

	AppBaseURL     string
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPassword   string
	SMTPRequireTLS bool
	MailFrom       string
}

// SMTPConfigured is false when SMTP_HOST is empty; emails are then only
// logged (with their content in development).
func (c Config) SMTPConfigured() bool {
	return c.SMTPHost != ""
}

func Load() Config {
	return Config{
		// Defaults to production so a forgotten APP_ENV never enables
		// development behaviour (emails with links printed to the log).
		Env:                 env("APP_ENV", "production"),
		Port:                env("PORT", "8080"),
		MongoURI:            env("MONGODB_URI", "mongodb://localhost:27017"),
		Database:            env("MONGODB_DATABASE", "siem"),
		AllowedOrigins:      envList("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		MongoConnectRetries: envInt("MONGO_CONNECT_RETRIES", 12),

		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTTTL:       envDuration("JWT_TTL", 15*time.Minute),
		RefreshTTL:   envDuration("REFRESH_TTL", 7*24*time.Hour),
		CookieSecure: envBool("COOKIE_SECURE", true),
		TrustProxy:   envBool("TRUST_PROXY", false),

		RateLimitWindow:   envDuration("RATE_LIMIT_WINDOW", time.Minute),
		RateLimitPerIP:    envInt("RATE_LIMIT_AUTH_PER_IP", 20),
		RateLimitPerEmail: envInt("RATE_LIMIT_AUTH_PER_EMAIL", 5),

		RateLimitIngestPerKey: envInt("RATE_LIMIT_INGEST_PER_KEY", 600),
		BruteForceThreshold:   envInt("DETECT_BRUTE_FORCE_THRESHOLD", 10),
		BruteForceWindow:      envDuration("DETECT_BRUTE_FORCE_WINDOW", 5*time.Minute),

		AppBaseURL:     strings.TrimRight(env("APP_BASE_URL", "http://localhost:5173"), "/"),
		SMTPHost:       os.Getenv("SMTP_HOST"),
		SMTPPort:       env("SMTP_PORT", "587"),
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPRequireTLS: envBool("SMTP_REQUIRE_TLS", true),
		MailFrom:       os.Getenv("MAIL_FROM"),
	}
}

func (c Config) Address() string {
	return ":" + c.Port
}

func (c Config) MongoDatabase() string {
	return c.Database
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func Validate(c Config) error {
	if c.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}
	if c.MongoURI == "" {
		return fmt.Errorf("MONGODB_URI cannot be empty")
	}
	if c.Database == "" {
		return fmt.Errorf("MONGODB_DATABASE cannot be empty")
	}
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return fmt.Errorf("APP_ENV must be development, production or test")
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET is required and must be at least %d characters (generate one with: openssl rand -hex 32)", minJWTSecretLength)
	}
	if c.JWTTTL <= 0 || c.RefreshTTL <= 0 || c.RateLimitWindow <= 0 {
		return fmt.Errorf("JWT_TTL, REFRESH_TTL and RATE_LIMIT_WINDOW must be positive durations such as 15m or 168h")
	}
	if c.BruteForceWindow <= 0 {
		return fmt.Errorf("DETECT_BRUTE_FORCE_WINDOW must be a positive duration such as 5m")
	}
	if c.RefreshTTL <= c.JWTTTL {
		return fmt.Errorf("REFRESH_TTL must be longer than JWT_TTL")
	}
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot be empty")
	}
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain '*' because credentials are allowed")
		}
	}
	if c.IsProduction() && !c.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE cannot be false when APP_ENV=production")
	}
	base, err := url.Parse(c.AppBaseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return fmt.Errorf("APP_BASE_URL must be an absolute http(s) URL such as https://siem.example.com")
	}
	if c.IsProduction() && base.Scheme != "https" {
		return fmt.Errorf("APP_BASE_URL must use https when APP_ENV=production")
	}
	if c.SMTPConfigured() {
		if c.MailFrom == "" {
			return fmt.Errorf("MAIL_FROM is required when SMTP_HOST is set")
		}
		if (c.SMTPUser == "") != (c.SMTPPassword == "") {
			return fmt.Errorf("SMTP_USER and SMTP_PASSWORD must be set together")
		}
		if c.IsProduction() && !c.SMTPRequireTLS {
			return fmt.Errorf("SMTP_REQUIRE_TLS cannot be false when APP_ENV=production")
		}
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

// envDuration returns -1 for a malformed value so Validate rejects it
// instead of silently using the fallback.
func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return -1
	}
	return value
}

func envList(key, fallback string) []string {
	var values []string
	for _, value := range strings.Split(env(key, fallback), ",") {
		if value = strings.TrimRight(strings.TrimSpace(value), "/"); value != "" {
			values = append(values, value)
		}
	}
	return values
}
