package risk

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cerberus/backend/internal/platform/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterRoutes registers all risk endpoints
func RegisterRoutes(r chi.Router, service *Service, conversationService *ConversationService, authRepo *auth.Repository) {
	r.Route("/programs/{programId}/risks", func(r chi.Router) {
		r.Use(auth.RequireProgramAccess(auth.RoleViewer, authRepo))
		// Risk CRUD
		r.Post("/", handleCreateRisk(service))
		r.Get("/", handleListRisks(service))

		// Risk suggestions
		r.Get("/suggestions", handleListSuggestions(service))
		r.Post("/suggestions/{suggestionId}/approve", handleApproveSuggestion(service))
		r.Post("/suggestions/{suggestionId}/dismiss", handleDismissSuggestion(service))

		// Individual risk operations
		r.Route("/{riskId}", func(r chi.Router) {
			r.Get("/", handleGetRisk(service))
			r.Get("/context", handleGetRiskWithContext(service))
			r.Patch("/", handleUpdateRisk(service))
			r.Delete("/", handleDeleteRisk(service))

			// Mitigations
			r.Post("/mitigations", handleAddMitigation(service))
			r.Patch("/mitigations/{mitigationId}", handleUpdateMitigation(service))
			r.Delete("/mitigations/{mitigationId}", handleDeleteMitigation(service))

			// Artifact linking
			r.Post("/artifacts", handleLinkArtifact(service))
			r.Delete("/artifacts/{linkId}", handleUnlinkArtifact(service))
			r.Get("/artifacts", handleGetLinkedArtifacts(service))

			// Enrichments
			r.Get("/enrichments", handleGetEnrichments(service))
			r.Post("/enrichments/{enrichmentId}/accept", handleAcceptEnrichment(service))
			r.Post("/enrichments/{enrichmentId}/reject", handleRejectEnrichment(service))

			// Conversation threads
			r.Post("/threads", handleCreateThread(conversationService))
			r.Get("/threads", handleListThreads(conversationService))
			r.Route("/threads/{threadId}", func(r chi.Router) {
				r.Get("/", handleGetThread(conversationService))
				r.Post("/resolve", handleResolveThread(conversationService))
				r.Delete("/", handleDeleteThread(conversationService))

				// Thread messages
				r.Post("/messages", handleAddMessage(conversationService))
				r.Get("/messages", handleGetMessages(conversationService))
				r.Delete("/messages/{messageId}", handleDeleteMessage(conversationService))
			})
		})
	})

	// Artifact-centric view
	r.Route("/artifacts/{artifactId}/risks", func(r chi.Router) {
		r.Get("/", handleGetRisksByArtifact(service))
	})
}

// handleCreateRisk handles risk creation
func handleCreateRisk(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req CreateRiskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Set program ID from URL
		req.ProgramID = programID

		// TODO: Get createdBy from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		riskID, err := service.CreateRisk(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"risk_id": riskID.String(),
			"message": "Risk created successfully",
		})
	}
}

// handleGetRisk handles retrieving a single risk
func handleGetRisk(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		risk, err := service.GetRisk(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}

		respondSuccess(w, risk)
	}
}

// handleGetRiskWithContext handles retrieving a risk with all related data
func handleGetRiskWithContext(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		riskWithMetadata, err := service.GetRiskWithContext(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}

		respondSuccess(w, riskWithMetadata)
	}
}

// handleListRisks handles listing risks with filters
func handleListRisks(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		// Parse query parameters
		filter := RiskFilterRequest{
			ProgramID: programID,
			Status:    r.URL.Query().Get("status"),
			Category:  r.URL.Query().Get("category"),
			Severity:  r.URL.Query().Get("severity"),
			Limit:     parseIntParam(r, "limit", 50),
			Offset:    parseIntParam(r, "offset", 0),
		}

		// Parse owner filter if provided
		if ownerIDStr := r.URL.Query().Get("owner_user_id"); ownerIDStr != "" {
			ownerID, err := uuid.Parse(ownerIDStr)
			if err == nil {
				filter.OwnerUserID = &ownerID
			}
		}

		// Check if suggestions should be included
		includeSuggestions := r.URL.Query().Get("include_suggestions") == "true"

		if includeSuggestions {
			// Use new endpoint that includes suggestions
			response, err := service.ListRisksWithSuggestions(r.Context(), filter, true)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			respondSuccess(w, response)
		} else {
			// Use existing endpoint (backward compatibility)
			risks, err := service.ListRisks(r.Context(), filter)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			respondSuccess(w, map[string]interface{}{
				"risks": risks,
			})
		}
	}
}

// handleUpdateRisk handles updating a risk
func handleUpdateRisk(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		var req UpdateRiskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		err = service.UpdateRisk(r.Context(), riskID, req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Risk updated successfully",
		})
	}
}

// handleDeleteRisk handles deleting a risk
func handleDeleteRisk(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		err = service.DeleteRisk(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Risk deleted successfully",
		})
	}
}

// handleListSuggestions handles listing risk suggestions
func handleListSuggestions(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		includeProcessed := r.URL.Query().Get("include_processed") == "true"

		suggestions, err := service.ListSuggestions(r.Context(), programID, includeProcessed)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"suggestions": suggestions,
		})
	}
}

// handleApproveSuggestion handles approving a risk suggestion
func handleApproveSuggestion(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		suggestionIDStr := chi.URLParam(r, "suggestionId")
		suggestionID, err := uuid.Parse(suggestionIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid suggestion ID")
			return
		}

		var req ApproveSuggestionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.SuggestionID = suggestionID

		// TODO: Get approvedBy from JWT claims
		req.ApprovedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		riskID, err := service.ApproveSuggestion(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"risk_id": riskID.String(),
			"message": "Suggestion approved and risk created",
		})
	}
}

// handleDismissSuggestion handles dismissing a risk suggestion
func handleDismissSuggestion(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		suggestionIDStr := chi.URLParam(r, "suggestionId")
		suggestionID, err := uuid.Parse(suggestionIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid suggestion ID")
			return
		}

		var req DismissSuggestionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.SuggestionID = suggestionID

		// TODO: Get dismissedBy from JWT claims
		req.DismissedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		err = service.DismissSuggestion(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Suggestion dismissed",
		})
	}
}

// handleAddMitigation handles adding a mitigation
func handleAddMitigation(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		var req CreateMitigationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.RiskID = riskID

		// TODO: Get createdBy from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		mitigationID, err := service.AddMitigation(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"mitigation_id": mitigationID.String(),
			"message":       "Mitigation created successfully",
		})
	}
}

// handleUpdateMitigation handles updating a mitigation
func handleUpdateMitigation(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mitigationIDStr := chi.URLParam(r, "mitigationId")
		mitigationID, err := uuid.Parse(mitigationIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid mitigation ID")
			return
		}

		var req UpdateMitigationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		err = service.UpdateMitigation(r.Context(), mitigationID, req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Mitigation updated successfully",
		})
	}
}

// handleDeleteMitigation handles deleting a mitigation
func handleDeleteMitigation(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mitigationIDStr := chi.URLParam(r, "mitigationId")
		mitigationID, err := uuid.Parse(mitigationIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid mitigation ID")
			return
		}

		err = service.DeleteMitigation(r.Context(), mitigationID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Mitigation deleted successfully",
		})
	}
}

// handleLinkArtifact handles linking an artifact to a risk
func handleLinkArtifact(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		var req LinkArtifactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.RiskID = riskID

		// TODO: Get createdBy from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		linkID, err := service.LinkArtifact(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"link_id": linkID.String(),
			"message": "Artifact linked successfully",
		})
	}
}

// handleUnlinkArtifact handles unlinking an artifact from a risk
func handleUnlinkArtifact(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkIDStr := chi.URLParam(r, "linkId")
		linkID, err := uuid.Parse(linkIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid link ID")
			return
		}

		err = service.UnlinkArtifact(r.Context(), linkID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Artifact unlinked successfully",
		})
	}
}

// handleGetLinkedArtifacts handles retrieving artifacts linked to a risk
func handleGetLinkedArtifacts(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		links, err := service.GetLinkedArtifacts(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, links)
	}
}

// handleGetRisksByArtifact handles retrieving risks associated with an artifact
func handleGetRisksByArtifact(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		risks, err := service.GetRisksByArtifact(r.Context(), artifactID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, risks)
	}
}

// Conversation handlers

// handleCreateThread handles creating a conversation thread
func handleCreateThread(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		var req CreateThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.RiskID = riskID

		// TODO: Get createdBy from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		threadID, err := service.CreateThread(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"thread_id": threadID.String(),
			"message":   "Thread created successfully",
		})
	}
}

// handleGetThread handles retrieving a thread with messages
func handleGetThread(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadIDStr := chi.URLParam(r, "threadId")
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid thread ID")
			return
		}

		thread, err := service.GetThreadWithMessages(r.Context(), threadID)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, thread)
	}
}

// handleListThreads handles listing threads for a risk
func handleListThreads(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		threads, err := service.ListThreads(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, threads)
	}
}

// handleResolveThread handles marking a thread as resolved
func handleResolveThread(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadIDStr := chi.URLParam(r, "threadId")
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid thread ID")
			return
		}

		// TODO: Get resolvedBy from JWT claims
		resolvedBy := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		err = service.ResolveThread(r.Context(), threadID, resolvedBy)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Thread resolved successfully",
		})
	}
}

// handleDeleteThread handles deleting a thread
func handleDeleteThread(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadIDStr := chi.URLParam(r, "threadId")
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid thread ID")
			return
		}

		err = service.DeleteThread(r.Context(), threadID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Thread deleted successfully",
		})
	}
}

// handleAddMessage handles adding a message to a thread
func handleAddMessage(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadIDStr := chi.URLParam(r, "threadId")
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid thread ID")
			return
		}

		var req CreateMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		req.ThreadID = threadID

		// TODO: Get createdBy from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		messageID, err := service.AddMessage(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"message_id": messageID.String(),
			"message":    "Message added successfully",
		})
	}
}

// handleGetMessages handles retrieving messages for a thread
func handleGetMessages(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadIDStr := chi.URLParam(r, "threadId")
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid thread ID")
			return
		}

		messages, err := service.GetMessages(r.Context(), threadID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, messages)
	}
}

// handleDeleteMessage handles deleting a message
func handleDeleteMessage(service *ConversationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messageIDStr := chi.URLParam(r, "messageId")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid message ID")
			return
		}

		err = service.DeleteMessage(r.Context(), messageID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Message deleted successfully",
		})
	}
}

// handleGetEnrichments retrieves enrichments for a risk
func handleGetEnrichments(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		enrichments, err := service.GetEnrichments(r.Context(), riskID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"enrichments": enrichments,
		})
	}
}

// handleAcceptEnrichment accepts an enrichment as relevant
func handleAcceptEnrichment(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		enrichmentIDStr := chi.URLParam(r, "enrichmentId")
		enrichmentID, err := uuid.Parse(enrichmentIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid enrichment ID")
			return
		}

		// Get reviewed_by from request body or auth context
		var req struct {
			ReviewedBy uuid.UUID `json:"reviewed_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		err = service.AcceptEnrichment(r.Context(), riskID, enrichmentID, req.ReviewedBy)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]string{
			"message": "Enrichment accepted successfully",
		})
	}
}

// handleRejectEnrichment rejects an enrichment as not relevant
func handleRejectEnrichment(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		riskIDStr := chi.URLParam(r, "riskId")
		riskID, err := uuid.Parse(riskIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid risk ID")
			return
		}

		enrichmentIDStr := chi.URLParam(r, "enrichmentId")
		enrichmentID, err := uuid.Parse(enrichmentIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid enrichment ID")
			return
		}

		// Get reviewed_by from request body or auth context
		var req struct {
			ReviewedBy uuid.UUID `json:"reviewed_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		err = service.RejectEnrichment(r.Context(), riskID, enrichmentID, req.ReviewedBy)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]string{
			"message": "Enrichment rejected successfully",
		})
	}
}

// Helper functions

func parseIntParam(r *http.Request, key string, defaultValue int) int {
	if str := r.URL.Query().Get(key); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return val
		}
	}
	return defaultValue
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondSuccess(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

func respondCreated(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"data": data,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
