package financial

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RepositoryInterface defines methods for financial data access
type RepositoryInterface interface {
	// Rate Cards
	CreateRateCard(ctx context.Context, rateCard *RateCard) error
	GetRateCardByID(ctx context.Context, rateCardID uuid.UUID) (*RateCard, error)
	ListRateCardsByProgram(ctx context.Context, programID uuid.UUID, limit, offset int) ([]RateCard, error)
	UpdateRateCard(ctx context.Context, rateCard *RateCard) error
	DeleteRateCard(ctx context.Context, rateCardID uuid.UUID) error
	GetActiveRateCards(ctx context.Context, programID uuid.UUID) ([]RateCard, error)

	// Rate Card Items
	CreateRateCardItems(ctx context.Context, items []RateCardItem) error
	GetRateCardItems(ctx context.Context, rateCardID uuid.UUID) ([]RateCardItem, error)
	GetRateCardItemByPersonName(ctx context.Context, rateCardID uuid.UUID, personName string) (*RateCardItem, error)
	GetRateCardItemByRole(ctx context.Context, rateCardID uuid.UUID, roleTitle string) (*RateCardItem, error)
	DeleteRateCardItems(ctx context.Context, rateCardID uuid.UUID) error

	// Invoices
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, filter InvoiceFilterRequest) ([]Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *Invoice) error
	DeleteInvoice(ctx context.Context, invoiceID uuid.UUID) error
	GetInvoiceByArtifactID(ctx context.Context, artifactID uuid.UUID) (*Invoice, error)

	// Invoice Line Items
	SaveLineItems(ctx context.Context, lineItems []InvoiceLineItem) error
	GetLineItems(ctx context.Context, invoiceID uuid.UUID) ([]InvoiceLineItem, error)
	UpdateLineItem(ctx context.Context, lineItem *InvoiceLineItem) error

	// Composed Queries
	GetInvoiceWithLineItems(ctx context.Context, invoiceID uuid.UUID) (*InvoiceWithMetadata, error)
	GetRateCardWithItems(ctx context.Context, rateCardID uuid.UUID) (*RateCardWithItems, error)

	// Budget Categories
	CreateBudgetCategory(ctx context.Context, category *BudgetCategory) error
	GetBudgetCategoryByID(ctx context.Context, categoryID uuid.UUID) (*BudgetCategory, error)
	ListBudgetCategories(ctx context.Context, programID uuid.UUID, fiscalYear int) ([]BudgetCategory, error)
	UpdateBudgetCategory(ctx context.Context, category *BudgetCategory) error
	DeleteBudgetCategory(ctx context.Context, categoryID uuid.UUID) error

	// Financial Variances
	SaveVariances(ctx context.Context, variances []FinancialVariance) error
	GetVariances(ctx context.Context, invoiceID uuid.UUID) ([]FinancialVariance, error)
	GetVariancesByProgram(ctx context.Context, programID uuid.UUID, severityFilter string) ([]FinancialVariance, error)
	DismissVariance(ctx context.Context, varianceID uuid.UUID, dismissedBy uuid.UUID, reason string) error
	ResolveVariance(ctx context.Context, varianceID uuid.UUID, resolvedBy uuid.UUID, notes string) error

	// Direct DB access
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Repository handles database operations for financial module
type Repository struct {
	db *db.DB
}

// NewRepository creates a new financial repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ExecContext executes a query without returning rows
func (r *Repository) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (r *Repository) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}

// CreateRateCard inserts a new rate card
func (r *Repository) CreateRateCard(ctx context.Context, rateCard *RateCard) error {
	query := `
		INSERT INTO rate_cards (
			rate_card_id, program_id, name, description, effective_start_date,
			effective_end_date, currency, is_active, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		rateCard.RateCardID,
		rateCard.ProgramID,
		rateCard.Name,
		rateCard.Description,
		rateCard.EffectiveStartDate,
		rateCard.EffectiveEndDate,
		rateCard.Currency,
		rateCard.IsActive,
		rateCard.CreatedBy,
		rateCard.CreatedAt,
		rateCard.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create rate card: %w", err)
	}

	return nil
}

// GetRateCardByID retrieves a rate card by ID
func (r *Repository) GetRateCardByID(ctx context.Context, rateCardID uuid.UUID) (*RateCard, error) {
	query := `
		SELECT rate_card_id, program_id, name, description, effective_start_date,
			   effective_end_date, currency, is_active, created_at, created_by,
			   updated_at, updated_by, deleted_at
		FROM rate_cards
		WHERE rate_card_id = $1 AND deleted_at IS NULL
	`

	var rc RateCard
	err := r.db.QueryRowContext(ctx, query, rateCardID).Scan(
		&rc.RateCardID,
		&rc.ProgramID,
		&rc.Name,
		&rc.Description,
		&rc.EffectiveStartDate,
		&rc.EffectiveEndDate,
		&rc.Currency,
		&rc.IsActive,
		&rc.CreatedAt,
		&rc.CreatedBy,
		&rc.UpdatedAt,
		&rc.UpdatedBy,
		&rc.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rate card not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card: %w", err)
	}

	return &rc, nil
}

// ListRateCardsByProgram retrieves all rate cards for a program
func (r *Repository) ListRateCardsByProgram(ctx context.Context, programID uuid.UUID, limit, offset int) ([]RateCard, error) {
	query := `
		SELECT rate_card_id, program_id, name, description, effective_start_date,
			   effective_end_date, currency, is_active, created_at, created_by,
			   updated_at, updated_by
		FROM rate_cards
		WHERE program_id = $1 AND deleted_at IS NULL
		ORDER BY effective_start_date DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, programID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate cards: %w", err)
	}
	defer rows.Close()

	var rateCards []RateCard
	for rows.Next() {
		var rc RateCard
		err := rows.Scan(
			&rc.RateCardID,
			&rc.ProgramID,
			&rc.Name,
			&rc.Description,
			&rc.EffectiveStartDate,
			&rc.EffectiveEndDate,
			&rc.Currency,
			&rc.IsActive,
			&rc.CreatedAt,
			&rc.CreatedBy,
			&rc.UpdatedAt,
			&rc.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rate card: %w", err)
		}
		rateCards = append(rateCards, rc)
	}

	return rateCards, nil
}

// GetActiveRateCards retrieves active rate cards for a program
func (r *Repository) GetActiveRateCards(ctx context.Context, programID uuid.UUID) ([]RateCard, error) {
	query := `
		SELECT rate_card_id, program_id, name, description, effective_start_date,
			   effective_end_date, currency, is_active, created_at, created_by,
			   updated_at, updated_by
		FROM rate_cards
		WHERE program_id = $1
		  AND is_active = TRUE
		  AND deleted_at IS NULL
		  AND effective_start_date <= CURRENT_DATE
		  AND (effective_end_date IS NULL OR effective_end_date >= CURRENT_DATE)
		ORDER BY effective_start_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rate cards: %w", err)
	}
	defer rows.Close()

	var rateCards []RateCard
	for rows.Next() {
		var rc RateCard
		err := rows.Scan(
			&rc.RateCardID,
			&rc.ProgramID,
			&rc.Name,
			&rc.Description,
			&rc.EffectiveStartDate,
			&rc.EffectiveEndDate,
			&rc.Currency,
			&rc.IsActive,
			&rc.CreatedAt,
			&rc.CreatedBy,
			&rc.UpdatedAt,
			&rc.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rate card: %w", err)
		}
		rateCards = append(rateCards, rc)
	}

	return rateCards, nil
}

// UpdateRateCard updates an existing rate card
func (r *Repository) UpdateRateCard(ctx context.Context, rateCard *RateCard) error {
	query := `
		UPDATE rate_cards
		SET name = $1, description = $2, effective_start_date = $3,
		    effective_end_date = $4, is_active = $5, updated_by = $6, updated_at = $7
		WHERE rate_card_id = $8 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		rateCard.Name,
		rateCard.Description,
		rateCard.EffectiveStartDate,
		rateCard.EffectiveEndDate,
		rateCard.IsActive,
		rateCard.UpdatedBy,
		rateCard.UpdatedAt,
		rateCard.RateCardID,
	)

	if err != nil {
		return fmt.Errorf("failed to update rate card: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rate card not found or already deleted")
	}

	return nil
}

// DeleteRateCard soft-deletes a rate card
func (r *Repository) DeleteRateCard(ctx context.Context, rateCardID uuid.UUID) error {
	query := `
		UPDATE rate_cards
		SET deleted_at = NOW()
		WHERE rate_card_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, rateCardID)
	if err != nil {
		return fmt.Errorf("failed to delete rate card: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rate card not found or already deleted")
	}

	return nil
}

// CreateRateCardItems inserts multiple rate card items
func (r *Repository) CreateRateCardItems(ctx context.Context, items []RateCardItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO rate_card_items (
			item_id, rate_card_id, person_name, role_title, seniority_level,
			rate_type, rate_amount, currency, expected_hours_per_week,
			expected_hours_per_month, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, item := range items {
		_, err := r.db.ExecContext(ctx, query,
			item.ItemID,
			item.RateCardID,
			item.PersonName,
			item.RoleTitle,
			item.SeniorityLevel,
			item.RateType,
			item.RateAmount,
			item.Currency,
			item.ExpectedHoursPerWeek,
			item.ExpectedHoursPerMonth,
			item.Notes,
		)
		if err != nil {
			return fmt.Errorf("failed to create rate card item: %w", err)
		}
	}

	return nil
}

// GetRateCardItems retrieves all items for a rate card
func (r *Repository) GetRateCardItems(ctx context.Context, rateCardID uuid.UUID) ([]RateCardItem, error) {
	query := `
		SELECT item_id, rate_card_id, person_name, role_title, seniority_level,
			   rate_type, rate_amount, currency, expected_hours_per_week,
			   expected_hours_per_month, notes, created_at
		FROM rate_card_items
		WHERE rate_card_id = $1
		ORDER BY person_name, role_title
	`

	rows, err := r.db.QueryContext(ctx, query, rateCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card items: %w", err)
	}
	defer rows.Close()

	var items []RateCardItem
	for rows.Next() {
		var item RateCardItem
		err := rows.Scan(
			&item.ItemID,
			&item.RateCardID,
			&item.PersonName,
			&item.RoleTitle,
			&item.SeniorityLevel,
			&item.RateType,
			&item.RateAmount,
			&item.Currency,
			&item.ExpectedHoursPerWeek,
			&item.ExpectedHoursPerMonth,
			&item.Notes,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rate card item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetRateCardItemByPersonName finds a rate card item by person name
func (r *Repository) GetRateCardItemByPersonName(ctx context.Context, rateCardID uuid.UUID, personName string) (*RateCardItem, error) {
	query := `
		SELECT item_id, rate_card_id, person_name, role_title, seniority_level,
			   rate_type, rate_amount, currency, expected_hours_per_week,
			   expected_hours_per_month, notes, created_at
		FROM rate_card_items
		WHERE rate_card_id = $1 AND person_name = $2
		LIMIT 1
	`

	var item RateCardItem
	err := r.db.QueryRowContext(ctx, query, rateCardID, personName).Scan(
		&item.ItemID,
		&item.RateCardID,
		&item.PersonName,
		&item.RoleTitle,
		&item.SeniorityLevel,
		&item.RateType,
		&item.RateAmount,
		&item.Currency,
		&item.ExpectedHoursPerWeek,
		&item.ExpectedHoursPerMonth,
		&item.Notes,
		&item.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rate card item not found for person: %s", personName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card item: %w", err)
	}

	return &item, nil
}

// GetRateCardItemByRole finds a rate card item by role title
func (r *Repository) GetRateCardItemByRole(ctx context.Context, rateCardID uuid.UUID, roleTitle string) (*RateCardItem, error) {
	query := `
		SELECT item_id, rate_card_id, person_name, role_title, seniority_level,
			   rate_type, rate_amount, currency, expected_hours_per_week,
			   expected_hours_per_month, notes, created_at
		FROM rate_card_items
		WHERE rate_card_id = $1 AND role_title = $2
		LIMIT 1
	`

	var item RateCardItem
	err := r.db.QueryRowContext(ctx, query, rateCardID, roleTitle).Scan(
		&item.ItemID,
		&item.RateCardID,
		&item.PersonName,
		&item.RoleTitle,
		&item.SeniorityLevel,
		&item.RateType,
		&item.RateAmount,
		&item.Currency,
		&item.ExpectedHoursPerWeek,
		&item.ExpectedHoursPerMonth,
		&item.Notes,
		&item.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rate card item not found for role: %s", roleTitle)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card item: %w", err)
	}

	return &item, nil
}

// DeleteRateCardItems deletes all items for a rate card
func (r *Repository) DeleteRateCardItems(ctx context.Context, rateCardID uuid.UUID) error {
	query := `DELETE FROM rate_card_items WHERE rate_card_id = $1`

	_, err := r.db.ExecContext(ctx, query, rateCardID)
	if err != nil {
		return fmt.Errorf("failed to delete rate card items: %w", err)
	}

	return nil
}

// CreateInvoice inserts a new invoice
func (r *Repository) CreateInvoice(ctx context.Context, invoice *Invoice) error {
	query := `
		INSERT INTO invoices (
			invoice_id, program_id, artifact_id, invoice_number, vendor_name,
			vendor_id, invoice_date, due_date, period_start_date, period_end_date,
			subtotal_amount, tax_amount, total_amount, currency, processing_status,
			payment_status, ai_model_version, ai_confidence_score, ai_processing_time_ms,
			submitted_by, submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	_, err := r.db.ExecContext(ctx, query,
		invoice.InvoiceID,
		invoice.ProgramID,
		invoice.ArtifactID,
		invoice.InvoiceNumber,
		invoice.VendorName,
		invoice.VendorID,
		invoice.InvoiceDate,
		invoice.DueDate,
		invoice.PeriodStartDate,
		invoice.PeriodEndDate,
		invoice.SubtotalAmount,
		invoice.TaxAmount,
		invoice.TotalAmount,
		invoice.Currency,
		invoice.ProcessingStatus,
		invoice.PaymentStatus,
		invoice.AIModelVersion,
		invoice.AIConfidenceScore,
		invoice.AIProcessingTimeMs,
		invoice.SubmittedBy,
		invoice.SubmittedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	return nil
}

// GetInvoiceByID retrieves an invoice by ID
func (r *Repository) GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error) {
	query := `
		SELECT invoice_id, program_id, artifact_id, invoice_number, vendor_name,
			   vendor_id, invoice_date, due_date, period_start_date, period_end_date,
			   subtotal_amount, tax_amount, total_amount, currency, processing_status,
			   payment_status, ai_model_version, ai_confidence_score, ai_processing_time_ms,
			   submitted_by, submitted_at, approved_by, approved_at, rejected_reason, deleted_at
		FROM invoices
		WHERE invoice_id = $1 AND deleted_at IS NULL
	`

	var inv Invoice
	err := r.db.QueryRowContext(ctx, query, invoiceID).Scan(
		&inv.InvoiceID,
		&inv.ProgramID,
		&inv.ArtifactID,
		&inv.InvoiceNumber,
		&inv.VendorName,
		&inv.VendorID,
		&inv.InvoiceDate,
		&inv.DueDate,
		&inv.PeriodStartDate,
		&inv.PeriodEndDate,
		&inv.SubtotalAmount,
		&inv.TaxAmount,
		&inv.TotalAmount,
		&inv.Currency,
		&inv.ProcessingStatus,
		&inv.PaymentStatus,
		&inv.AIModelVersion,
		&inv.AIConfidenceScore,
		&inv.AIProcessingTimeMs,
		&inv.SubmittedBy,
		&inv.SubmittedAt,
		&inv.ApprovedBy,
		&inv.ApprovedAt,
		&inv.RejectedReason,
		&inv.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	return &inv, nil
}

// GetInvoiceByArtifactID retrieves an invoice by artifact ID
func (r *Repository) GetInvoiceByArtifactID(ctx context.Context, artifactID uuid.UUID) (*Invoice, error) {
	query := `
		SELECT invoice_id, program_id, artifact_id, invoice_number, vendor_name,
			   vendor_id, invoice_date, due_date, period_start_date, period_end_date,
			   subtotal_amount, tax_amount, total_amount, currency, processing_status,
			   payment_status, ai_model_version, ai_confidence_score, ai_processing_time_ms,
			   submitted_by, submitted_at, approved_by, approved_at, rejected_reason, deleted_at
		FROM invoices
		WHERE artifact_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var inv Invoice
	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(
		&inv.InvoiceID,
		&inv.ProgramID,
		&inv.ArtifactID,
		&inv.InvoiceNumber,
		&inv.VendorName,
		&inv.VendorID,
		&inv.InvoiceDate,
		&inv.DueDate,
		&inv.PeriodStartDate,
		&inv.PeriodEndDate,
		&inv.SubtotalAmount,
		&inv.TaxAmount,
		&inv.TotalAmount,
		&inv.Currency,
		&inv.ProcessingStatus,
		&inv.PaymentStatus,
		&inv.AIModelVersion,
		&inv.AIConfidenceScore,
		&inv.AIProcessingTimeMs,
		&inv.SubmittedBy,
		&inv.SubmittedAt,
		&inv.ApprovedBy,
		&inv.ApprovedAt,
		&inv.RejectedReason,
		&inv.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found for artifact")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice by artifact: %w", err)
	}

	return &inv, nil
}

// ListInvoices retrieves invoices with optional filters
func (r *Repository) ListInvoices(ctx context.Context, filter InvoiceFilterRequest) ([]Invoice, error) {
	query := `
		SELECT invoice_id, program_id, artifact_id, invoice_number, vendor_name,
			   vendor_id, invoice_date, due_date, period_start_date, period_end_date,
			   subtotal_amount, tax_amount, total_amount, currency, processing_status,
			   payment_status, submitted_by, submitted_at
		FROM invoices
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{filter.ProgramID}
	argCount := 1

	// Add filters
	if filter.ProcessingStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND processing_status = $%d", argCount)
		args = append(args, filter.ProcessingStatus)
	}

	if filter.PaymentStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND payment_status = $%d", argCount)
		args = append(args, filter.PaymentStatus)
	}

	if filter.VendorName != "" {
		argCount++
		query += fmt.Sprintf(" AND vendor_name ILIKE $%d", argCount)
		args = append(args, "%"+filter.VendorName+"%")
	}

	if filter.DateFrom != nil {
		argCount++
		query += fmt.Sprintf(" AND invoice_date >= $%d", argCount)
		args = append(args, filter.DateFrom)
	}

	if filter.DateTo != nil {
		argCount++
		query += fmt.Sprintf(" AND invoice_date <= $%d", argCount)
		args = append(args, filter.DateTo)
	}

	query += " ORDER BY invoice_date DESC"

	// Add pagination
	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, filter.Limit)

	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var inv Invoice
		err := rows.Scan(
			&inv.InvoiceID,
			&inv.ProgramID,
			&inv.ArtifactID,
			&inv.InvoiceNumber,
			&inv.VendorName,
			&inv.VendorID,
			&inv.InvoiceDate,
			&inv.DueDate,
			&inv.PeriodStartDate,
			&inv.PeriodEndDate,
			&inv.SubtotalAmount,
			&inv.TaxAmount,
			&inv.TotalAmount,
			&inv.Currency,
			&inv.ProcessingStatus,
			&inv.PaymentStatus,
			&inv.SubmittedBy,
			&inv.SubmittedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}

	return invoices, nil
}

// UpdateInvoice updates an existing invoice
func (r *Repository) UpdateInvoice(ctx context.Context, invoice *Invoice) error {
	query := `
		UPDATE invoices
		SET processing_status = $1, payment_status = $2, approved_by = $3,
		    approved_at = $4, rejected_reason = $5
		WHERE invoice_id = $6 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		invoice.ProcessingStatus,
		invoice.PaymentStatus,
		invoice.ApprovedBy,
		invoice.ApprovedAt,
		invoice.RejectedReason,
		invoice.InvoiceID,
	)

	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("invoice not found or already deleted")
	}

	return nil
}

// DeleteInvoice soft-deletes an invoice
func (r *Repository) DeleteInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	query := `
		UPDATE invoices
		SET deleted_at = NOW()
		WHERE invoice_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("invoice not found or already deleted")
	}

	return nil
}

// SaveLineItems inserts multiple invoice line items
func (r *Repository) SaveLineItems(ctx context.Context, lineItems []InvoiceLineItem) error {
	if len(lineItems) == 0 {
		return nil
	}

	query := `
		INSERT INTO invoice_line_items (
			line_item_id, invoice_id, line_number, description, quantity, unit_rate,
			line_amount, person_name, role_description, matched_rate_card_item_id,
			expected_rate, rate_variance_amount, rate_variance_percentage,
			billed_hours, expected_hours, hours_variance, spend_category,
			budget_category_id, has_variance, variance_severity, needs_review,
			review_notes, ai_confidence_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	for _, item := range lineItems {
		_, err := r.db.ExecContext(ctx, query,
			item.LineItemID,
			item.InvoiceID,
			item.LineNumber,
			item.Description,
			item.Quantity,
			item.UnitRate,
			item.LineAmount,
			item.PersonName,
			item.RoleDescription,
			item.MatchedRateCardItemID,
			item.ExpectedRate,
			item.RateVarianceAmount,
			item.RateVariancePercentage,
			item.BilledHours,
			item.ExpectedHours,
			item.HoursVariance,
			item.SpendCategory,
			item.BudgetCategoryID,
			item.HasVariance,
			item.VarianceSeverity,
			item.NeedsReview,
			item.ReviewNotes,
			item.AIConfidenceScore,
		)
		if err != nil {
			return fmt.Errorf("failed to save line item %d: %w", item.LineNumber, err)
		}
	}

	return nil
}

// GetLineItems retrieves all line items for an invoice
func (r *Repository) GetLineItems(ctx context.Context, invoiceID uuid.UUID) ([]InvoiceLineItem, error) {
	query := `
		SELECT line_item_id, invoice_id, line_number, description, quantity, unit_rate,
			   line_amount, person_name, role_description, matched_rate_card_item_id,
			   expected_rate, rate_variance_amount, rate_variance_percentage,
			   billed_hours, expected_hours, hours_variance, spend_category,
			   budget_category_id, has_variance, variance_severity, needs_review,
			   review_notes, ai_confidence_score
		FROM invoice_line_items
		WHERE invoice_id = $1
		ORDER BY line_number
	`

	rows, err := r.db.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}
	defer rows.Close()

	var items []InvoiceLineItem
	for rows.Next() {
		var item InvoiceLineItem
		err := rows.Scan(
			&item.LineItemID,
			&item.InvoiceID,
			&item.LineNumber,
			&item.Description,
			&item.Quantity,
			&item.UnitRate,
			&item.LineAmount,
			&item.PersonName,
			&item.RoleDescription,
			&item.MatchedRateCardItemID,
			&item.ExpectedRate,
			&item.RateVarianceAmount,
			&item.RateVariancePercentage,
			&item.BilledHours,
			&item.ExpectedHours,
			&item.HoursVariance,
			&item.SpendCategory,
			&item.BudgetCategoryID,
			&item.HasVariance,
			&item.VarianceSeverity,
			&item.NeedsReview,
			&item.ReviewNotes,
			&item.AIConfidenceScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan line item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateLineItem updates a single line item
func (r *Repository) UpdateLineItem(ctx context.Context, lineItem *InvoiceLineItem) error {
	query := `
		UPDATE invoice_line_items
		SET matched_rate_card_item_id = $1, expected_rate = $2,
		    rate_variance_amount = $3, rate_variance_percentage = $4,
		    expected_hours = $5, hours_variance = $6, has_variance = $7,
		    variance_severity = $8, needs_review = $9, review_notes = $10
		WHERE line_item_id = $11
	`

	result, err := r.db.ExecContext(ctx, query,
		lineItem.MatchedRateCardItemID,
		lineItem.ExpectedRate,
		lineItem.RateVarianceAmount,
		lineItem.RateVariancePercentage,
		lineItem.ExpectedHours,
		lineItem.HoursVariance,
		lineItem.HasVariance,
		lineItem.VarianceSeverity,
		lineItem.NeedsReview,
		lineItem.ReviewNotes,
		lineItem.LineItemID,
	)

	if err != nil {
		return fmt.Errorf("failed to update line item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("line item not found")
	}

	return nil
}

// GetInvoiceWithLineItems retrieves invoice with all line items
func (r *Repository) GetInvoiceWithLineItems(ctx context.Context, invoiceID uuid.UUID) (*InvoiceWithMetadata, error) {
	// Get invoice
	invoice, err := r.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	result := &InvoiceWithMetadata{
		Invoice: *invoice,
	}

	// Get line items
	lineItems, err := r.GetLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}
	result.LineItems = lineItems

	// Get variances
	variances, err := r.GetVariances(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variances: %w", err)
	}
	result.Variances = variances

	return result, nil
}

// GetRateCardWithItems retrieves rate card with all items
func (r *Repository) GetRateCardWithItems(ctx context.Context, rateCardID uuid.UUID) (*RateCardWithItems, error) {
	// Get rate card
	rateCard, err := r.GetRateCardByID(ctx, rateCardID)
	if err != nil {
		return nil, err
	}

	result := &RateCardWithItems{
		RateCard: *rateCard,
	}

	// Get items
	items, err := r.GetRateCardItems(ctx, rateCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate card items: %w", err)
	}
	result.Items = items

	return result, nil
}

// CreateBudgetCategory inserts a new budget category
func (r *Repository) CreateBudgetCategory(ctx context.Context, category *BudgetCategory) error {
	query := `
		INSERT INTO budget_categories (
			category_id, program_id, category_name, description, budgeted_amount,
			currency, fiscal_year, fiscal_quarter, actual_spend, committed_spend,
			variance_amount, variance_percentage
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		category.CategoryID,
		category.ProgramID,
		category.CategoryName,
		category.Description,
		category.BudgetedAmount,
		category.Currency,
		category.FiscalYear,
		category.FiscalQuarter,
		category.ActualSpend,
		category.CommittedSpend,
		category.VarianceAmount,
		category.VariancePercentage,
	)

	if err != nil {
		return fmt.Errorf("failed to create budget category: %w", err)
	}

	return nil
}

// GetBudgetCategoryByID retrieves a budget category by ID
func (r *Repository) GetBudgetCategoryByID(ctx context.Context, categoryID uuid.UUID) (*BudgetCategory, error) {
	query := `
		SELECT category_id, program_id, category_name, description, budgeted_amount,
			   currency, fiscal_year, fiscal_quarter, actual_spend, committed_spend,
			   variance_amount, variance_percentage, created_at, updated_at, deleted_at
		FROM budget_categories
		WHERE category_id = $1 AND deleted_at IS NULL
	`

	var cat BudgetCategory
	err := r.db.QueryRowContext(ctx, query, categoryID).Scan(
		&cat.CategoryID,
		&cat.ProgramID,
		&cat.CategoryName,
		&cat.Description,
		&cat.BudgetedAmount,
		&cat.Currency,
		&cat.FiscalYear,
		&cat.FiscalQuarter,
		&cat.ActualSpend,
		&cat.CommittedSpend,
		&cat.VarianceAmount,
		&cat.VariancePercentage,
		&cat.CreatedAt,
		&cat.UpdatedAt,
		&cat.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("budget category not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get budget category: %w", err)
	}

	return &cat, nil
}

// ListBudgetCategories retrieves budget categories for a program and fiscal year
func (r *Repository) ListBudgetCategories(ctx context.Context, programID uuid.UUID, fiscalYear int) ([]BudgetCategory, error) {
	query := `
		SELECT category_id, program_id, category_name, description, budgeted_amount,
			   currency, fiscal_year, fiscal_quarter, actual_spend, committed_spend,
			   variance_amount, variance_percentage, created_at, updated_at
		FROM budget_categories
		WHERE program_id = $1 AND fiscal_year = $2 AND deleted_at IS NULL
		ORDER BY category_name
	`

	rows, err := r.db.QueryContext(ctx, query, programID, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("failed to list budget categories: %w", err)
	}
	defer rows.Close()

	var categories []BudgetCategory
	for rows.Next() {
		var cat BudgetCategory
		err := rows.Scan(
			&cat.CategoryID,
			&cat.ProgramID,
			&cat.CategoryName,
			&cat.Description,
			&cat.BudgetedAmount,
			&cat.Currency,
			&cat.FiscalYear,
			&cat.FiscalQuarter,
			&cat.ActualSpend,
			&cat.CommittedSpend,
			&cat.VarianceAmount,
			&cat.VariancePercentage,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget category: %w", err)
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

// UpdateBudgetCategory updates an existing budget category
func (r *Repository) UpdateBudgetCategory(ctx context.Context, category *BudgetCategory) error {
	query := `
		UPDATE budget_categories
		SET budgeted_amount = $1, actual_spend = $2, committed_spend = $3
		WHERE category_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		category.BudgetedAmount,
		category.ActualSpend,
		category.CommittedSpend,
		category.CategoryID,
	)

	if err != nil {
		return fmt.Errorf("failed to update budget category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("budget category not found or already deleted")
	}

	return nil
}

// DeleteBudgetCategory soft-deletes a budget category
func (r *Repository) DeleteBudgetCategory(ctx context.Context, categoryID uuid.UUID) error {
	query := `
		UPDATE budget_categories
		SET deleted_at = NOW()
		WHERE category_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, categoryID)
	if err != nil {
		return fmt.Errorf("failed to delete budget category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("budget category not found or already deleted")
	}

	return nil
}

// SaveVariances inserts multiple financial variances
func (r *Repository) SaveVariances(ctx context.Context, variances []FinancialVariance) error {
	if len(variances) == 0 {
		return nil
	}

	query := `
		INSERT INTO financial_variances (
			variance_id, program_id, invoice_id, line_item_id, variance_type,
			severity, title, description, expected_value, actual_value,
			variance_amount, variance_percentage, source_artifact_ids,
			conflicting_values, ai_confidence_score, ai_detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	for _, v := range variances {
		_, err := r.db.ExecContext(ctx, query,
			v.VarianceID,
			v.ProgramID,
			v.InvoiceID,
			v.LineItemID,
			v.VarianceType,
			v.Severity,
			v.Title,
			v.Description,
			v.ExpectedValue,
			v.ActualValue,
			v.VarianceAmount,
			v.VariancePercentage,
			pq.Array(v.SourceArtifactIDs),
			v.ConflictingValues,
			v.AIConfidenceScore,
			v.AIDetectedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to save variance: %w", err)
		}
	}

	return nil
}

// GetVariances retrieves variances for an invoice
func (r *Repository) GetVariances(ctx context.Context, invoiceID uuid.UUID) ([]FinancialVariance, error) {
	query := `
		SELECT variance_id, program_id, invoice_id, line_item_id, variance_type,
			   severity, title, description, expected_value, actual_value,
			   variance_amount, variance_percentage, source_artifact_ids,
			   conflicting_values, ai_confidence_score, ai_detected_at,
			   is_dismissed, dismissed_by, dismissed_at, dismissal_reason,
			   resolution_notes, resolved_at, resolved_by
		FROM financial_variances
		WHERE invoice_id = $1
		ORDER BY severity DESC, ai_detected_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variances: %w", err)
	}
	defer rows.Close()

	var variances []FinancialVariance
	for rows.Next() {
		var v FinancialVariance
		err := rows.Scan(
			&v.VarianceID,
			&v.ProgramID,
			&v.InvoiceID,
			&v.LineItemID,
			&v.VarianceType,
			&v.Severity,
			&v.Title,
			&v.Description,
			&v.ExpectedValue,
			&v.ActualValue,
			&v.VarianceAmount,
			&v.VariancePercentage,
			pq.Array(&v.SourceArtifactIDs),
			&v.ConflictingValues,
			&v.AIConfidenceScore,
			&v.AIDetectedAt,
			&v.IsDismissed,
			&v.DismissedBy,
			&v.DismissedAt,
			&v.DismissalReason,
			&v.ResolutionNotes,
			&v.ResolvedAt,
			&v.ResolvedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variance: %w", err)
		}
		variances = append(variances, v)
	}

	return variances, nil
}

// GetVariancesByProgram retrieves variances for a program with optional severity filter
func (r *Repository) GetVariancesByProgram(ctx context.Context, programID uuid.UUID, severityFilter string) ([]FinancialVariance, error) {
	query := `
		SELECT variance_id, program_id, invoice_id, line_item_id, variance_type,
			   severity, title, description, expected_value, actual_value,
			   variance_amount, variance_percentage, source_artifact_ids,
			   conflicting_values, ai_confidence_score, ai_detected_at,
			   is_dismissed, dismissed_by, dismissed_at, dismissal_reason,
			   resolution_notes, resolved_at, resolved_by
		FROM financial_variances
		WHERE program_id = $1 AND is_dismissed = FALSE
	`

	args := []interface{}{programID}

	if severityFilter != "" {
		query += " AND severity = $2"
		args = append(args, severityFilter)
	}

	query += " ORDER BY severity DESC, ai_detected_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get program variances: %w", err)
	}
	defer rows.Close()

	var variances []FinancialVariance
	for rows.Next() {
		var v FinancialVariance
		err := rows.Scan(
			&v.VarianceID,
			&v.ProgramID,
			&v.InvoiceID,
			&v.LineItemID,
			&v.VarianceType,
			&v.Severity,
			&v.Title,
			&v.Description,
			&v.ExpectedValue,
			&v.ActualValue,
			&v.VarianceAmount,
			&v.VariancePercentage,
			pq.Array(&v.SourceArtifactIDs),
			&v.ConflictingValues,
			&v.AIConfidenceScore,
			&v.AIDetectedAt,
			&v.IsDismissed,
			&v.DismissedBy,
			&v.DismissedAt,
			&v.DismissalReason,
			&v.ResolutionNotes,
			&v.ResolvedAt,
			&v.ResolvedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variance: %w", err)
		}
		variances = append(variances, v)
	}

	return variances, nil
}

// DismissVariance marks a variance as dismissed
func (r *Repository) DismissVariance(ctx context.Context, varianceID uuid.UUID, dismissedBy uuid.UUID, reason string) error {
	query := `
		UPDATE financial_variances
		SET is_dismissed = TRUE, dismissed_by = $1, dismissed_at = NOW(), dismissal_reason = $2
		WHERE variance_id = $3
	`

	result, err := r.db.ExecContext(ctx, query, dismissedBy, reason, varianceID)
	if err != nil {
		return fmt.Errorf("failed to dismiss variance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("variance not found")
	}

	return nil
}

// ResolveVariance marks a variance as resolved
func (r *Repository) ResolveVariance(ctx context.Context, varianceID uuid.UUID, resolvedBy uuid.UUID, notes string) error {
	query := `
		UPDATE financial_variances
		SET resolved_by = $1, resolved_at = NOW(), resolution_notes = $2
		WHERE variance_id = $3
	`

	result, err := r.db.ExecContext(ctx, query, resolvedBy, notes, varianceID)
	if err != nil {
		return fmt.Errorf("failed to resolve variance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("variance not found")
	}

	return nil
}
