package risk

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RiskDetector analyzes various inputs for risk indicators
type RiskDetector struct {
	repo RepositoryInterface
}

// NewRiskDetector creates a new risk detector
func NewRiskDetector(repo *Repository) *RiskDetector {
	return &RiskDetector{
		repo: repo,
	}
}

// ArtifactInsight represents an insight from an artifact (simplified for demo)
type ArtifactInsight struct {
	InsightID       uuid.UUID
	ArtifactID      uuid.UUID
	ProgramID       uuid.UUID
	InsightType     string
	Title           string
	Description     string
	Severity        string
	ConfidenceScore float64
}

// FinancialVariance represents a financial variance (simplified for demo)
type FinancialVariance struct {
	VarianceID      uuid.UUID
	ProgramID       uuid.UUID
	VarianceType    string
	Title           string
	Description     string
	Severity        string
	VarianceAmount  float64
	ConfidenceScore float64
	SourceArtifactIDs []uuid.UUID
}

// AnalyzeForRisks analyzes artifact insights for risk indicators
func (d *RiskDetector) AnalyzeForRisks(ctx context.Context, insights []ArtifactInsight) error {
	for _, insight := range insights {
		// Only create suggestions for risk-related insights
		if !isRiskRelatedInsight(insight.InsightType) {
			continue
		}

		// Create risk suggestion from insight
		suggestion := d.createSuggestionFromInsight(insight)

		// Check if similar suggestion already exists
		existingSuggestions, err := d.repo.ListSuggestions(ctx, insight.ProgramID, false)
		if err != nil {
			return fmt.Errorf("failed to check existing suggestions: %w", err)
		}

		// Skip if similar suggestion exists
		if d.hasSimilarSuggestion(suggestion, existingSuggestions) {
			continue
		}

		// Save suggestion
		err = d.repo.CreateSuggestion(ctx, suggestion)
		if err != nil {
			return fmt.Errorf("failed to create risk suggestion: %w", err)
		}
	}

	return nil
}

// ProcessFinancialVariance converts financial variances into risk suggestions
func (d *RiskDetector) ProcessFinancialVariance(ctx context.Context, variance FinancialVariance) error {
	// Create risk suggestion from variance
	suggestion := d.createSuggestionFromVariance(variance)

	// Check if similar suggestion already exists
	existingSuggestions, err := d.repo.ListSuggestions(ctx, variance.ProgramID, false)
	if err != nil {
		return fmt.Errorf("failed to check existing suggestions: %w", err)
	}

	// Skip if similar suggestion exists
	if d.hasSimilarSuggestion(suggestion, existingSuggestions) {
		return nil
	}

	// Save suggestion
	err = d.repo.CreateSuggestion(ctx, suggestion)
	if err != nil {
		return fmt.Errorf("failed to create risk suggestion: %w", err)
	}

	return nil
}

// createSuggestionFromInsight converts an artifact insight to a risk suggestion
func (d *RiskDetector) createSuggestionFromInsight(insight ArtifactInsight) *RiskSuggestion {
	// Map insight type to risk category
	category := mapInsightToRiskCategory(insight.InsightType)

	// Map severity to probability and impact
	probability, impact := mapSeverityToProbabilityImpact(insight.Severity)

	// Generate rationale
	rationale := fmt.Sprintf(
		"AI detected a %s insight from document analysis with %0.1f%% confidence. "+
			"The insight type '%s' indicates potential %s risk that requires attention.",
		insight.Severity,
		insight.ConfidenceScore*100,
		insight.InsightType,
		category,
	)

	return &RiskSuggestion{
		SuggestionID:         uuid.New(),
		ProgramID:            insight.ProgramID,
		Title:                insight.Title,
		Description:          insight.Description,
		Rationale:            rationale,
		SuggestedProbability: probability,
		SuggestedImpact:      impact,
		SuggestedCategory:    category,
		SourceType:           "artifact_insight",
		SourceArtifactIDs:    []uuid.UUID{insight.ArtifactID},
		SourceInsightID:      uuid.NullUUID{UUID: insight.InsightID, Valid: true},
		AIConfidenceScore:    sql.NullFloat64{Float64: insight.ConfidenceScore, Valid: true},
		AIDetectedAt:         time.Now(),
		IsApproved:           false,
		IsDismissed:          false,
	}
}

// createSuggestionFromVariance converts a financial variance to a risk suggestion
func (d *RiskDetector) createSuggestionFromVariance(variance FinancialVariance) *RiskSuggestion {
	// Generate title based on variance type
	title := fmt.Sprintf("Financial Risk: %s", variance.Title)

	// Generate description with variance details
	description := fmt.Sprintf(
		"%s\n\nVariance Amount: $%.2f\n\nThis financial discrepancy may indicate "+
			"budget overrun, billing issues, or resource allocation problems that require "+
			"immediate attention and corrective action.",
		variance.Description,
		variance.VarianceAmount,
	)

	// Generate rationale
	rationale := fmt.Sprintf(
		"AI detected a %s financial variance of $%.2f with %0.1f%% confidence. "+
			"Financial variances of this magnitude typically indicate budget risks, "+
			"cost control issues, or billing discrepancies that could impact program delivery.",
		variance.Severity,
		variance.VarianceAmount,
		variance.ConfidenceScore*100,
	)

	// Map severity to probability and impact
	probability, impact := mapVarianceSeverityToRisk(variance.Severity, variance.VarianceAmount)

	return &RiskSuggestion{
		SuggestionID:         uuid.New(),
		ProgramID:            variance.ProgramID,
		Title:                title,
		Description:          description,
		Rationale:            rationale,
		SuggestedProbability: probability,
		SuggestedImpact:      impact,
		SuggestedCategory:    "financial",
		SourceType:           "financial_variance",
		SourceArtifactIDs:    variance.SourceArtifactIDs,
		SourceVarianceID:     uuid.NullUUID{UUID: variance.VarianceID, Valid: true},
		AIConfidenceScore:    sql.NullFloat64{Float64: variance.ConfidenceScore, Valid: true},
		AIDetectedAt:         time.Now(),
		IsApproved:           false,
		IsDismissed:          false,
	}
}

// hasSimilarSuggestion checks if a similar suggestion already exists
func (d *RiskDetector) hasSimilarSuggestion(suggestion *RiskSuggestion, existing []RiskSuggestion) bool {
	titleLower := strings.ToLower(suggestion.Title)

	for _, existingSuggestion := range existing {
		// Check for exact title match
		if strings.ToLower(existingSuggestion.Title) == titleLower {
			return true
		}

		// Check for same source insight or variance
		if suggestion.SourceInsightID.Valid && existingSuggestion.SourceInsightID.Valid {
			if suggestion.SourceInsightID.UUID == existingSuggestion.SourceInsightID.UUID {
				return true
			}
		}
		if suggestion.SourceVarianceID.Valid && existingSuggestion.SourceVarianceID.Valid {
			if suggestion.SourceVarianceID.UUID == existingSuggestion.SourceVarianceID.UUID {
				return true
			}
		}
	}

	return false
}

// Mapping helpers

func isRiskRelatedInsight(insightType string) bool {
	riskInsightTypes := map[string]bool{
		"risk":              true,
		"anomaly":           true,
		"action_required":   true,
		"concern":           true,
		"deadline_miss":     true,
		"budget_overrun":    true,
		"resource_shortage": true,
		"quality_issue":     true,
		"dependency_risk":   true,
		"compliance_issue":  true,
	}
	return riskInsightTypes[insightType]
}

func mapInsightToRiskCategory(insightType string) string {
	// Map insight types to risk categories
	categoryMap := map[string]string{
		"risk":              "technical",
		"anomaly":           "technical",
		"action_required":   "schedule",
		"deadline_miss":     "schedule",
		"budget_overrun":    "financial",
		"resource_shortage": "resource",
		"quality_issue":     "technical",
		"dependency_risk":   "external",
		"compliance_issue":  "external",
	}

	if category, ok := categoryMap[insightType]; ok {
		return category
	}
	return "technical" // Default
}

func mapSeverityToProbabilityImpact(severity string) (string, string) {
	// Map severity levels to probability and impact
	severityMap := map[string]struct{ probability, impact string }{
		"critical": {"high", "very_high"},
		"high":     {"high", "high"},
		"medium":   {"medium", "medium"},
		"low":      {"low", "low"},
	}

	if mapping, ok := severityMap[severity]; ok {
		return mapping.probability, mapping.impact
	}
	return "medium", "medium" // Default
}

func mapVarianceSeverityToRisk(severity string, amount float64) (string, string) {
	// For financial variances, consider both severity and amount

	// High variance amounts suggest higher probability
	probability := "medium"
	if amount > 100000 {
		probability = "high"
	} else if amount > 50000 {
		probability = "medium"
	} else {
		probability = "low"
	}

	// Severity maps directly to impact
	impactMap := map[string]string{
		"critical": "very_high",
		"high":     "high",
		"medium":   "medium",
		"low":      "low",
	}

	impact := impactMap[severity]
	if impact == "" {
		impact = "medium"
	}

	return probability, impact
}


// DetectRiskFromInsight is a public helper for creating risk suggestions from insights
func DetectRiskFromInsight(ctx context.Context, repo RepositoryInterface, insight ArtifactInsight) error {
	detector := &RiskDetector{repo: repo}
	return detector.AnalyzeForRisks(ctx, []ArtifactInsight{insight})
}

// DetectRiskFromFinancialVariance is a public helper for creating risk suggestions from variances
func DetectRiskFromFinancialVariance(ctx context.Context, repo RepositoryInterface, variance FinancialVariance) error {
	detector := &RiskDetector{repo: repo}
	return detector.ProcessFinancialVariance(ctx, variance)
}

// AnalyzeScheduleRisk analyzes schedule-related risks (placeholder for future implementation)
func (d *RiskDetector) AnalyzeScheduleRisk(ctx context.Context, programID uuid.UUID) error {
	// TODO: Implement schedule risk analysis
	// - Analyze task delays
	// - Check milestone slippage
	// - Identify critical path risks
	// - Detect resource bottlenecks

	return nil
}

// AnalyzeDependencyRisk analyzes external dependency risks (placeholder for future implementation)
func (d *RiskDetector) AnalyzeDependencyRisk(ctx context.Context, programID uuid.UUID) error {
	// TODO: Implement dependency risk analysis
	// - Identify external dependencies
	// - Assess vendor reliability
	// - Check contract expiration dates
	// - Analyze supply chain risks

	return nil
}
