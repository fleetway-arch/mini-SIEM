package config

import (
	"fmt"
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
}

func Load() Config {
	return Config{
		Env:                 env("APP_ENV", "development"),
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
