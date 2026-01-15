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

// EnrichExistingRisks detects when new insights are related to existing risks and creates enrichments
func (d *RiskDetector) EnrichExistingRisks(ctx context.Context, insights []ArtifactInsight) error {
	for _, insight := range insights {
		// Get active risks for this program (status: open or monitoring)
		risks, err := d.getActiveRisks(ctx, insight.ProgramID)
		if err != nil {
			return fmt.Errorf("failed to get active risks: %w", err)
		}

		// For each risk, calculate match score
		for _, risk := range risks {
			matchScore, matchMethod := d.calculateMatchScore(insight, risk)

			// Only create enrichment if match score exceeds threshold
			if matchScore < 0.5 {
				continue
			}

			// Determine enrichment type based on insight type
			enrichmentType := determineEnrichmentType(insight.InsightType)

			// Create enrichment record
			enrichment := &RiskEnrichment{
				EnrichmentID:    uuid.New(),
				RiskID:          risk.RiskID,
				ArtifactID:      insight.ArtifactID,
				SourceInsightID: uuid.NullUUID{UUID: insight.InsightID, Valid: true},
				EnrichmentType:  enrichmentType,
				Title:           insight.Title,
				Content:         insight.Description,
				MatchScore:      matchScore,
				MatchMethod:     matchMethod,
				CreatedAt:       time.Now(),
			}

			// Save enrichment
			err = d.repo.CreateEnrichment(ctx, enrichment)
			if err != nil {
				// Log error but continue processing other insights
				fmt.Printf("Warning: failed to create enrichment: %v\n", err)
				continue
			}
		}
	}

	return nil
}

// getActiveRisks retrieves risks that are open or monitoring status
func (d *RiskDetector) getActiveRisks(ctx context.Context, programID uuid.UUID) ([]Risk, error) {
	// Get open risks
	openFilter := RiskFilterRequest{
		ProgramID: programID,
		Status:    "open",
		Limit:     100, // Limit to recent risks
	}
	openRisks, err := d.repo.ListRisks(ctx, openFilter)
	if err != nil {
		return nil, err
	}

	// Get monitoring risks
	monitoringFilter := RiskFilterRequest{
		ProgramID: programID,
		Status:    "monitoring",
		Limit:     100,
	}
	monitoringRisks, err := d.repo.ListRisks(ctx, monitoringFilter)
	if err != nil {
		return nil, err
	}

	// Combine results
	return append(openRisks, monitoringRisks...), nil
}

// calculateMatchScore determines how well an insight matches a risk
func (d *RiskDetector) calculateMatchScore(insight ArtifactInsight, risk Risk) (float64, string) {
	var totalScore float64
	var method string

	// Category alignment score (0.4 weight)
	categoryScore := d.calculateCategoryScore(insight, risk)
	totalScore += categoryScore * 0.4

	// Severity alignment score (0.3 weight)
	severityScore := d.calculateSeverityScore(insight, risk)
	totalScore += severityScore * 0.3

	// Title/description similarity (0.3 weight)
	similarityScore := d.calculateTextSimilarity(insight, risk)
	totalScore += similarityScore * 0.3

	// Determine primary match method
	if categoryScore > 0.7 {
		method = "category_match"
	} else if similarityScore > 0.5 {
		method = "text_similarity"
	} else {
		method = "composite"
	}

	return totalScore, method
}

// calculateCategoryScore checks if insight type maps to risk category
func (d *RiskDetector) calculateCategoryScore(insight ArtifactInsight, risk Risk) float64 {
	insightCategory := mapInsightToRiskCategory(insight.InsightType)

	if strings.ToLower(insightCategory) == strings.ToLower(risk.Category) {
		return 1.0
	}

	// Partial matches
	if insightCategory == "technical" && (risk.Category == "quality" || risk.Category == "compliance") {
		return 0.6
	}
	if insightCategory == "schedule" && risk.Category == "resource" {
		return 0.5
	}

	return 0.0
}

// calculateSeverityScore compares insight severity with risk severity
func (d *RiskDetector) calculateSeverityScore(insight ArtifactInsight, risk Risk) float64 {
	insightSev := strings.ToLower(insight.Severity)
	riskSev := strings.ToLower(risk.Severity)

	if insightSev == riskSev {
		return 1.0
	}

	// Adjacent severity levels
	severityOrder := []string{"low", "medium", "high", "critical"}
	insightIdx := indexOf(severityOrder, insightSev)
	riskIdx := indexOf(severityOrder, riskSev)

	if insightIdx >= 0 && riskIdx >= 0 {
		diff := abs(insightIdx - riskIdx)
		if diff == 1 {
			return 0.7
		}
		if diff == 2 {
			return 0.4
		}
	}

	return 0.0
}

// calculateTextSimilarity checks for keyword overlap in title/description
func (d *RiskDetector) calculateTextSimilarity(insight ArtifactInsight, risk Risk) float64 {
	// Simple keyword matching
	insightText := strings.ToLower(insight.Title + " " + insight.Description)
	riskText := strings.ToLower(risk.Title + " " + risk.Description)

	// Extract keywords (words > 4 characters)
	insightWords := extractKeywords(insightText)
	riskWords := extractKeywords(riskText)

	if len(insightWords) == 0 || len(riskWords) == 0 {
		return 0.0
	}

	// Count common keywords
	common := 0
	for word := range insightWords {
		if riskWords[word] {
			common++
		}
	}

	// Jaccard similarity
	totalUnique := len(insightWords) + len(riskWords) - common
	if totalUnique == 0 {
		return 0.0
	}

	return float64(common) / float64(totalUnique)
}

// Helper functions

func determineEnrichmentType(insightType string) string {
	typeMap := map[string]string{
		"risk":              "new_evidence",
		"anomaly":           "new_evidence",
		"action_required":   "status_change",
		"concern":           "impact_update",
		"deadline_miss":     "status_change",
		"budget_overrun":    "impact_update",
		"resource_shortage": "impact_update",
		"quality_issue":     "new_evidence",
		"dependency_risk":   "new_evidence",
		"compliance_issue":  "new_evidence",
	}

	if enrichType, ok := typeMap[insightType]; ok {
		return enrichType
	}
	return "new_evidence"
}

func extractKeywords(text string) map[string]bool {
	words := strings.Fields(text)
	keywords := make(map[string]bool)

	// Common stop words to ignore
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "that": true, "this": true,
		"with": true, "from": true, "are": true, "was": true, "has": true,
		"have": true, "been": true, "will": true, "would": true, "should": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))
		if len(word) > 4 && !stopWords[word] {
			keywords[word] = true
		}
	}

	return keywords
}

func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
