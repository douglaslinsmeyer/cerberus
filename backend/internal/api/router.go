package api

import (
	"net/http"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new API router
func NewRouter(database *db.DB) chi.Router {
	r := chi.NewRouter()

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handleRegister(database))
		r.Post("/login", handleLogin(database))
		r.Post("/refresh", handleRefreshToken(database))
	})

	// Protected routes (require authentication)
	// r.Group(func(r chi.Router) {
	// 	r.Use(authMiddleware)

	// 	// Programs
	// 	r.Route("/programs", func(r chi.Router) {
	// 		r.Get("/", handleListPrograms(database))
	// 		r.Post("/", handleCreateProgram(database))
	// 		r.Get("/{programId}", handleGetProgram(database))
	// 		r.Patch("/{programId}", handleUpdateProgram(database))
	// 		r.Delete("/{programId}", handleDeleteProgram(database))
	// 	})
	// })

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
