package programs

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterRoutes registers all program endpoints
func RegisterRoutes(r chi.Router, service *Service) {
	r.Route("/programs", func(r chi.Router) {
		r.Get("/", handleListPrograms(service))
		r.Post("/", handleCreateProgram(service))

		r.Route("/{programId}", func(r chi.Router) {
			r.Get("/", handleGetProgram(service))
			r.Patch("/", handleUpdateProgram(service))
		})
	})
}

// handleListPrograms lists all programs with statistics
func handleListPrograms(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programs, err := service.ListPrograms(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, ListProgramsResponse{
			Programs: programs,
			Total:    len(programs),
		})
	}
}

// handleGetProgram retrieves a single program by ID
func handleGetProgram(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		program, err := service.GetProgram(r.Context(), programID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, http.StatusNotFound, "Program not found")
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, program)
	}
}

// handleCreateProgram creates a new program
func handleCreateProgram(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateProgramRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// TODO: Get user ID from JWT claims
		// For now, use hardcoded demo user ID
		userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		programID, err := service.CreateProgram(r.Context(), req, userID)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				respondError(w, http.StatusConflict, err.Error())
				return
			}
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondCreated(w, map[string]interface{}{
			"program_id": programID,
			"message":    "Program created successfully",
		})
	}
}

// handleUpdateProgram updates an existing program
func handleUpdateProgram(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req UpdateProgramRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := service.UpdateProgram(r.Context(), programID, req); err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, http.StatusNotFound, "Program not found")
				return
			}
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Program updated successfully",
		})
	}
}

// Note: Helper functions (respondSuccess, respondCreated, respondError) are defined in config_handlers.go
