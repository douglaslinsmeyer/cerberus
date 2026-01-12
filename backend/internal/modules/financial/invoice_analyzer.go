package financial

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/google/uuid"
)

// InvoiceAnalyzer handles AI-powered invoice analysis and variance detection
type InvoiceAnalyzer struct {
	client  *ai.Client
	repo    RepositoryInterface
	prompts *ai.PromptLibrary
}

// NewInvoiceAnalyzer creates a new invoice analyzer
func NewInvoiceAnalyzer(client *ai.Client, repo RepositoryInterface) *InvoiceAnalyzer {
	return &InvoiceAnalyzer{
		client:  client,
		repo:    repo,
		prompts: ai.NewPromptLibrary(),
	}
}

// InvoiceExtractionResponse matches the JSON schema from the AI extraction prompt
type InvoiceExtractionResponse struct {
	InvoiceNumber string `json:"invoice_number"`
	VendorName    string `json:"vendor_name"`
	VendorID      string `json:"vendor_id,omitempty"`
	InvoiceDate   string `json:"invoice_date"`
	DueDate       string `json:"due_date,omitempty"`
	PeriodStart   string `json:"period_start,omitempty"`
	PeriodEnd     string `json:"period_end,omitempty"`
	Subtotal      float64 `json:"subtotal,omitempty"`
	Tax           float64 `json:"tax,omitempty"`
	Total         float64 `json:"total"`
	Currency      string `json:"currency"`
	LineItems     []struct {
		LineNumber      int     `json:"line_number"`
		Description     string  `json:"description"`
		PersonName      string  `json:"person_name,omitempty"`
		RoleDescription string  `json:"role_description,omitempty"`
		Quantity        float64 `json:"quantity,omitempty"`
		UnitRate        float64 `json:"unit_rate,omitempty"`
		BilledHours     float64 `json:"billed_hours,omitempty"`
		LineAmount      float64 `json:"line_amount"`
		SpendCategory   string  `json:"spend_category,omitempty"`
		Confidence      float64 `json:"confidence"`
	} `json:"line_items"`
	OverallConfidence float64 `json:"overall_confidence"`
}

// VarianceDetectionResult contains detected variances from cross-document analysis
type VarianceDetectionResult struct {
	Variances      []FinancialVariance
	ProcessingTime time.Duration
	TokensUsed     int
	Cost           float64
}

// AnalyzeInvoice extracts invoice data from artifact content using AI
func (a *InvoiceAnalyzer) AnalyzeInvoice(ctx context.Context, artifactContent string, programID uuid.UUID) (*Invoice, []InvoiceLineItem, error) {
	startTime := time.Now()

	// Build prompt for invoice extraction
	systemPrompt := `You are an expert financial analyst specializing in invoice processing. Your task is to extract structured data from invoice documents with high accuracy.

Extract the following information:
1. Invoice header: invoice number, vendor, dates, amounts
2. Line items: description, person/role, hours, rates, amounts
3. Categorize spend as: labor, materials, software, travel, or other

Return JSON matching this exact schema:
{
  "invoice_number": "string",
  "vendor_name": "string",
  "vendor_id": "string (optional)",
  "invoice_date": "YYYY-MM-DD",
  "due_date": "YYYY-MM-DD (optional)",
  "period_start": "YYYY-MM-DD (optional)",
  "period_end": "YYYY-MM-DD (optional)",
  "subtotal": number,
  "tax": number,
  "total": number,
  "currency": "USD",
  "line_items": [
    {
      "line_number": 1,
      "description": "string",
      "person_name": "string (if labor)",
      "role_description": "string (if labor)",
      "quantity": number,
      "unit_rate": number,
      "billed_hours": number (if labor),
      "line_amount": number,
      "spend_category": "labor|materials|software|travel|other",
      "confidence": 0.0-1.0
    }
  ],
  "overall_confidence": 0.0-1.0
}

IMPORTANT:
- Extract person names from line items that represent labor/consulting work
- Identify hourly rates and hours worked
- For labor items, always populate person_name, billed_hours, and unit_rate
- Confidence scores should reflect certainty of extraction`

	userPrompt := fmt.Sprintf(`Extract invoice data from this document:

---
%s
---

Return only the JSON, no explanation.`, artifactContent)

	// Call Claude API
	resp, err := a.client.RequestWithContext(
		ctx,
		ai.ModelSonnet4,
		systemPrompt,
		"", // No static context for invoice extraction
		userPrompt,
		4096,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("Claude API request failed: %w", err)
	}

	// Parse JSON response
	responseText := resp.GetExtractedText()
	responseText = stripMarkdownCodeBlocks(responseText)

	var extraction InvoiceExtractionResponse
	if err := json.Unmarshal([]byte(responseText), &extraction); err != nil {
		return nil, nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Convert to domain models
	invoice := &Invoice{
		InvoiceID:          uuid.New(),
		ProgramID:          programID,
		InvoiceNumber:      toNullString(extraction.InvoiceNumber),
		VendorName:         extraction.VendorName,
		VendorID:           toNullString(extraction.VendorID),
		InvoiceDate:        parseDate(extraction.InvoiceDate),
		DueDate:            toNullTime(extraction.DueDate),
		PeriodStartDate:    toNullTime(extraction.PeriodStart),
		PeriodEndDate:      toNullTime(extraction.PeriodEnd),
		SubtotalAmount:     toNullFloat64(extraction.Subtotal),
		TaxAmount:          toNullFloat64(extraction.Tax),
		TotalAmount:        extraction.Total,
		Currency:           extraction.Currency,
		ProcessingStatus:   "processing",
		PaymentStatus:      "unpaid",
		AIModelVersion:     toNullString(resp.Model),
		AIConfidenceScore:  toNullFloat64(extraction.OverallConfidence),
		AIProcessingTimeMs: sql.NullInt32{Int32: int32(time.Since(startTime).Milliseconds()), Valid: true},
		SubmittedAt:        time.Now(),
	}

	// Convert line items
	var lineItems []InvoiceLineItem
	for _, li := range extraction.LineItems {
		lineItem := InvoiceLineItem{
			LineItemID:        uuid.New(),
			InvoiceID:         invoice.InvoiceID,
			LineNumber:        li.LineNumber,
			Description:       li.Description,
			Quantity:          toNullFloat64(li.Quantity),
			UnitRate:          toNullFloat64(li.UnitRate),
			LineAmount:        li.LineAmount,
			PersonName:        toNullString(li.PersonName),
			RoleDescription:   toNullString(li.RoleDescription),
			BilledHours:       toNullFloat64(li.BilledHours),
			SpendCategory:     toNullString(li.SpendCategory),
			AIConfidenceScore: toNullFloat64(li.Confidence),
			HasVariance:       false,
			NeedsReview:       false,
		}
		lineItems = append(lineItems, lineItem)
	}

	return invoice, lineItems, nil
}

// ValidateAgainstRateCards checks invoice line items against rate cards
func (a *InvoiceAnalyzer) ValidateAgainstRateCards(ctx context.Context, invoice *Invoice, lineItems []InvoiceLineItem) ([]FinancialVariance, error) {
	// Get active rate cards for the program
	rateCards, err := a.repo.GetActiveRateCards(ctx, invoice.ProgramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rate cards: %w", err)
	}

	if len(rateCards) == 0 {
		// No rate cards to validate against
		return nil, nil
	}

	var variances []FinancialVariance

	// For each line item, try to match against rate cards
	for i := range lineItems {
		lineItem := &lineItems[i]

		// Skip non-labor items
		if !lineItem.PersonName.Valid || lineItem.PersonName.String == "" {
			continue
		}

		// Try to find matching rate card item
		var matchedItem *RateCardItem
		for _, rc := range rateCards {
			// Try exact person name match first
			item, err := a.repo.GetRateCardItemByPersonName(ctx, rc.RateCardID, lineItem.PersonName.String)
			if err == nil {
				matchedItem = item
				break
			}

			// Try role match as fallback
			if lineItem.RoleDescription.Valid {
				item, err := a.repo.GetRateCardItemByRole(ctx, rc.RateCardID, lineItem.RoleDescription.String)
				if err == nil {
					matchedItem = item
					break
				}
			}
		}

		if matchedItem == nil {
			// No rate card found for this person/role
			continue
		}

		// Update line item with matched rate card
		lineItem.MatchedRateCardItemID = uuid.NullUUID{UUID: matchedItem.ItemID, Valid: true}
		lineItem.ExpectedRate = sql.NullFloat64{Float64: matchedItem.RateAmount, Valid: true}

		// Check rate variance
		if lineItem.UnitRate.Valid && lineItem.UnitRate.Float64 > 0 {
			actualRate := lineItem.UnitRate.Float64
			expectedRate := matchedItem.RateAmount

			if actualRate != expectedRate {
				rateVariance := actualRate - expectedRate
				rateVariancePct := (rateVariance / expectedRate) * 100

				lineItem.RateVarianceAmount = sql.NullFloat64{Float64: rateVariance, Valid: true}
				lineItem.RateVariancePercentage = sql.NullFloat64{Float64: rateVariancePct, Valid: true}
				lineItem.HasVariance = true

				// Determine severity based on variance percentage
				severity := determineSeverity(rateVariancePct)
				lineItem.VarianceSeverity = sql.NullString{String: severity, Valid: true}

				if severity == "high" || severity == "critical" {
					lineItem.NeedsReview = true
				}

				// Create variance record
				variance := FinancialVariance{
					VarianceID:         uuid.New(),
					ProgramID:          invoice.ProgramID,
					InvoiceID:          uuid.NullUUID{UUID: invoice.InvoiceID, Valid: true},
					LineItemID:         uuid.NullUUID{UUID: lineItem.LineItemID, Valid: true},
					VarianceType:       "rate_overage",
					Severity:           severity,
					Title:              fmt.Sprintf("Rate variance for %s: billed at $%.2f/hr vs expected $%.2f/hr", lineItem.PersonName.String, actualRate, expectedRate),
					Description:        fmt.Sprintf("Person %s was billed at $%.2f per hour, but the rate card specifies $%.2f per hour. Variance: $%.2f (%.1f%%)", lineItem.PersonName.String, actualRate, expectedRate, rateVariance, rateVariancePct),
					ExpectedValue:      sql.NullFloat64{Float64: expectedRate, Valid: true},
					ActualValue:        sql.NullFloat64{Float64: actualRate, Valid: true},
					VarianceAmount:     sql.NullFloat64{Float64: rateVariance, Valid: true},
					VariancePercentage: sql.NullFloat64{Float64: rateVariancePct, Valid: true},
					SourceArtifactIDs:  []uuid.UUID{},
					AIConfidenceScore:  sql.NullFloat64{Float64: 0.95, Valid: true},
					AIDetectedAt:       time.Now(),
					IsDismissed:        false,
				}

				if invoice.ArtifactID.Valid {
					variance.SourceArtifactIDs = append(variance.SourceArtifactIDs, invoice.ArtifactID.UUID)
				}

				variances = append(variances, variance)
			}
		}

		// Check hours variance
		if lineItem.BilledHours.Valid && matchedItem.ExpectedHoursPerWeek.Valid {
			billedHours := lineItem.BilledHours.Float64
			expectedHours := matchedItem.ExpectedHoursPerWeek.Float64

			lineItem.ExpectedHours = sql.NullFloat64{Float64: expectedHours, Valid: true}

			if billedHours > expectedHours {
				hoursVariance := billedHours - expectedHours
				hoursVariancePct := (hoursVariance / expectedHours) * 100

				lineItem.HoursVariance = sql.NullFloat64{Float64: hoursVariance, Valid: true}
				lineItem.HasVariance = true

				// Determine severity
				severity := determineSeverity(hoursVariancePct)
				if !lineItem.VarianceSeverity.Valid || severity == "critical" {
					lineItem.VarianceSeverity = sql.NullString{String: severity, Valid: true}
				}

				if severity == "high" || severity == "critical" {
					lineItem.NeedsReview = true
				}

				// Create variance record
				variance := FinancialVariance{
					VarianceID:         uuid.New(),
					ProgramID:          invoice.ProgramID,
					InvoiceID:          uuid.NullUUID{UUID: invoice.InvoiceID, Valid: true},
					LineItemID:         uuid.NullUUID{UUID: lineItem.LineItemID, Valid: true},
					VarianceType:       "hours_overage",
					Severity:           severity,
					Title:              fmt.Sprintf("Hours variance for %s: billed %.1f hours vs expected %.1f hours/week", lineItem.PersonName.String, billedHours, expectedHours),
					Description:        fmt.Sprintf("Person %s was billed for %.1f hours, but the rate card expects %.1f hours per week. Overage: %.1f hours (%.1f%%)", lineItem.PersonName.String, billedHours, expectedHours, hoursVariance, hoursVariancePct),
					ExpectedValue:      sql.NullFloat64{Float64: expectedHours, Valid: true},
					ActualValue:        sql.NullFloat64{Float64: billedHours, Valid: true},
					VarianceAmount:     sql.NullFloat64{Float64: hoursVariance, Valid: true},
					VariancePercentage: sql.NullFloat64{Float64: hoursVariancePct, Valid: true},
					SourceArtifactIDs:  []uuid.UUID{},
					AIConfidenceScore:  sql.NullFloat64{Float64: 0.95, Valid: true},
					AIDetectedAt:       time.Now(),
					IsDismissed:        false,
				}

				if invoice.ArtifactID.Valid {
					variance.SourceArtifactIDs = append(variance.SourceArtifactIDs, invoice.ArtifactID.UUID)
				}

				variances = append(variances, variance)
			}
		}

		// Update line item in database
		if err := a.repo.UpdateLineItem(ctx, lineItem); err != nil {
			return nil, fmt.Errorf("failed to update line item: %w", err)
		}
	}

	return variances, nil
}

// DetectCrossDocumentConflicts checks invoice against other artifacts for conflicts
func (a *InvoiceAnalyzer) DetectCrossDocumentConflicts(ctx context.Context, invoice *Invoice, lineItems []InvoiceLineItem, programContext *ai.ProgramContext) ([]FinancialVariance, error) {
	// This method queries other artifacts in the program to find conflicting information
	// For example: planning documents that specify different rates or hours

	// Build prompt for cross-document analysis
	systemPrompt := `You are an expert financial auditor. Analyze invoice line items against program context (planning documents, contracts, rate cards) to detect conflicts.

Look for:
1. Person billed at rate X, but documents say rate Y
2. Person billed for X hours, but planning says Y hours
3. Person on invoice not mentioned in planning documents
4. Billing period mismatches

Return JSON array of conflicts:
[
  {
    "variance_type": "cross_document_conflict",
    "severity": "low|medium|high|critical",
    "title": "Brief title",
    "description": "Detailed description with evidence",
    "person_name": "string",
    "expected_value": number,
    "actual_value": number,
    "source_documents": ["doc names"],
    "confidence": 0.0-1.0
  }
]`

	// Build context from line items
	var lineItemsJSON bytes.Buffer
	lineItemsData := make([]map[string]interface{}, len(lineItems))
	for i, li := range lineItems {
		lineItemsData[i] = map[string]interface{}{
			"line_number":      li.LineNumber,
			"description":      li.Description,
			"person_name":      li.PersonName.String,
			"role_description": li.RoleDescription.String,
			"billed_hours":     li.BilledHours.Float64,
			"unit_rate":        li.UnitRate.Float64,
			"line_amount":      li.LineAmount,
		}
	}
	json.NewEncoder(&lineItemsJSON).Encode(lineItemsData)

	userPrompt := fmt.Sprintf(`Invoice Details:
Vendor: %s
Invoice Date: %s
Period: %s to %s

Line Items:
%s

Program Context:
%s

Analyze these invoice line items against the program context. Identify any conflicts or discrepancies.`,
		invoice.VendorName,
		invoice.InvoiceDate.Format("2006-01-02"),
		formatNullTime(invoice.PeriodStartDate),
		formatNullTime(invoice.PeriodEndDate),
		lineItemsJSON.String(),
		programContext.ToPromptString(),
	)

	// Call Claude API
	resp, err := a.client.RequestWithContext(
		ctx,
		ai.ModelSonnet4,
		systemPrompt,
		programContext.ToPromptString(), // Cache program context
		userPrompt,
		4096,
	)

	if err != nil {
		return nil, fmt.Errorf("Claude API request failed: %w", err)
	}

	// Parse response
	responseText := resp.GetExtractedText()
	responseText = stripMarkdownCodeBlocks(responseText)

	var conflictResults []struct {
		VarianceType    string   `json:"variance_type"`
		Severity        string   `json:"severity"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		PersonName      string   `json:"person_name,omitempty"`
		ExpectedValue   float64  `json:"expected_value,omitempty"`
		ActualValue     float64  `json:"actual_value,omitempty"`
		SourceDocuments []string `json:"source_documents"`
		Confidence      float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(responseText), &conflictResults); err != nil {
		// If no conflicts found or parsing fails, return empty
		return []FinancialVariance{}, nil
	}

	// Convert to domain models
	var variances []FinancialVariance
	for _, conflict := range conflictResults {
		// Find matching line item
		var lineItemID uuid.NullUUID
		for _, li := range lineItems {
			if li.PersonName.Valid && li.PersonName.String == conflict.PersonName {
				lineItemID = uuid.NullUUID{UUID: li.LineItemID, Valid: true}
				break
			}
		}

		variance := FinancialVariance{
			VarianceID:         uuid.New(),
			ProgramID:          invoice.ProgramID,
			InvoiceID:          uuid.NullUUID{UUID: invoice.InvoiceID, Valid: true},
			LineItemID:         lineItemID,
			VarianceType:       conflict.VarianceType,
			Severity:           conflict.Severity,
			Title:              conflict.Title,
			Description:        conflict.Description,
			ExpectedValue:      toNullFloat64(conflict.ExpectedValue),
			ActualValue:        toNullFloat64(conflict.ActualValue),
			SourceArtifactIDs:  []uuid.UUID{},
			AIConfidenceScore:  toNullFloat64(conflict.Confidence),
			AIDetectedAt:       time.Now(),
			IsDismissed:        false,
		}

		if invoice.ArtifactID.Valid {
			variance.SourceArtifactIDs = append(variance.SourceArtifactIDs, invoice.ArtifactID.UUID)
		}

		// Store source documents in conflicting_values JSONB
		conflictingData := map[string]interface{}{
			"source_documents": conflict.SourceDocuments,
			"person_name":      conflict.PersonName,
		}
		conflictingJSON, _ := json.Marshal(conflictingData)
		variance.ConflictingValues = conflictingJSON

		variances = append(variances, variance)
	}

	return variances, nil
}

// CalculateVariances combines rate card validation and cross-document analysis
func (a *InvoiceAnalyzer) CalculateVariances(ctx context.Context, invoice *Invoice, lineItems []InvoiceLineItem, programContext *ai.ProgramContext) (*VarianceDetectionResult, error) {
	startTime := time.Now()

	var allVariances []FinancialVariance

	// Validate against rate cards
	rateCardVariances, err := a.ValidateAgainstRateCards(ctx, invoice, lineItems)
	if err != nil {
		return nil, fmt.Errorf("failed to validate against rate cards: %w", err)
	}
	allVariances = append(allVariances, rateCardVariances...)

	// Detect cross-document conflicts
	crossDocVariances, err := a.DetectCrossDocumentConflicts(ctx, invoice, lineItems, programContext)
	if err != nil {
		// Log error but don't fail - cross-document analysis is optional
		fmt.Printf("Warning: cross-document analysis failed: %v\n", err)
	} else {
		allVariances = append(allVariances, crossDocVariances...)
	}

	result := &VarianceDetectionResult{
		Variances:      allVariances,
		ProcessingTime: time.Since(startTime),
		TokensUsed:     0, // Would be calculated from API responses
		Cost:           0, // Would be calculated from API responses
	}

	return result, nil
}

// ProcessInvoice performs complete invoice analysis: extract + validate + detect variances
func (a *InvoiceAnalyzer) ProcessInvoice(ctx context.Context, artifactID uuid.UUID, artifactContent string, programID uuid.UUID, programContext *ai.ProgramContext) error {
	// Extract invoice data
	invoice, lineItems, err := a.AnalyzeInvoice(ctx, artifactContent, programID)
	if err != nil {
		return fmt.Errorf("failed to analyze invoice: %w", err)
	}

	// Link to artifact
	invoice.ArtifactID = uuid.NullUUID{UUID: artifactID, Valid: true}

	// Save invoice to database
	if err := a.repo.CreateInvoice(ctx, invoice); err != nil {
		return fmt.Errorf("failed to save invoice: %w", err)
	}

	// Save line items
	if err := a.repo.SaveLineItems(ctx, lineItems); err != nil {
		return fmt.Errorf("failed to save line items: %w", err)
	}

	// Calculate variances
	varianceResult, err := a.CalculateVariances(ctx, invoice, lineItems, programContext)
	if err != nil {
		return fmt.Errorf("failed to calculate variances: %w", err)
	}

	// Save variances
	if len(varianceResult.Variances) > 0 {
		if err := a.repo.SaveVariances(ctx, varianceResult.Variances); err != nil {
			return fmt.Errorf("failed to save variances: %w", err)
		}
	}

	// Update invoice status
	invoice.ProcessingStatus = "validated"
	if err := a.repo.UpdateInvoice(ctx, invoice); err != nil {
		return fmt.Errorf("failed to update invoice status: %w", err)
	}

	return nil
}

// Helper functions

func stripMarkdownCodeBlocks(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		firstNewline := strings.Index(text, "\n")
		if firstNewline > 0 {
			text = text[firstNewline+1:]
		}
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func toNullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: f != 0}
}

func toNullTime(dateStr string) sql.NullTime {
	if dateStr == "" {
		return sql.NullTime{Valid: false}
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}

func formatNullTime(nt sql.NullTime) string {
	if !nt.Valid {
		return "N/A"
	}
	return nt.Time.Format("2006-01-02")
}

func determineSeverity(variancePct float64) string {
	absVariance := variancePct
	if absVariance < 0 {
		absVariance = -absVariance
	}

	switch {
	case absVariance >= 50:
		return "critical"
	case absVariance >= 25:
		return "high"
	case absVariance >= 10:
		return "medium"
	default:
		return "low"
	}
}
