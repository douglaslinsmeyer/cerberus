package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/cerberus/backend/internal/platform/events"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get configuration from environment
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "cerberus")
	dbUser := getEnv("DB_USER", "cerberus")
	dbPassword := getEnv("DB_PASSWORD", "cerberus_dev")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	// Connect to database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	database, err := db.Connect(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Database connection established")

	// Create event bus
	eventBus, err := events.NewNATSBus(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer eventBus.Close()

	log.Println("NATS connection established")

	// Create worker context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event bus
	go func() {
		if err := eventBus.Start(ctx); err != nil {
			log.Printf("Event bus error: %v", err)
		}
	}()

	log.Println("Worker started, processing events...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received, stopping worker...")

	cancel()
	log.Println("Worker stopped gracefully")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
