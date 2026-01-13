package api

import (
	"net/http"
	"os"

	"github.com/cerberus/backend/internal/modules/artifacts"
	"github.com/cerberus/backend/internal/modules/financial"
	"github.com/cerberus/backend/internal/modules/programs"
	"github.com/cerberus/backend/internal/modules/risk"
	"github.com/cerberus/backend/internal/platform/db"
	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new API router
func NewRouter(database *db.DB) chi.Router {
	r := chi.NewRouter()

	// Initialize storage client
	storageEndpoint := os.Getenv("STORAGE_ENDPOINT")
	if storageEndpoint == "" {
		storageEndpoint = "http://localhost:9000"
	}
	storageClient := storage.NewRustFSClient(storageEndpoint)

	// Initialize AI client
	// TODO: Move AI client initialization to shared location
	// For now, create a placeholder (will be properly initialized with Redis/metrics later)

	// Initialize artifacts module
	artifactsRepo := artifacts.NewRepository(database)
	artifactsService := artifacts.NewService(artifactsRepo, storageClient)

	// Initialize financial module
	financialRepo := financial.NewRepository(database)
	// Note: financialService needs aiClient but we'll pass nil for now
	// Worker will handle AI analysis via events
	financialService := financial.NewServiceWithMocks(financialRepo, storageClient, nil)

	// Initialize risk module
	riskRepo := risk.NewRepository(database)
	riskService := risk.NewService(riskRepo)
	conversationService := risk.NewConversationService(riskRepo)

	// Initialize programs module
	programsRepo := programs.NewRepository(database)
	programsService := programs.NewService(programsRepo)
	configService := programs.NewConfigService(database)
	stakeholderRepo := programs.NewStakeholderRepository(database)

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handleRegister(database))
		r.Post("/login", handleLogin(database))
		r.Post("/refresh", handleRefreshToken(database))
	})

	// Artifacts routes (Phase 2: no auth required yet)
	artifacts.RegisterRoutes(r, artifactsService)

	// Financial routes (Phase 3: no auth required yet)
	financial.RegisterRoutes(r, financialService)

	// Risk routes (Phase 3: no auth required yet)
	risk.RegisterRoutes(r, riskService, conversationService)

	// Programs configuration routes (no auth required yet)
	programs.RegisterConfigRoutes(r, configService)
	programs.RegisterStakeholderRoutes(r, stakeholderRepo)

	// Programs CRUD routes (Phase 4 - no auth required yet)
	programs.RegisterRoutes(r, programsService)

	return r
}

// Placeholder handlers (to be implemented)
func handleRegister(db *db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusNotImplemented, ErrorResponse{Error: "Not implemented yet"})
	}
}

func handleLogin(db *db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusNotImplemented, ErrorResponse{Error: "Not implemented yet"})
	}
}

func handleRefreshToken(db *db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusNotImplemented, ErrorResponse{Error: "Not implemented yet"})
	}
}
