package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"siem-dashboard/server/internal/api"
	"siem-dashboard/server/internal/auth"
	"siem-dashboard/server/internal/config"
	"siem-dashboard/server/internal/repository"
)

func main() {
	_ = godotenv.Load()
	configuration := config.Load()
	if err := config.Validate(configuration); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	var data *repository.Mongo
	var err error
	for attempt := 1; attempt <= configuration.MongoConnectRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		data, err = repository.NewMongo(ctx, configuration.MongoURI, configuration.MongoDatabase())
		cancel()
		if err == nil {
			break
		}
		log.Printf("MongoDB connection attempt %d/%d failed: %v", attempt, configuration.MongoConnectRetries, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("could not connect to MongoDB after %d attempts: %v", configuration.MongoConnectRetries, err)
	}
	if err := data.EnsureIndexes(context.Background()); err != nil {
		log.Fatalf("could not ensure MongoDB indexes: %v", err)
	}
	defer data.Close(context.Background())

	application := api.New(data, api.Options{
		Tokens:            auth.NewTokenManager(configuration.JWTSecret, configuration.JWTTTL),
		RefreshTTL:        configuration.RefreshTTL,
		CookieSecure:      configuration.CookieSecure,
		AllowedOrigins:    configuration.AllowedOrigins,
		TrustProxy:        configuration.TrustProxy,
		RateLimitWindow:   configuration.RateLimitWindow,
		RateLimitPerIP:    configuration.RateLimitPerIP,
		RateLimitPerEmail: configuration.RateLimitPerEmail,
	})
	server := &http.Server{
		Addr:              configuration.Address(),
		Handler:           application.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("SIEM API listening on %s (env=%s)", configuration.Address(), configuration.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
