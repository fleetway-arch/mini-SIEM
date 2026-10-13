package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                string
	MongoURI            string
	Database            string
	CORSOrigin          string
	MongoConnectRetries int
}

func Load() Config {
	return Config{
		Port:                env("PORT", "8080"),
		MongoURI:            env("MONGODB_URI", "mongodb://localhost:27017"),
		Database:            env("MONGODB_DATABASE", "siem"),
		CORSOrigin:          env("CORS_ORIGIN", "http://localhost:5173"),
		MongoConnectRetries: envInt("MONGO_CONNECT_RETRIES", 12),
	}
}

func (c Config) Address() string {
	return ":" + c.Port
}

func (c Config) MongoDatabase() string {
	if c.Database == "" {
		return ""
	}
	return c.Database
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
