package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cerberus/backend/internal/modules/artifacts"
	"github.com/cerberus/backend/internal/modules/financial"
	"github.com/cerberus/backend/internal/modules/programs"
	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/cerberus/backend/internal/platform/db"
	"github.com/cerberus/backend/internal/platform/events"
	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
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

	// Get API keys
	anthropicKey := getEnv("ANTHROPIC_API_KEY", "")
	openaiKey := getEnv("OPENAI_API_KEY", "")
	redisURL := getEnv("REDIS_URL", "redis:6379")

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Redis connection established")

	// Create AI client
	metricsTracker := ai.NewDBMetricsTracker(database)
	claudeClient := ai.NewClient(&ai.ClientConfig{
		APIKey:         anthropicKey,
		RedisClient:    redisClient,
		MetricsTracker: metricsTracker,
	})

	// Create storage client for OCR
	storageEndpoint := getEnv("STORAGE_ENDPOINT", "http://rustfs:9000")
	storageClient := storage.NewRustFSClient(storageEndpoint)

	// Create artifacts repository and analyzer
	artifactsRepo := artifacts.NewRepository(database)
	aiAnalyzer := artifacts.NewAIAnalyzer(claudeClient, artifactsRepo)
	embeddingsService := artifacts.NewEmbeddingsService(openaiKey, artifactsRepo)
	ocrService := artifacts.NewOCRService(artifactsRepo, storageClient)

	// Create financial module services
	financialRepo := financial.NewRepository(database)
	invoiceAnalyzer := financial.NewInvoiceAnalyzer(claudeClient, financialRepo)

	// Create program context builder
	configService := programs.NewConfigService(database)
	stakeholderRepo := programs.NewStakeholderRepository(database)
	contextBuilder := ai.NewContextBuilder(configService, stakeholderRepo)

	// Create event bus
	eventBus, err := events.NewNATSBus(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer eventBus.Close()

	log.Println("NATS connection established")

	// Subscribe to artifact.uploaded events
	eventBus.Subscribe(events.ArtifactUploaded, func(ctx context.Context, event *events.Event) error {
		log.Printf("Processing artifact upload event: %s", event.ID)

		// Extract artifact ID from payload
		artifactIDStr, ok := event.Payload["artifact_id"].(string)
		if !ok {
			return fmt.Errorf("invalid artifact_id in payload")
		}

		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			return fmt.Errorf("failed to parse artifact ID: %w", err)
		}

		// Get artifact
		artifact, err := artifactsRepo.GetByID(ctx, artifactID)
		if err != nil {
			return fmt.Errorf("failed to get artifact: %w", err)
		}

		// Create program context (simplified for now)
		// Build program context from database
		programContext := contextBuilder.BuildContextOrDefault(ctx, artifact.ProgramID)

		// Process artifact with AI analysis
		if err := aiAnalyzer.ProcessArtifact(ctx, artifact, programContext); err != nil {
			log.Printf("Failed to process artifact %s: %v", artifactID, err)
			return err
		}

		log.Printf("Successfully analyzed artifact: %s (%s)", artifact.Filename, artifactID)

		// Generate embeddings if configured
		if openaiKey != "" {
			if err := embeddingsService.GenerateEmbeddings(ctx, artifactID); err != nil {
				log.Printf("Failed to generate embeddings for %s: %v", artifactID, err)
				// Don't fail the whole process if embeddings fail
			} else {
				log.Printf("Generated embeddings for artifact: %s", artifactID)
			}
		}

		// Publish artifact.analyzed event
		analyzedEvent := events.NewEvent(
			events.ArtifactAnalyzed,
			event.ProgramID,
			"artifacts",
			map[string]interface{}{
				"artifact_id": artifactID.String(),
			},
		).WithCorrelationID(event.CorrelationID)

		if err := eventBus.Publish(ctx, analyzedEvent); err != nil {
			log.Printf("Failed to publish artifact.analyzed event: %v", err)
		}

		return nil
	})

	// Create worker context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event bus
	go func() {
		if err := eventBus.Start(ctx); err != nil {
			log.Printf("Event bus error: %v", err)
		}
	}()

	// Also poll database for pending artifacts (backup mechanism)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get pending artifacts
				pending, err := artifactsRepo.GetPendingArtifacts(ctx, 10)
				if err != nil {
					log.Printf("Failed to get pending artifacts: %v", err)
					continue
				}

				for _, artifact := range pending {
					log.Printf("Processing pending artifact from database: %s", artifact.ArtifactID)

					programContext := &ai.ProgramContext{
						ProgramName: "Default Program",
						ProgramCode: "DEFAULT",
					}

					if err := aiAnalyzer.ProcessArtifact(ctx, &artifact, programContext); err != nil {
						log.Printf("Failed to process artifact %s: %v", artifact.ArtifactID, err)
						continue
					}

					// Generate embeddings
					if openaiKey != "" {
						if err := embeddingsService.GenerateEmbeddings(ctx, artifact.ArtifactID); err != nil {
							log.Printf("Failed to generate embeddings: %v", err)
						}
					}

					// Check if artifact is an invoice and process financially
					updatedArtifact, err := artifactsRepo.GetByID(ctx, artifact.ArtifactID)
					if err == nil && updatedArtifact.ArtifactCategory.Valid {
						category := updatedArtifact.ArtifactCategory.String
						log.Printf("Artifact %s categorized as: %s", artifact.ArtifactID, category)

						if category == "invoice" && updatedArtifact.RawContent.Valid {
							log.Printf("Processing invoice: %s", artifact.ArtifactID)

							programContext := &ai.ProgramContext{
								ProgramName: "Default Program",
								ProgramCode: "DEFAULT",
							}

							err := invoiceAnalyzer.ProcessInvoice(
								ctx,
								updatedArtifact.ArtifactID,
								updatedArtifact.RawContent.String,
								updatedArtifact.ProgramID,
								programContext,
							)
							if err != nil {
								log.Printf("Failed to process invoice %s: %v", artifact.ArtifactID, err)
							} else {
								log.Printf("Invoice processed successfully: %s", artifact.ArtifactID)
							}
						}
					}
				}
			}
		}
	}()

	// Poll for OCR-required artifacts
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get OCR-required artifacts
				rows, err := database.QueryContext(ctx, `
					SELECT artifact_id FROM artifacts
					WHERE processing_status = 'ocr_required'
					  AND deleted_at IS NULL
					ORDER BY uploaded_at ASC
					LIMIT 5
				`)
				if err != nil {
					log.Printf("Failed to query OCR artifacts: %v", err)
					continue
				}

				var ocrArtifacts []uuid.UUID
				for rows.Next() {
					var id uuid.UUID
					if err := rows.Scan(&id); err != nil {
						log.Printf("Failed to scan artifact ID: %v", err)
						continue
					}
					ocrArtifacts = append(ocrArtifacts, id)
				}
				rows.Close()

				// Process each OCR artifact
				for _, artifactID := range ocrArtifacts {
					log.Printf("Processing OCR-required artifact: %s", artifactID)

					if err := ocrService.ProcessOCRRequired(ctx, artifactID); err != nil {
						log.Printf("Failed to process OCR for %s: %v", artifactID, err)
						continue
					}

					log.Printf("OCR completed for artifact: %s (now pending AI analysis)", artifactID)
				}
			}
		}
	}()

	log.Println("Worker started, processing artifacts and OCR...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received, stopping worker...")

	cancel()
	time.Sleep(2 * time.Second) // Give handlers time to finish
	log.Println("Worker stopped gracefully")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
