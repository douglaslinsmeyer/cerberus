package api

import (
	"os"

	"github.com/cerberus/backend/internal/modules/artifacts"
	"github.com/cerberus/backend/internal/modules/financial"
	"github.com/cerberus/backend/internal/modules/programs"
	"github.com/cerberus/backend/internal/modules/risk"
	"github.com/cerberus/backend/internal/platform/auth"
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

	// Initialize auth
	authRepo := auth.NewRepository(database)
	tokenService := auth.NewTokenService()
	authService := auth.NewService(authRepo, tokenService)

	// Auth routes (PUBLIC - no middleware)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handleRegister(database))
		r.Post("/login", handleLogin(database, authService))
		r.Post("/refresh", handleRefreshToken(database, authService))

		// Protected auth routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware(tokenService, authRepo))
			r.Post("/logout", handleLogout(authService))
			r.Post("/switch-program", handleSwitchProgram(authService))
		})
	})

	// PROTECTED ROUTES - Require authentication
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(tokenService, authRepo))

		// Register module routes (pass authRepo for program access checks)
		artifacts.RegisterRoutes(r, artifactsService, authRepo)
		financial.RegisterRoutes(r, financialService, authRepo)
		risk.RegisterRoutes(r, riskService, conversationService, authRepo)
		programs.RegisterRoutes(r, programsService, authRepo)
		programs.RegisterConfigRoutes(r, configService, authRepo)
		programs.RegisterStakeholderRoutes(r, stakeholderRepo, authRepo)
	})

	return r
}
