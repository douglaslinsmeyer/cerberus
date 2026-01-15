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
		r.Get("/suggestions", handleGetSuggestions(repo))
		r.Get("/suggestions/grouped", handleGetGroupedSuggestions(repo))
		r.Post("/suggestions/refresh-grouping", handleRefreshGrouping(repo))

		r.Route("/suggestions/groups/{groupId}", func(r chi.Router) {
			r.Post("/confirm", handleConfirmMergeGroup(repo))
			r.Post("/reject", handleRejectMergeGroup(repo))
			r.Post("/members", handleModifyGroupMembers(repo))
		})

		r.Route("/{stakeholderId}", func(r chi.Router) {
			r.Get("/", handleGetStakeholder(repo))
			r.Put("/", handleUpdateStakeholder(repo))
			r.Delete("/", handleDeleteStakeholder(repo))
			r.Get("/artifacts", handleGetLinkedArtifacts(repo))
		})
	})

	// Person linking endpoint
	r.Post("/programs/{programId}/persons/{personId}/link", handleLinkPerson(repo))
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

// handleGetSuggestions returns person mentions that haven't been linked to stakeholders
func handleGetSuggestions(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		suggestions, err := repo.GetSuggestions(r.Context(), programID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"suggestions": suggestions,
		})
	}
}

// handleLinkPerson links a person mention to a stakeholder
func handleLinkPerson(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		_, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		personIDStr := chi.URLParam(r, "personId")
		personID, err := uuid.Parse(personIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid person ID")
			return
		}

		var req LinkPersonRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := repo.LinkPersonToStakeholder(r.Context(), personID, req.StakeholderID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Person successfully linked to stakeholder",
		})
	}
}

// handleGetGroupedSuggestions retrieves grouped person suggestions
func handleGetGroupedSuggestions(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		groups, err := repo.GetGroupedSuggestions(r.Context(), programID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"groups": groups,
		})
	}
}

// handleConfirmMergeGroup confirms a merge group and optionally creates a stakeholder
func handleConfirmMergeGroup(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := chi.URLParam(r, "groupId")
		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid group ID")
			return
		}

		var req ConfirmMergeGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		if req.SelectedName == "" {
			respondError(w, http.StatusBadRequest, "selected_name is required")
			return
		}

		stakeholder, err := repo.ConfirmMergeGroup(r.Context(), groupID, req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response := map[string]interface{}{
			"message": "Merge group confirmed successfully",
		}

		if stakeholder != nil {
			response["stakeholder"] = stakeholder
		}

		respondSuccess(w, response)
	}
}

// handleRejectMergeGroup marks a merge group as rejected
func handleRejectMergeGroup(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := chi.URLParam(r, "groupId")
		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid group ID")
			return
		}

		if err := repo.RejectMergeGroup(r.Context(), groupID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Merge group rejected successfully",
		})
	}
}

// handleModifyGroupMembers adds or removes persons from a merge group
func handleModifyGroupMembers(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := chi.URLParam(r, "groupId")
		groupID, err := uuid.Parse(groupIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid group ID")
			return
		}

		var req ModifyGroupMembersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := repo.ModifyGroupMembers(r.Context(), groupID, req); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Group members modified successfully",
		})
	}
}

// handleRefreshGrouping triggers re-run of grouping algorithm
func handleRefreshGrouping(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		if err := repo.GroupPersonSuggestions(r.Context(), programID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Person grouping refreshed successfully",
		})
	}
}

// handleGetLinkedArtifacts retrieves all artifacts where a stakeholder is mentioned
func handleGetLinkedArtifacts(repo *StakeholderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stakeholderIDStr := chi.URLParam(r, "stakeholderId")
		stakeholderID, err := uuid.Parse(stakeholderIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid stakeholder ID")
			return
		}

		artifacts, err := repo.GetLinkedArtifacts(r.Context(), stakeholderID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"artifacts": artifacts,
		})
	}
}
