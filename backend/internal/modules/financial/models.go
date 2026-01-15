package financial

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// RateCard represents approved billing rates for a program
type RateCard struct {
	RateCardID         uuid.UUID      `json:"rate_card_id"`
	ProgramID          uuid.UUID      `json:"program_id"`
	Name               string         `json:"name"`
	Description        sql.NullString `json:"description,omitempty"`
	EffectiveStartDate time.Time      `json:"effective_start_date"`
	EffectiveEndDate   sql.NullTime   `json:"effective_end_date,omitempty"`
	Currency           string         `json:"currency"`
	IsActive           bool           `json:"is_active"`
	CreatedAt          time.Time      `json:"created_at"`
	CreatedBy          uuid.UUID      `json:"created_by"`
	UpdatedAt          time.Time      `json:"updated_at"`
	UpdatedBy          uuid.NullUUID  `json:"updated_by,omitempty"`
	DeletedAt          sql.NullTime   `json:"deleted_at,omitempty"`
}

// RateCardItem represents an individual rate for a person or role
type RateCardItem struct {
	ItemID                uuid.UUID       `json:"item_id"`
	RateCardID            uuid.UUID       `json:"rate_card_id"`
	PersonName            sql.NullString  `json:"person_name,omitempty"`
	RoleTitle             sql.NullString  `json:"role_title,omitempty"`
	SeniorityLevel        sql.NullString  `json:"seniority_level,omitempty"`
	RateType              string          `json:"rate_type"`
	RateAmount            float64         `json:"rate_amount"`
	Currency              string          `json:"currency"`
	ExpectedHoursPerWeek  sql.NullFloat64 `json:"expected_hours_per_week,omitempty"`
	ExpectedHoursPerMonth sql.NullFloat64 `json:"expected_hours_per_month,omitempty"`
	Notes                 sql.NullString  `json:"notes,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

// Invoice represents an invoice extracted from an artifact
type Invoice struct {
	InvoiceID           uuid.UUID       `json:"invoice_id"`
	ProgramID           uuid.UUID       `json:"program_id"`
	ArtifactID          uuid.NullUUID   `json:"artifact_id,omitempty"`
	InvoiceNumber       sql.NullString  `json:"invoice_number,omitempty"`
	VendorName          string          `json:"vendor_name"`
	VendorID            sql.NullString  `json:"vendor_id,omitempty"`
	InvoiceDate         time.Time       `json:"invoice_date"`
	DueDate             sql.NullTime    `json:"due_date,omitempty"`
	PeriodStartDate     sql.NullTime    `json:"period_start_date,omitempty"`
	PeriodEndDate       sql.NullTime    `json:"period_end_date,omitempty"`
	SubtotalAmount      sql.NullFloat64 `json:"subtotal_amount,omitempty"`
	TaxAmount           sql.NullFloat64 `json:"tax_amount,omitempty"`
	TotalAmount         float64         `json:"total_amount"`
	Currency            string          `json:"currency"`
	ProcessingStatus    string          `json:"processing_status"`
	PaymentStatus       string          `json:"payment_status"`
	AIModelVersion      sql.NullString  `json:"ai_model_version,omitempty"`
	AIConfidenceScore   sql.NullFloat64 `json:"ai_confidence_score,omitempty"`
	AIProcessingTimeMs  sql.NullInt32   `json:"ai_processing_time_ms,omitempty"`
	SubmittedBy         uuid.NullUUID   `json:"submitted_by,omitempty"`
	SubmittedAt         time.Time       `json:"submitted_at"`
	ApprovedBy          uuid.NullUUID   `json:"approved_by,omitempty"`
	ApprovedAt          sql.NullTime    `json:"approved_at,omitempty"`
	RejectedReason      sql.NullString  `json:"rejected_reason,omitempty"`
	DeletedAt           sql.NullTime    `json:"deleted_at,omitempty"`
	ReplacedByInvoiceID uuid.NullUUID   `json:"replaced_by_invoice_id,omitempty"`
}

// InvoiceLineItem represents a line item from an invoice
type InvoiceLineItem struct {
	LineItemID             uuid.UUID       `json:"line_item_id"`
	InvoiceID              uuid.UUID       `json:"invoice_id"`
	LineNumber             int             `json:"line_number"`
	Description            string          `json:"description"`
	Quantity               sql.NullFloat64 `json:"quantity,omitempty"`
	UnitRate               sql.NullFloat64 `json:"unit_rate,omitempty"`
	LineAmount             float64         `json:"line_amount"`
	PersonName             sql.NullString  `json:"person_name,omitempty"`
	RoleDescription        sql.NullString  `json:"role_description,omitempty"`
	MatchedRateCardItemID  uuid.NullUUID   `json:"matched_rate_card_item_id,omitempty"`
	ExpectedRate           sql.NullFloat64 `json:"expected_rate,omitempty"`
	RateVarianceAmount     sql.NullFloat64 `json:"rate_variance_amount,omitempty"`
	RateVariancePercentage sql.NullFloat64 `json:"rate_variance_percentage,omitempty"`
	BilledHours            sql.NullFloat64 `json:"billed_hours,omitempty"`
	ExpectedHours          sql.NullFloat64 `json:"expected_hours,omitempty"`
	HoursVariance          sql.NullFloat64 `json:"hours_variance,omitempty"`
	SpendCategory          sql.NullString  `json:"spend_category,omitempty"`
	BudgetCategoryID       uuid.NullUUID   `json:"budget_category_id,omitempty"`
	HasVariance            bool            `json:"has_variance"`
	VarianceSeverity       sql.NullString  `json:"variance_severity,omitempty"`
	NeedsReview            bool            `json:"needs_review"`
	ReviewNotes            sql.NullString  `json:"review_notes,omitempty"`
	AIConfidenceScore      sql.NullFloat64 `json:"ai_confidence_score,omitempty"`
}

// BudgetCategory represents a budget category for tracking spend
type BudgetCategory struct {
	CategoryID        uuid.UUID       `json:"category_id"`
	ProgramID         uuid.UUID       `json:"program_id"`
	CategoryName      string          `json:"category_name"`
	Description       sql.NullString  `json:"description,omitempty"`
	BudgetedAmount    float64         `json:"budgeted_amount"`
	Currency          string          `json:"currency"`
	FiscalYear        int             `json:"fiscal_year"`
	FiscalQuarter     sql.NullInt32   `json:"fiscal_quarter,omitempty"`
	ActualSpend       float64         `json:"actual_spend"`
	CommittedSpend    float64         `json:"committed_spend"`
	VarianceAmount    float64         `json:"variance_amount"`
	VariancePercentage float64        `json:"variance_percentage"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         sql.NullTime    `json:"deleted_at,omitempty"`
}

// FinancialVariance represents an AI-detected billing discrepancy
type FinancialVariance struct {
	VarianceID         uuid.UUID       `json:"variance_id"`
	ProgramID          uuid.UUID       `json:"program_id"`
	InvoiceID          uuid.NullUUID   `json:"invoice_id,omitempty"`
	LineItemID         uuid.NullUUID   `json:"line_item_id,omitempty"`
	VarianceType       string          `json:"variance_type"`
	Severity           string          `json:"severity"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	ExpectedValue      sql.NullFloat64 `json:"expected_value,omitempty"`
	ActualValue        sql.NullFloat64 `json:"actual_value,omitempty"`
	VarianceAmount     sql.NullFloat64 `json:"variance_amount,omitempty"`
	VariancePercentage sql.NullFloat64 `json:"variance_percentage,omitempty"`
	SourceArtifactIDs  []uuid.UUID     `json:"source_artifact_ids"`
	ConflictingValues  []byte          `json:"conflicting_values"` // JSONB
	AIConfidenceScore  sql.NullFloat64 `json:"ai_confidence_score,omitempty"`
	AIDetectedAt       time.Time       `json:"ai_detected_at"`
	IsDismissed        bool            `json:"is_dismissed"`
	DismissedBy        uuid.NullUUID   `json:"dismissed_by,omitempty"`
	DismissedAt        sql.NullTime    `json:"dismissed_at,omitempty"`
	DismissalReason    sql.NullString  `json:"dismissal_reason,omitempty"`
	ResolutionNotes    sql.NullString  `json:"resolution_notes,omitempty"`
	ResolvedAt         sql.NullTime    `json:"resolved_at,omitempty"`
	ResolvedBy         uuid.NullUUID   `json:"resolved_by,omitempty"`
}

// InvoiceWithMetadata combines invoice with its line items and variances
type InvoiceWithMetadata struct {
	Invoice
	LineItems []InvoiceLineItem   `json:"line_items,omitempty"`
	Variances []FinancialVariance `json:"variances,omitempty"`
}

// RateCardWithItems combines rate card with its items
type RateCardWithItems struct {
	RateCard
	Items []RateCardItem `json:"items,omitempty"`
}

// CreateRateCardRequest represents a request to create a rate card
type CreateRateCardRequest struct {
	ProgramID          uuid.UUID              `json:"program_id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	EffectiveStartDate time.Time              `json:"effective_start_date"`
	EffectiveEndDate   *time.Time             `json:"effective_end_date,omitempty"`
	Currency           string                 `json:"currency"`
	Items              []CreateRateCardItemRequest `json:"items"`
	CreatedBy          uuid.UUID              `json:"created_by"`
}

// CreateRateCardItemRequest represents a rate card item in a create request
type CreateRateCardItemRequest struct {
	PersonName            string   `json:"person_name,omitempty"`
	RoleTitle             string   `json:"role_title,omitempty"`
	SeniorityLevel        string   `json:"seniority_level,omitempty"`
	RateType              string   `json:"rate_type"`
	RateAmount            float64  `json:"rate_amount"`
	Currency              string   `json:"currency"`
	ExpectedHoursPerWeek  *float64 `json:"expected_hours_per_week,omitempty"`
	ExpectedHoursPerMonth *float64 `json:"expected_hours_per_month,omitempty"`
	Notes                 string   `json:"notes,omitempty"`
}

// InvoiceFilterRequest represents filters for invoice listing
type InvoiceFilterRequest struct {
	ProgramID        uuid.UUID
	ProcessingStatus string
	PaymentStatus    string
	VendorName       string
	DateFrom         *time.Time
	DateTo           *time.Time
	Limit            int
	Offset           int
}
