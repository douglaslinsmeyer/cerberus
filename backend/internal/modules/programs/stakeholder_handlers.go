package programs

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterStakeholderRoutes registers stakeholder management endpoints
func RegisterStakeholderRoutes(r chi.Router, repo *StakeholderRepository) {
	r.Route("/programs/{programId}/stakeholders", func(r chi.Router) {
		r.Get("/", handleListStakeholders(repo))
		r.Post("/", handleCreateStakeholder(repo))

		r.Route("/{stakeholderId}", func(r chi.Router) {
			r.Get("/", handleGetStakeholder(repo))
			r.Put("/", handleUpdateStakeholder(repo))
			r.Delete("/", handleDeleteStakeholder(repo))
		})
	})
}

// handleListStakeholders lists all stakeholders for a program with optional filtering
func handleListStakeholders(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		// Parse query parameters
		filter := StakeholderFilter{
			ProgramID: programID,
			Limit:     50,  // Default limit
			Offset:    0,   // Default offset
		}

		// Optional filters
		if typeParam := r.URL.Query().Get("type"); typeParam != "" {
			filter.StakeholderType = typeParam
		}

		if internalParam := r.URL.Query().Get("is_internal"); internalParam != "" {
			if internalParam == "true" {
				isInternal := true
				filter.IsInternal = &isInternal
			} else if internalParam == "false" {
				isInternal := false
				filter.IsInternal = &isInternal
			}
		}

		if engagementParam := r.URL.Query().Get("engagement_level"); engagementParam != "" {
			filter.EngagementLevel = engagementParam
		}

		// Pagination
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				filter.Limit = limit
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
				filter.Offset = offset
			}
		}

		stakeholders, err := repo.ListByProgram(r.Context(), filter)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"stakeholders": stakeholders,
			"limit":        filter.Limit,
			"offset":       filter.Offset,
		})
	}
}

// handleCreateStakeholder creates a new stakeholder
func handleCreateStakeholder(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req CreateStakeholderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		if req.PersonName == "" {
			respondError(w, http.StatusBadRequest, "Person name is required")
			return
		}
		if req.StakeholderType == "" {
			respondError(w, http.StatusBadRequest, "Stakeholder type is required")
			return
		}

		// Validate stakeholder type
		validTypes := map[string]bool{
			"internal": true,
			"external": true,
			"vendor":   true,
			"partner":  true,
			"customer": true,
		}
		if !validTypes[req.StakeholderType] {
			respondError(w, http.StatusBadRequest, "Invalid stakeholder type (must be: internal, external, vendor, partner, customer)")
			return
		}

		// Validate engagement level if provided
		if req.EngagementLevel != nil {
			validLevels := map[string]bool{
				"key":       true,
				"primary":   true,
				"secondary": true,
				"observer":  true,
			}
			if !validLevels[*req.EngagementLevel] {
				respondError(w, http.StatusBadRequest, "Invalid engagement level (must be: key, primary, secondary, observer)")
				return
			}
		}

		// Create stakeholder
		stakeholder := &Stakeholder{
			StakeholderID:   uuid.New(),
			ProgramID:       programID,
			PersonName:      req.PersonName,
			StakeholderType: req.StakeholderType,
			IsInternal:      req.IsInternal,
		}

		if req.Email != nil {
			stakeholder.Email = toNullString(*req.Email)
		}
		if req.Role != nil {
			stakeholder.Role = toNullString(*req.Role)
		}
		if req.Organization != nil {
			stakeholder.Organization = toNullString(*req.Organization)
		}
		if req.EngagementLevel != nil {
			stakeholder.EngagementLevel = toNullString(*req.EngagementLevel)
		}
		if req.Department != nil {
			stakeholder.Department = toNullString(*req.Department)
		}
		if req.Notes != nil {
			stakeholder.Notes = toNullString(*req.Notes)
		}

		if err := repo.Create(r.Context(), stakeholder); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, stakeholder)
	}
}

// handleGetStakeholder retrieves a single stakeholder
func handleGetStakeholder(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stakeholderIDStr := chi.URLParam(r, "stakeholderId")
		stakeholderID, err := uuid.Parse(stakeholderIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid stakeholder ID")
			return
		}

		stakeholder, err := repo.GetByID(r.Context(), stakeholderID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Stakeholder not found")
			return
		}

		respondSuccess(w, stakeholder)
	}
}

// handleUpdateStakeholder updates a stakeholder
func handleUpdateStakeholder(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stakeholderIDStr := chi.URLParam(r, "stakeholderId")
		stakeholderID, err := uuid.Parse(stakeholderIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid stakeholder ID")
			return
		}

		var req UpdateStakeholderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate stakeholder type if provided
		if req.StakeholderType != nil {
			validTypes := map[string]bool{
				"internal": true,
				"external": true,
				"vendor":   true,
				"partner":  true,
				"customer": true,
			}
			if !validTypes[*req.StakeholderType] {
				respondError(w, http.StatusBadRequest, "Invalid stakeholder type")
				return
			}
		}

		// Validate engagement level if provided
		if req.EngagementLevel != nil {
			validLevels := map[string]bool{
				"key":       true,
				"primary":   true,
				"secondary": true,
				"observer":  true,
			}
			if !validLevels[*req.EngagementLevel] {
				respondError(w, http.StatusBadRequest, "Invalid engagement level")
				return
			}
		}

		if err := repo.Update(r.Context(), stakeholderID, req); err != nil {
			if err.Error() == "no fields to update" {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err.Error() == "stakeholder not found or already deleted" {
				respondError(w, http.StatusNotFound, err.Error())
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return updated stakeholder
		stakeholder, err := repo.GetByID(r.Context(), stakeholderID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to retrieve updated stakeholder")
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message":     "Stakeholder updated successfully",
			"stakeholder": stakeholder,
		})
	}
}

// handleDeleteStakeholder deletes a stakeholder
func handleDeleteStakeholder(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stakeholderIDStr := chi.URLParam(r, "stakeholderId")
		stakeholderID, err := uuid.Parse(stakeholderIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid stakeholder ID")
			return
		}

		if err := repo.Delete(r.Context(), stakeholderID); err != nil {
			if err.Error() == "stakeholder not found or already deleted" {
				respondError(w, http.StatusNotFound, err.Error())
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondNoContent(w)
	}
}

// Helper functions

func respondCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
