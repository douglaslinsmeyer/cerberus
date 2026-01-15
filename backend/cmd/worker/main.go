package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cerberus/backend/internal/modules/artifacts"
	"github.com/cerberus/backend/internal/modules/financial"
	"github.com/cerberus/backend/internal/modules/programs"
	"github.com/cerberus/backend/internal/modules/risk"
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

	// Initialize AI Analyzer with enriched context support
	log.Println("Initializing AI Analyzer with enriched context graph support...")
	aiAnalyzer, err := artifacts.InitializeAIAnalyzerWithContext(claudeClient, artifactsRepo, redisClient)
	if err != nil {
		log.Printf("Warning: Failed to initialize context graph, using basic analyzer: %v", err)
		aiAnalyzer = artifacts.NewAIAnalyzer(claudeClient, artifactsRepo)
	} else {
		log.Println("✅ Enriched context graph system ENABLED")
	}

	embeddingsService := artifacts.NewEmbeddingsService(openaiKey, artifactsRepo)
	ocrService := artifacts.NewOCRService(artifactsRepo, storageClient)

	// Create financial module services
	financialRepo := financial.NewRepository(database)
	invoiceAnalyzer := financial.NewInvoiceAnalyzer(claudeClient, financialRepo)

	// Create risk detection services
	riskRepo := risk.NewRepository(database)
	riskDetector := risk.NewRiskDetector(riskRepo)

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

		// Detect risks from insights (best effort - don't fail artifact processing)
		insights, err := fetchArtifactInsights(ctx, database, artifactID, artifact.ProgramID)
		if err != nil {
			log.Printf("Warning: Failed to fetch insights for risk detection: %v", err)
		} else if len(insights) > 0 {
			log.Printf("Analyzing %d insights for risk detection...", len(insights))
			if err := riskDetector.AnalyzeForRisks(ctx, insights); err != nil {
				log.Printf("Warning: Risk detection failed: %v", err)
			} else {
				log.Printf("Risk detection completed for artifact: %s", artifactID)
			}

			// Enrich existing risks with new insights
			log.Printf("Checking for risk enrichment opportunities from %d insights...", len(insights))
			if err := riskDetector.EnrichExistingRisks(ctx, insights); err != nil {
				log.Printf("Warning: Risk enrichment failed: %v", err)
			} else {
				log.Printf("Risk enrichment check completed for artifact: %s", artifactID)
			}
		}

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

					// Detect risks from insights (best effort)
					insights, err := fetchArtifactInsights(ctx, database, artifact.ArtifactID, artifact.ProgramID)
					if err != nil {
						log.Printf("Warning: Failed to fetch insights for risk detection: %v", err)
					} else if len(insights) > 0 {
						log.Printf("Analyzing %d insights for risk detection...", len(insights))
						if err := riskDetector.AnalyzeForRisks(ctx, insights); err != nil {
							log.Printf("Warning: Risk detection failed: %v", err)
						} else {
							log.Printf("Risk detection completed for artifact: %s", artifact.ArtifactID)
						}

						// Enrich existing risks with new insights
						log.Printf("Checking for risk enrichment opportunities from %d insights...", len(insights))
						if err := riskDetector.EnrichExistingRisks(ctx, insights); err != nil {
							log.Printf("Warning: Risk enrichment failed: %v", err)
						} else {
							log.Printf("Risk enrichment check completed for artifact: %s", artifact.ArtifactID)
						}
					}

					// Check if artifact is an invoice and process financially
					updatedArtifact, err := artifactsRepo.GetByID(ctx, artifact.ArtifactID)
					if err == nil && updatedArtifact.ArtifactCategory.Valid {
						category := updatedArtifact.ArtifactCategory.String
						log.Printf("Artifact %s categorized as: %s", artifact.ArtifactID, category)

						if category == "invoice" && updatedArtifact.RawContent.Valid {
							log.Printf("Processing invoice artifact: %s (filename: %s)", artifact.ArtifactID, updatedArtifact.Filename)

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
								log.Printf("ERROR: Failed to process invoice: %v", err)
							} else {
								log.Printf("SUCCESS: Invoice processed for artifact: %s", artifact.ArtifactID)
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

	// Poll for aggregate risk analysis (cross-artifact pattern detection)
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("Running aggregate risk analysis...")

				// Query for programs with recent insights
				query := `
					SELECT DISTINCT a.program_id
					FROM artifact_insights ai
					INNER JOIN artifacts a ON ai.artifact_id = a.artifact_id
					WHERE ai.extracted_at >= NOW() - INTERVAL '24 hours'
					  AND ai.is_dismissed = FALSE
					  AND a.deleted_at IS NULL
				`

				rows, err := database.QueryContext(ctx, query)
				if err != nil {
					log.Printf("Failed to query programs for aggregate analysis: %v", err)
					continue
				}

				var programIDs []uuid.UUID
				for rows.Next() {
					var programID uuid.UUID
					if err := rows.Scan(&programID); err != nil {
						log.Printf("Failed to scan program ID: %v", err)
						continue
					}
					programIDs = append(programIDs, programID)
				}
				rows.Close()

				// Analyze each program's recent insights
				for _, programID := range programIDs {
					log.Printf("Analyzing aggregate risks for program: %s", programID)

					// Fetch insights from last 24 hours excluding those already converted to suggestions
					insightsQuery := `
						SELECT ai.insight_id, ai.artifact_id, ai.insight_type, ai.title,
						       ai.description, ai.severity, ai.confidence_score
						FROM artifact_insights ai
						INNER JOIN artifacts a ON ai.artifact_id = a.artifact_id
						WHERE a.program_id = $1
						  AND ai.extracted_at >= NOW() - INTERVAL '24 hours'
						  AND ai.is_dismissed = FALSE
						  AND a.deleted_at IS NULL
						  AND NOT EXISTS (
						      SELECT 1 FROM risk_suggestions rs
						      WHERE rs.source_insight_id = ai.insight_id
						  )
						ORDER BY ai.severity DESC, ai.confidence_score DESC
						LIMIT 100
					`

					insightRows, err := database.QueryContext(ctx, insightsQuery, programID)
					if err != nil {
						log.Printf("Failed to query insights for program %s: %v", programID, err)
						continue
					}

					var insights []risk.ArtifactInsight
					for insightRows.Next() {
						var insight risk.ArtifactInsight
						var severity sql.NullString
						var confidence sql.NullFloat64

						err := insightRows.Scan(
							&insight.InsightID,
							&insight.ArtifactID,
							&insight.InsightType,
							&insight.Title,
							&insight.Description,
							&severity,
							&confidence,
						)
						if err != nil {
							log.Printf("Failed to scan insight: %v", err)
							continue
						}

						insight.ProgramID = programID
						insight.Severity = severity.String
						insight.ConfidenceScore = confidence.Float64
						insights = append(insights, insight)
					}
					insightRows.Close()

					if len(insights) > 0 {
						log.Printf("Found %d unprocessed insights for program %s", len(insights), programID)
						if err := riskDetector.AnalyzeForRisks(ctx, insights); err != nil {
							log.Printf("Failed aggregate risk analysis for program %s: %v", programID, err)
						} else {
							log.Printf("Completed aggregate risk analysis for program %s", programID)
						}

						// Enrich existing risks with the insights
						log.Printf("Checking for risk enrichment opportunities for program %s...", programID)
						if err := riskDetector.EnrichExistingRisks(ctx, insights); err != nil {
							log.Printf("Failed aggregate risk enrichment for program %s: %v", programID, err)
						} else {
							log.Printf("Completed aggregate risk enrichment for program %s", programID)
						}
					}
				}

				log.Println("Aggregate risk analysis completed")
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

// fetchArtifactInsights retrieves insights for risk analysis
func fetchArtifactInsights(ctx context.Context, database *db.DB, artifactID, programID uuid.UUID) ([]risk.ArtifactInsight, error) {
	query := `
		SELECT insight_id, artifact_id, insight_type, title, description, severity, confidence_score
		FROM artifact_insights
		WHERE artifact_id = $1 AND is_dismissed = FALSE
		ORDER BY extracted_at DESC
	`

	rows, err := database.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to query insights: %w", err)
	}
	defer rows.Close()

	var insights []risk.ArtifactInsight
	for rows.Next() {
		var insight risk.ArtifactInsight
		var severity sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&insight.InsightID,
			&insight.ArtifactID,
			&insight.InsightType,
			&insight.Title,
			&insight.Description,
			&severity,
			&confidence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan insight: %w", err)
		}

		insight.ProgramID = programID
		insight.Severity = severity.String
		insight.ConfidenceScore = confidence.Float64

		insights = append(insights, insight)
	}

	return insights, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
