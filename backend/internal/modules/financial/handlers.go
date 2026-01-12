package financial

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterRoutes registers all financial endpoints
func RegisterRoutes(r chi.Router, service *Service) {
	r.Route("/programs/{programId}/financial", func(r chi.Router) {
		// Rate cards
		r.Route("/rate-cards", func(r chi.Router) {
			r.Post("/", handleCreateRateCard(service))
			r.Get("/", handleListRateCards(service))

			r.Route("/{rateCardId}", func(r chi.Router) {
				r.Get("/", handleGetRateCard(service))
				r.Put("/", handleUpdateRateCard(service))
				r.Delete("/", handleDeleteRateCard(service))
			})
		})

		// Invoices
		r.Route("/invoices", func(r chi.Router) {
			r.Get("/", handleListInvoices(service))
			r.Post("/process", handleProcessInvoice(service))

			r.Route("/{invoiceId}", func(r chi.Router) {
				r.Get("/", handleGetInvoice(service))
				r.Post("/approve", handleApproveInvoice(service))
				r.Post("/reject", handleRejectInvoice(service))
			})
		})

		// Variances
		r.Route("/variances", func(r chi.Router) {
			r.Get("/", handleListVariances(service))

			r.Route("/{varianceId}", func(r chi.Router) {
				r.Post("/dismiss", handleDismissVariance(service))
				r.Post("/resolve", handleResolveVariance(service))
			})
		})

		// Budget
		r.Route("/budget", func(r chi.Router) {
			r.Get("/status", handleBudgetStatus(service))
			r.Post("/categories", handleCreateBudgetCategory(service))
			r.Get("/categories", handleListBudgetCategories(service))

			r.Route("/categories/{categoryId}", func(r chi.Router) {
				r.Get("/", handleGetBudgetCategory(service))
				r.Put("/", handleUpdateBudgetCategory(service))
			})
		})
	})
}

// handleCreateRateCard creates a new rate card
func handleCreateRateCard(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req CreateRateCardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Override program ID from URL
		req.ProgramID = programID

		// TODO: Get created_by from JWT claims
		req.CreatedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")

		rateCardID, err := service.CreateRateCard(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]interface{}{
			"rate_card_id": rateCardID,
			"message":      "Rate card created successfully",
		})
	}
}

// handleListRateCards lists rate cards for a program
func handleListRateCards(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		// Parse pagination
		limit := parseIntQuery(r, "limit", 50)
		offset := parseIntQuery(r, "offset", 0)

		rateCards, err := service.ListRateCards(r.Context(), programID, limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"rate_cards": rateCards,
			"limit":      limit,
			"offset":     offset,
		})
	}
}

// handleGetRateCard retrieves a single rate card with items
func handleGetRateCard(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rateCardIDStr := chi.URLParam(r, "rateCardId")
		rateCardID, err := uuid.Parse(rateCardIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid rate card ID")
			return
		}

		rateCard, err := service.GetRateCardWithItems(r.Context(), rateCardID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Rate card not found")
			return
		}

		respondSuccess(w, rateCard)
	}
}

// handleUpdateRateCard updates a rate card
func handleUpdateRateCard(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rateCardIDStr := chi.URLParam(r, "rateCardId")
		rateCardID, err := uuid.Parse(rateCardIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid rate card ID")
			return
		}

		var rateCard RateCard
		if err := json.NewDecoder(r.Body).Decode(&rateCard); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		rateCard.RateCardID = rateCardID

		// TODO: Get updated_by from JWT claims
		rateCard.UpdatedBy = uuid.NullUUID{
			UUID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Valid: true,
		}

		if err := service.UpdateRateCard(r.Context(), &rateCard); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Rate card updated successfully",
		})
	}
}

// handleDeleteRateCard deletes a rate card
func handleDeleteRateCard(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rateCardIDStr := chi.URLParam(r, "rateCardId")
		rateCardID, err := uuid.Parse(rateCardIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid rate card ID")
			return
		}

		if err := service.DeleteRateCard(r.Context(), rateCardID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondNoContent(w)
	}
}

// handleListInvoices lists invoices with optional filters
func handleListInvoices(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		filter := InvoiceFilterRequest{
			ProgramID:        programID,
			ProcessingStatus: r.URL.Query().Get("processing_status"),
			PaymentStatus:    r.URL.Query().Get("payment_status"),
			VendorName:       r.URL.Query().Get("vendor_name"),
			Limit:            parseIntQuery(r, "limit", 50),
			Offset:           parseIntQuery(r, "offset", 0),
		}

		// Parse date filters
		if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
			if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
				filter.DateFrom = &dateFrom
			}
		}
		if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
			if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
				filter.DateTo = &dateTo
			}
		}

		invoices, err := service.ListInvoices(r.Context(), filter)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"invoices": invoices,
			"limit":    filter.Limit,
			"offset":   filter.Offset,
		})
	}
}

// handleGetInvoice retrieves an invoice with line items and variances
func handleGetInvoice(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceIDStr := chi.URLParam(r, "invoiceId")
		invoiceID, err := uuid.Parse(invoiceIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid invoice ID")
			return
		}

		invoice, err := service.GetInvoiceWithVariances(r.Context(), invoiceID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Invoice not found")
			return
		}

		respondSuccess(w, invoice)
	}
}

// handleProcessInvoice processes an invoice artifact
func handleProcessInvoice(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req struct {
			ArtifactID uuid.UUID `json:"artifact_id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// TODO: Get program context from program service
		// For now, use minimal context
		programContext := &ai.ProgramContext{
			ProgramName: "Program",
		}

		if err := service.ProcessInvoice(r.Context(), req.ArtifactID, programID, programContext); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Invoice processing queued",
		})
	}
}

// handleApproveInvoice approves an invoice
func handleApproveInvoice(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceIDStr := chi.URLParam(r, "invoiceId")
		invoiceID, err := uuid.Parse(invoiceIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid invoice ID")
			return
		}

		// TODO: Get approved_by from JWT claims
		approvedBy := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		if err := service.ApproveInvoice(r.Context(), invoiceID, approvedBy); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Invoice approved successfully",
		})
	}
}

// handleRejectInvoice rejects an invoice
func handleRejectInvoice(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceIDStr := chi.URLParam(r, "invoiceId")
		invoiceID, err := uuid.Parse(invoiceIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid invoice ID")
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := service.RejectInvoice(r.Context(), invoiceID, req.Reason); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Invoice rejected",
		})
	}
}

// handleListVariances lists variances for a program
func handleListVariances(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		severityFilter := r.URL.Query().Get("severity")

		variances, err := service.GetVariancesByProgram(r.Context(), programID, severityFilter)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"variances": variances,
		})
	}
}

// handleDismissVariance dismisses a variance
func handleDismissVariance(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		varianceIDStr := chi.URLParam(r, "varianceId")
		varianceID, err := uuid.Parse(varianceIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid variance ID")
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// TODO: Get dismissed_by from JWT claims
		dismissedBy := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		if err := service.DismissVariance(r.Context(), varianceID, dismissedBy, req.Reason); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Variance dismissed",
		})
	}
}

// handleResolveVariance resolves a variance
func handleResolveVariance(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		varianceIDStr := chi.URLParam(r, "varianceId")
		varianceID, err := uuid.Parse(varianceIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid variance ID")
			return
		}

		var req struct {
			Notes string `json:"notes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// TODO: Get resolved_by from JWT claims
		resolvedBy := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		if err := service.ResolveVariance(r.Context(), varianceID, resolvedBy, req.Notes); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Variance resolved",
		})
	}
}

// handleBudgetStatus returns budget status for a program
func handleBudgetStatus(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		fiscalYear := parseIntQuery(r, "fiscal_year", time.Now().Year())

		status, err := service.GetBudgetStatus(r.Context(), programID, fiscalYear)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, status)
	}
}

// handleCreateBudgetCategory creates a budget category
func handleCreateBudgetCategory(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var category BudgetCategory
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		category.ProgramID = programID

		if err := service.CreateBudgetCategory(r.Context(), &category); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]interface{}{
			"category_id": category.CategoryID,
			"message":     "Budget category created successfully",
		})
	}
}

// handleListBudgetCategories lists budget categories
func handleListBudgetCategories(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		fiscalYear := parseIntQuery(r, "fiscal_year", time.Now().Year())

		categories, err := service.ListBudgetCategories(r.Context(), programID, fiscalYear)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"categories":  categories,
			"fiscal_year": fiscalYear,
		})
	}
}

// handleGetBudgetCategory retrieves a budget category
func handleGetBudgetCategory(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		categoryIDStr := chi.URLParam(r, "categoryId")
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid category ID")
			return
		}

		category, err := service.GetBudgetCategory(r.Context(), categoryID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Budget category not found")
			return
		}

		respondSuccess(w, category)
	}
}

// handleUpdateBudgetCategory updates a budget category
func handleUpdateBudgetCategory(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		categoryIDStr := chi.URLParam(r, "categoryId")
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid category ID")
			return
		}

		var category BudgetCategory
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		category.CategoryID = categoryID

		if err := service.UpdateBudgetCategory(r.Context(), &category); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message": "Budget category updated successfully",
		})
	}
}

// Helper functions

func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	if valueStr := r.URL.Query().Get(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func respondSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func respondCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
