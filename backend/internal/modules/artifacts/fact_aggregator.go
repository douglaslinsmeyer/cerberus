package artifacts

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
)

// FactAggregator aggregates facts across multiple artifacts
// and detects conflicts or patterns
type FactAggregator struct {
	repo RepositoryInterface
}

// NewFactAggregator creates a new fact aggregator
func NewFactAggregator(repo RepositoryInterface) *FactAggregator {
	return &FactAggregator{
		repo: repo,
	}
}

// AggregateRelatedFacts collects and aggregates facts from related artifacts
func (fa *FactAggregator) AggregateRelatedFacts(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	relatedArtifactIDs []uuid.UUID,
) (*AggregatedFactsContext, error) {
	// Include target artifact in the query
	allArtifactIDs := append([]uuid.UUID{targetArtifactID}, relatedArtifactIDs...)

	// Get all facts from these artifacts
	facts, err := fa.repo.GetFactsByArtifacts(ctx, allArtifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}

	// Group facts by type
	financialFacts := []Fact{}
	dateFacts := []Fact{}
	metricFacts := []Fact{}

	for _, fact := range facts {
		switch fact.FactType {
		case "amount", "currency", "financial":
			financialFacts = append(financialFacts, fact)
		case "date", "deadline", "milestone":
			dateFacts = append(dateFacts, fact)
		case "metric", "count", "percentage":
			metricFacts = append(metricFacts, fact)
		}
	}

	// Aggregate each type
	aggregatedFinancial := fa.aggregateFactsByKey(ctx, financialFacts)
	aggregatedDates := fa.aggregateFactsByKey(ctx, dateFacts)
	aggregatedMetrics := fa.aggregateFactsByKey(ctx, metricFacts)

	// Detect conflicts
	conflicts := fa.detectConflicts(ctx, facts)

	// Estimate tokens
	tokensPerFact := 30
	tokensPerConflict := 50
	estimatedTokens := (len(aggregatedFinancial)+len(aggregatedDates)+len(aggregatedMetrics))*tokensPerFact +
		len(conflicts)*tokensPerConflict

	return &AggregatedFactsContext{
		FinancialFacts:  aggregatedFinancial,
		DateFacts:       aggregatedDates,
		MetricFacts:     aggregatedMetrics,
		Conflicts:       conflicts,
		EstimatedTokens: estimatedTokens,
	}, nil
}

// aggregateFactsByKey groups facts by key and aggregates their sources
func (fa *FactAggregator) aggregateFactsByKey(ctx context.Context, facts []Fact) []AggregatedFact {
	// Group by fact key
	factGroups := make(map[string][]Fact)
	for _, fact := range facts {
		key := fact.FactKey
		factGroups[key] = append(factGroups[key], fact)
	}

	// Create aggregated facts
	aggregated := []AggregatedFact{}

	for key, group := range factGroups {
		// If all values are the same, it's a confirmed fact
		// If values differ, it's a conflict (handled separately)

		// Check for consensus
		valueCount := make(map[string]int)
		var mostCommonValue string
		maxCount := 0

		for _, fact := range group {
			valueCount[fact.FactValue]++
			if valueCount[fact.FactValue] > maxCount {
				maxCount = valueCount[fact.FactValue]
				mostCommonValue = fact.FactValue
			}
		}

		// Get source artifact filenames
		sourceArtifacts := []string{}
		for _, fact := range group {
			filename, err := fa.repo.GetArtifactFilename(ctx, fact.ArtifactID)
			if err == nil {
				sourceArtifacts = append(sourceArtifacts, filename)
			}
		}

		// Calculate average confidence
		avgConfidence := 0.0
		confidenceCount := 0
		for _, fact := range group {
			if fact.ConfidenceScore.Valid {
				avgConfidence += fact.ConfidenceScore.Float64
				confidenceCount++
			}
		}
		if confidenceCount > 0 {
			avgConfidence /= float64(confidenceCount)
		}

		aggregatedFact := AggregatedFact{
			FactKey:         key,
			FactValue:       mostCommonValue,
			FactType:        group[0].FactType,
			SourceArtifacts: fa.uniqueStrings(sourceArtifacts),
			OccurrenceCount: len(group),
			ConfidenceScore: avgConfidence,
		}

		aggregated = append(aggregated, aggregatedFact)
	}

	// Sort by occurrence count (most common first)
	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].OccurrenceCount > aggregated[j].OccurrenceCount
	})

	// Limit to top 20
	if len(aggregated) > 20 {
		aggregated = aggregated[:20]
	}

	return aggregated
}

// detectConflicts identifies conflicting facts across artifacts
func (fa *FactAggregator) detectConflicts(ctx context.Context, facts []Fact) []FactConflict {
	conflicts := []FactConflict{}

	// Group facts by key
	factGroups := make(map[string][]Fact)
	for _, fact := range facts {
		factGroups[fact.FactKey] = append(factGroups[fact.FactKey], fact)
	}

	// For each key, check if values differ
	for key, group := range factGroups {
		if len(group) < 2 {
			continue
		}

		// Check if values are all the same
		firstValue := group[0].FactValue
		hasConflict := false

		for _, fact := range group[1:] {
			if fact.FactValue != firstValue {
				hasConflict = true
				break
			}
		}

		if !hasConflict {
			continue
		}

		// Build conflict object
		conflictingValues := []ConflictingValue{}
		seenValues := make(map[string]bool)

		for _, fact := range group {
			// Skip duplicate values from same artifact
			valueKey := fmt.Sprintf("%s-%s", fact.FactValue, fact.ArtifactID.String())
			if seenValues[valueKey] {
				continue
			}
			seenValues[valueKey] = true

			filename, err := fa.repo.GetArtifactFilename(ctx, fact.ArtifactID)
			if err != nil {
				filename = "Unknown"
			}

			confidence := 0.0
			if fact.ConfidenceScore.Valid {
				confidence = fact.ConfidenceScore.Float64
			}

			conflictingValues = append(conflictingValues, ConflictingValue{
				Value:            fact.FactValue,
				SourceArtifactID: fact.ArtifactID,
				SourceFilename:   filename,
				ConfidenceScore:  confidence,
			})
		}

		// Determine severity
		severity := fa.calculateConflictSeverity(group)

		conflict := FactConflict{
			FactKey:           key,
			ConflictingValues: conflictingValues,
			Severity:          severity,
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts
}

// calculateConflictSeverity determines how severe a conflict is
func (fa *FactAggregator) calculateConflictSeverity(facts []Fact) string {
	// Severity factors:
	// 1. Number of conflicting values (more = worse)
	// 2. Confidence scores (high confidence conflicts are worse)
	// 3. Type of fact (financial conflicts are worse than dates)

	uniqueValues := make(map[string]bool)
	for _, fact := range facts {
		uniqueValues[fact.FactValue] = true
	}

	avgConfidence := 0.0
	confidenceCount := 0
	for _, fact := range facts {
		if fact.ConfidenceScore.Valid {
			avgConfidence += fact.ConfidenceScore.Float64
			confidenceCount++
		}
	}
	if confidenceCount > 0 {
		avgConfidence /= float64(confidenceCount)
	}

	// Determine severity
	if len(uniqueValues) > 3 {
		return "major" // Many conflicting values
	}

	if avgConfidence > 0.8 && len(uniqueValues) > 2 {
		return "major" // High confidence but conflicting
	}

	if facts[0].FactType == "amount" || facts[0].FactType == "financial" {
		return "major" // Financial conflicts are serious
	}

	return "minor"
}

// FindRelatedFacts finds facts related to a specific query
func (fa *FactAggregator) FindRelatedFacts(
	ctx context.Context,
	programID uuid.UUID,
	factKey string,
) ([]Fact, error) {
	return fa.repo.FindFactsByKey(ctx, programID, factKey)
}

// GetFactStats returns statistics about facts in a program
func (fa *FactAggregator) GetFactStats(
	ctx context.Context,
	programID uuid.UUID,
) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total facts
	totalFacts, err := fa.repo.CountFactsInProgram(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats["total_facts"] = totalFacts

	// Count by type
	factsByType, err := fa.repo.CountFactsByType(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats["facts_by_type"] = factsByType

	// Count conflicts
	allFacts, err := fa.repo.GetAllFactsInProgram(ctx, programID)
	if err != nil {
		return nil, err
	}
	conflicts := fa.detectConflicts(ctx, allFacts)
	stats["total_conflicts"] = len(conflicts)

	majorConflicts := 0
	for _, conflict := range conflicts {
		if conflict.Severity == "major" {
			majorConflicts++
		}
	}
	stats["major_conflicts"] = majorConflicts

	return stats, nil
}

// Helper functions

func (fa *FactAggregator) uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range strs {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// CompareNumericFacts compares numeric facts to detect anomalies
func (fa *FactAggregator) CompareNumericFacts(
	ctx context.Context,
	facts []Fact,
) (map[string]interface{}, error) {
	// Extract numeric values
	values := []float64{}
	for _, fact := range facts {
		if fact.NormalizedValueNumeric.Valid {
			values = append(values, fact.NormalizedValueNumeric.Float64)
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no numeric values found")
	}

	// Calculate statistics
	mean := fa.mean(values)
	stdDev := fa.stdDeviation(values, mean)
	min := fa.min(values)
	max := fa.max(values)

	// Identify outliers (values > 2 std devs from mean)
	outliers := []float64{}
	for _, v := range values {
		if math.Abs(v-mean) > 2*stdDev {
			outliers = append(outliers, v)
		}
	}

	return map[string]interface{}{
		"mean":     mean,
		"std_dev":  stdDev,
		"min":      min,
		"max":      max,
		"outliers": outliers,
		"count":    len(values),
	}, nil
}

// Statistical helper functions

func (fa *FactAggregator) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (fa *FactAggregator) stdDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func (fa *FactAggregator) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (fa *FactAggregator) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

// FormatFactsForPrompt formats aggregated facts for the AI prompt
func (fa *FactAggregator) FormatFactsForPrompt(factsCtx *AggregatedFactsContext) string {
	if factsCtx == nil ||
		(len(factsCtx.FinancialFacts) == 0 && len(factsCtx.DateFacts) == 0 &&
			len(factsCtx.MetricFacts) == 0 && len(factsCtx.Conflicts) == 0) {
		return "No aggregated facts available."
	}

	output := "Aggregated Facts from Related Documents:\n\n"

	if len(factsCtx.FinancialFacts) > 0 {
		output += "Financial Facts:\n"
		for _, fact := range factsCtx.FinancialFacts {
			sourceList := ""
			if len(fact.SourceArtifacts) > 3 {
				sourceList = fmt.Sprintf("%s, %s, +%d more",
					fact.SourceArtifacts[0], fact.SourceArtifacts[1],
					len(fact.SourceArtifacts)-2)
			} else {
				sourceList = fmt.Sprintf("%v", fact.SourceArtifacts)
			}

			output += fmt.Sprintf("- %s: %s (from %s, confidence: %.2f)\n",
				fact.FactKey, fact.FactValue, sourceList, fact.ConfidenceScore)
		}
		output += "\n"
	}

	if len(factsCtx.DateFacts) > 0 {
		output += "Date Facts:\n"
		for _, fact := range factsCtx.DateFacts {
			output += fmt.Sprintf("- %s: %s (mentioned %d times)\n",
				fact.FactKey, fact.FactValue, fact.OccurrenceCount)
		}
		output += "\n"
	}

	if len(factsCtx.MetricFacts) > 0 {
		output += "Metric Facts:\n"
		for _, fact := range factsCtx.MetricFacts {
			output += fmt.Sprintf("- %s: %s (confidence: %.2f)\n",
				fact.FactKey, fact.FactValue, fact.ConfidenceScore)
		}
		output += "\n"
	}

	if len(factsCtx.Conflicts) > 0 {
		output += "⚠️ CONFLICTS DETECTED:\n"
		for _, conflict := range factsCtx.Conflicts {
			output += fmt.Sprintf("- \"%s\" has %d conflicting values:\n",
				conflict.FactKey, len(conflict.ConflictingValues))

			for _, value := range conflict.ConflictingValues {
				output += fmt.Sprintf("  • %s (from %s, confidence: %.2f)\n",
					value.Value, value.SourceFilename, value.ConfidenceScore)
			}
		}
	}

	return output
}
