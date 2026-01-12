// Package financial provides service layer for financial module management.
// This includes rate cards, invoice processing, variance detection, and budget tracking.
package financial

import (
	"context"
	"fmt"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/google/uuid"
)

// Service handles business logic for financial module
type Service struct {
	repo     RepositoryInterface
	storage  storage.Storage
	analyzer *InvoiceAnalyzer
}

// NewService creates a new financial service
func NewService(repo *Repository, stor storage.Storage, aiClient *ai.Client) *Service {
	return &Service{
		repo:     repo,
		storage:  stor,
		analyzer: NewInvoiceAnalyzer(aiClient, repo),
	}
}

// NewServiceWithMocks creates a service with mock dependencies (useful for testing)
func NewServiceWithMocks(repo RepositoryInterface, stor storage.Storage, analyzer *InvoiceAnalyzer) *Service {
	return &Service{
		repo:     repo,
		storage:  stor,
		analyzer: analyzer,
	}
}

// CreateRateCard creates a new rate card with items
func (s *Service) CreateRateCard(ctx context.Context, req CreateRateCardRequest) (uuid.UUID, error) {
	// Validate request
	if req.ProgramID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("program_id is required")
	}
	if req.Name == "" {
		return uuid.Nil, fmt.Errorf("rate card name is required")
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if len(req.Items) == 0 {
		return uuid.Nil, fmt.Errorf("at least one rate card item is required")
	}

	// Validate items
	for i, item := range req.Items {
		if item.PersonName == "" && item.RoleTitle == "" {
			return uuid.Nil, fmt.Errorf("item %d must have either person_name or role_title", i)
		}
		if item.RateType == "" {
			return uuid.Nil, fmt.Errorf("item %d: rate_type is required", i)
		}
		if item.RateAmount <= 0 {
			return uuid.Nil, fmt.Errorf("item %d: rate_amount must be positive", i)
		}
		if item.Currency == "" {
			req.Items[i].Currency = req.Currency // Inherit from rate card
		}
	}

	// Create rate card
	rateCardID := uuid.New()
	rateCard := &RateCard{
		RateCardID:         rateCardID,
		ProgramID:          req.ProgramID,
		Name:               req.Name,
		Description:        toNullString(req.Description),
		EffectiveStartDate: req.EffectiveStartDate,
		Currency:           req.Currency,
		IsActive:           true,
		CreatedAt:          time.Now(),
		CreatedBy:          req.CreatedBy,
		UpdatedAt:          time.Now(),
	}

	if req.EffectiveEndDate != nil {
		rateCard.EffectiveEndDate = toNullTime(req.EffectiveEndDate.Format("2006-01-02"))
	}

	err := s.repo.CreateRateCard(ctx, rateCard)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create rate card: %w", err)
	}

	// Create rate card items
	var items []RateCardItem
	for _, itemReq := range req.Items {
		item := RateCardItem{
			ItemID:       uuid.New(),
			RateCardID:   rateCardID,
			PersonName:   toNullString(itemReq.PersonName),
			RoleTitle:    toNullString(itemReq.RoleTitle),
			SeniorityLevel: toNullString(itemReq.SeniorityLevel),
			RateType:     itemReq.RateType,
			RateAmount:   itemReq.RateAmount,
			Currency:     itemReq.Currency,
			Notes:        toNullString(itemReq.Notes),
			CreatedAt:    time.Now(),
		}

		if itemReq.ExpectedHoursPerWeek != nil {
			item.ExpectedHoursPerWeek = toNullFloat64(*itemReq.ExpectedHoursPerWeek)
		}
		if itemReq.ExpectedHoursPerMonth != nil {
			item.ExpectedHoursPerMonth = toNullFloat64(*itemReq.ExpectedHoursPerMonth)
		}

		items = append(items, item)
	}

	err = s.repo.CreateRateCardItems(ctx, items)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create rate card items: %w", err)
	}

	return rateCardID, nil
}

// GetRateCard retrieves a rate card by ID
func (s *Service) GetRateCard(ctx context.Context, rateCardID uuid.UUID) (*RateCard, error) {
	if rateCardID == uuid.Nil {
		return nil, fmt.Errorf("rate_card_id is required")
	}

	rateCard, err := s.repo.GetRateCardByID(ctx, rateCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card: %w", err)
	}

	return rateCard, nil
}

// GetRateCardWithItems retrieves a rate card with all its items
func (s *Service) GetRateCardWithItems(ctx context.Context, rateCardID uuid.UUID) (*RateCardWithItems, error) {
	if rateCardID == uuid.Nil {
		return nil, fmt.Errorf("rate_card_id is required")
	}

	rateCardWithItems, err := s.repo.GetRateCardWithItems(ctx, rateCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card with items: %w", err)
	}

	return rateCardWithItems, nil
}

// ListRateCards retrieves rate cards for a program
func (s *Service) ListRateCards(ctx context.Context, programID uuid.UUID, limit, offset int) ([]RateCard, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate pagination
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	rateCards, err := s.repo.ListRateCardsByProgram(ctx, programID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate cards: %w", err)
	}

	return rateCards, nil
}

// UpdateRateCard updates an existing rate card
func (s *Service) UpdateRateCard(ctx context.Context, rateCard *RateCard) error {
	if rateCard.RateCardID == uuid.Nil {
		return fmt.Errorf("rate_card_id is required")
	}

	rateCard.UpdatedAt = time.Now()

	err := s.repo.UpdateRateCard(ctx, rateCard)
	if err != nil {
		return fmt.Errorf("failed to update rate card: %w", err)
	}

	return nil
}

// DeleteRateCard soft-deletes a rate card
func (s *Service) DeleteRateCard(ctx context.Context, rateCardID uuid.UUID) error {
	if rateCardID == uuid.Nil {
		return fmt.Errorf("rate_card_id is required")
	}

	err := s.repo.DeleteRateCard(ctx, rateCardID)
	if err != nil {
		return fmt.Errorf("failed to delete rate card: %w", err)
	}

	return nil
}

// ProcessInvoice analyzes an invoice artifact and detects variances
func (s *Service) ProcessInvoice(ctx context.Context, artifactID uuid.UUID, programID uuid.UUID, programContext *ai.ProgramContext) error {
	if artifactID == uuid.Nil {
		return fmt.Errorf("artifact_id is required")
	}
	if programID == uuid.Nil {
		return fmt.Errorf("program_id is required")
	}

	// This would typically get the artifact content from the artifacts service
	// For now, we'll assume the artifact content is passed or retrieved separately
	// In production, this would integrate with the artifacts module:
	// artifact, err := s.artifactsService.GetArtifact(ctx, artifactID)

	// Placeholder: In production implementation, retrieve artifact content
	return fmt.Errorf("ProcessInvoice requires integration with artifacts service to retrieve content")
}

// ProcessInvoiceFromContent analyzes invoice content directly
func (s *Service) ProcessInvoiceFromContent(ctx context.Context, artifactID uuid.UUID, artifactContent string, programID uuid.UUID, programContext *ai.ProgramContext) error {
	if artifactID == uuid.Nil {
		return fmt.Errorf("artifact_id is required")
	}
	if programID == uuid.Nil {
		return fmt.Errorf("program_id is required")
	}
	if artifactContent == "" {
		return fmt.Errorf("artifact content is required")
	}

	// Process invoice using analyzer
	err := s.analyzer.ProcessInvoice(ctx, artifactID, artifactContent, programID, programContext)
	if err != nil {
		return fmt.Errorf("failed to process invoice: %w", err)
	}

	return nil
}

// GetInvoice retrieves an invoice by ID
func (s *Service) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error) {
	if invoiceID == uuid.Nil {
		return nil, fmt.Errorf("invoice_id is required")
	}

	invoice, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	return invoice, nil
}

// GetInvoiceWithVariances retrieves an invoice with line items and variances
func (s *Service) GetInvoiceWithVariances(ctx context.Context, invoiceID uuid.UUID) (*InvoiceWithMetadata, error) {
	if invoiceID == uuid.Nil {
		return nil, fmt.Errorf("invoice_id is required")
	}

	invoiceWithMetadata, err := s.repo.GetInvoiceWithLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice with metadata: %w", err)
	}

	return invoiceWithMetadata, nil
}

// ListInvoices retrieves invoices with optional filters
func (s *Service) ListInvoices(ctx context.Context, filter InvoiceFilterRequest) ([]Invoice, error) {
	if filter.ProgramID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate pagination
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Validate status filters
	if filter.ProcessingStatus != "" {
		validStatuses := map[string]bool{
			"pending":    true,
			"processing": true,
			"validated":  true,
			"approved":   true,
			"rejected":   true,
		}
		if !validStatuses[filter.ProcessingStatus] {
			return nil, fmt.Errorf("invalid processing_status: %s", filter.ProcessingStatus)
		}
	}

	if filter.PaymentStatus != "" {
		validStatuses := map[string]bool{
			"unpaid":  true,
			"paid":    true,
			"partial": true,
			"overdue": true,
		}
		if !validStatuses[filter.PaymentStatus] {
			return nil, fmt.Errorf("invalid payment_status: %s", filter.PaymentStatus)
		}
	}

	invoices, err := s.repo.ListInvoices(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	return invoices, nil
}

// ApproveInvoice marks an invoice as approved
func (s *Service) ApproveInvoice(ctx context.Context, invoiceID uuid.UUID, approvedBy uuid.UUID) error {
	if invoiceID == uuid.Nil {
		return fmt.Errorf("invoice_id is required")
	}
	if approvedBy == uuid.Nil {
		return fmt.Errorf("approved_by is required")
	}

	// Get invoice
	invoice, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Update status
	invoice.ProcessingStatus = "approved"
	invoice.ApprovedBy = uuid.NullUUID{UUID: approvedBy, Valid: true}
	invoice.ApprovedAt = toNullTime(time.Now().Format("2006-01-02"))

	err = s.repo.UpdateInvoice(ctx, invoice)
	if err != nil {
		return fmt.Errorf("failed to approve invoice: %w", err)
	}

	return nil
}

// RejectInvoice marks an invoice as rejected
func (s *Service) RejectInvoice(ctx context.Context, invoiceID uuid.UUID, reason string) error {
	if invoiceID == uuid.Nil {
		return fmt.Errorf("invoice_id is required")
	}
	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	// Get invoice
	invoice, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Update status
	invoice.ProcessingStatus = "rejected"
	invoice.RejectedReason = toNullString(reason)

	err = s.repo.UpdateInvoice(ctx, invoice)
	if err != nil {
		return fmt.Errorf("failed to reject invoice: %w", err)
	}

	return nil
}

// GetVariancesByProgram retrieves variances for a program
func (s *Service) GetVariancesByProgram(ctx context.Context, programID uuid.UUID, severityFilter string) ([]FinancialVariance, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate severity filter
	if severityFilter != "" {
		validSeverities := map[string]bool{
			"low":      true,
			"medium":   true,
			"high":     true,
			"critical": true,
		}
		if !validSeverities[severityFilter] {
			return nil, fmt.Errorf("invalid severity filter: %s", severityFilter)
		}
	}

	variances, err := s.repo.GetVariancesByProgram(ctx, programID, severityFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get variances: %w", err)
	}

	return variances, nil
}

// DismissVariance dismisses a variance
func (s *Service) DismissVariance(ctx context.Context, varianceID uuid.UUID, dismissedBy uuid.UUID, reason string) error {
	if varianceID == uuid.Nil {
		return fmt.Errorf("variance_id is required")
	}
	if dismissedBy == uuid.Nil {
		return fmt.Errorf("dismissed_by is required")
	}
	if reason == "" {
		return fmt.Errorf("dismissal reason is required")
	}

	err := s.repo.DismissVariance(ctx, varianceID, dismissedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to dismiss variance: %w", err)
	}

	return nil
}

// ResolveVariance resolves a variance
func (s *Service) ResolveVariance(ctx context.Context, varianceID uuid.UUID, resolvedBy uuid.UUID, notes string) error {
	if varianceID == uuid.Nil {
		return fmt.Errorf("variance_id is required")
	}
	if resolvedBy == uuid.Nil {
		return fmt.Errorf("resolved_by is required")
	}
	if notes == "" {
		return fmt.Errorf("resolution notes are required")
	}

	err := s.repo.ResolveVariance(ctx, varianceID, resolvedBy, notes)
	if err != nil {
		return fmt.Errorf("failed to resolve variance: %w", err)
	}

	return nil
}

// CreateBudgetCategory creates a new budget category
func (s *Service) CreateBudgetCategory(ctx context.Context, category *BudgetCategory) error {
	if category.ProgramID == uuid.Nil {
		return fmt.Errorf("program_id is required")
	}
	if category.CategoryName == "" {
		return fmt.Errorf("category_name is required")
	}
	if category.BudgetedAmount <= 0 {
		return fmt.Errorf("budgeted_amount must be positive")
	}
	if category.FiscalYear <= 0 {
		return fmt.Errorf("fiscal_year is required")
	}

	category.CategoryID = uuid.New()
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	if category.Currency == "" {
		category.Currency = "USD"
	}

	err := s.repo.CreateBudgetCategory(ctx, category)
	if err != nil {
		return fmt.Errorf("failed to create budget category: %w", err)
	}

	return nil
}

// GetBudgetCategory retrieves a budget category by ID
func (s *Service) GetBudgetCategory(ctx context.Context, categoryID uuid.UUID) (*BudgetCategory, error) {
	if categoryID == uuid.Nil {
		return nil, fmt.Errorf("category_id is required")
	}

	category, err := s.repo.GetBudgetCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget category: %w", err)
	}

	return category, nil
}

// ListBudgetCategories retrieves budget categories for a program
func (s *Service) ListBudgetCategories(ctx context.Context, programID uuid.UUID, fiscalYear int) ([]BudgetCategory, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}
	if fiscalYear <= 0 {
		return nil, fmt.Errorf("fiscal_year is required")
	}

	categories, err := s.repo.ListBudgetCategories(ctx, programID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("failed to list budget categories: %w", err)
	}

	return categories, nil
}

// UpdateBudgetCategory updates a budget category
func (s *Service) UpdateBudgetCategory(ctx context.Context, category *BudgetCategory) error {
	if category.CategoryID == uuid.Nil {
		return fmt.Errorf("category_id is required")
	}

	err := s.repo.UpdateBudgetCategory(ctx, category)
	if err != nil {
		return fmt.Errorf("failed to update budget category: %w", err)
	}

	return nil
}

// GetBudgetStatus calculates budget status for a program
func (s *Service) GetBudgetStatus(ctx context.Context, programID uuid.UUID, fiscalYear int) (map[string]interface{}, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}
	if fiscalYear <= 0 {
		return nil, fmt.Errorf("fiscal_year is required")
	}

	// Get all budget categories
	categories, err := s.repo.ListBudgetCategories(ctx, programID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget categories: %w", err)
	}

	// Calculate totals
	var totalBudgeted float64
	var totalActual float64
	var totalCommitted float64

	categorySummary := make([]map[string]interface{}, len(categories))
	for i, cat := range categories {
		totalBudgeted += cat.BudgetedAmount
		totalActual += cat.ActualSpend
		totalCommitted += cat.CommittedSpend

		categorySummary[i] = map[string]interface{}{
			"category_id":         cat.CategoryID,
			"category_name":       cat.CategoryName,
			"budgeted_amount":     cat.BudgetedAmount,
			"actual_spend":        cat.ActualSpend,
			"committed_spend":     cat.CommittedSpend,
			"remaining_budget":    cat.BudgetedAmount - cat.ActualSpend - cat.CommittedSpend,
			"variance_amount":     cat.VarianceAmount,
			"variance_percentage": cat.VariancePercentage,
		}
	}

	totalRemaining := totalBudgeted - totalActual - totalCommitted
	totalVariance := totalActual - totalBudgeted
	totalVariancePct := 0.0
	if totalBudgeted > 0 {
		totalVariancePct = (totalVariance / totalBudgeted) * 100
	}

	status := map[string]interface{}{
		"program_id":           programID,
		"fiscal_year":          fiscalYear,
		"total_budgeted":       totalBudgeted,
		"total_actual_spend":   totalActual,
		"total_committed":      totalCommitted,
		"total_remaining":      totalRemaining,
		"total_variance":       totalVariance,
		"variance_percentage":  totalVariancePct,
		"categories":           categorySummary,
		"budget_health":        determineBudgetHealth(totalVariancePct),
	}

	return status, nil
}

// Helper function to determine budget health
func determineBudgetHealth(variancePct float64) string {
	switch {
	case variancePct < -10:
		return "under_budget"
	case variancePct > 10:
		return "over_budget"
	default:
		return "on_track"
	}
}
