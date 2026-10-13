package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"siem-dashboard/server/internal/api"
	"siem-dashboard/server/internal/config"
	"siem-dashboard/server/internal/middleware"
	"siem-dashboard/server/internal/repository"
)

func main() {
	_ = godotenv.Load()
	configuration := config.Load()
	if err := config.Validate(configuration); err != nil {
		log.Fatal(err)
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
	application := api.New(data)
	server := &http.Server{
		Addr:              configuration.Address(),
		Handler:           middleware.CORS(configuration.CORSOrigin, application.Routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("SIEM API listening on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
