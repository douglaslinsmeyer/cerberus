package artifacts

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ContextSelector implements the weighted scoring and selection algorithm
// for choosing the most relevant artifacts to include in enriched context.
//
// Scoring Formula:
//
//	Relevance Score = (
//	    0.40 × Semantic Similarity      // Vector cosine similarity
//	  + 0.25 × Entity Overlap          // Shared people/stakeholders
//	  + 0.20 × Temporal Proximity      // Recent documents prioritized
//	  + 0.10 × Document Type Match     // Same category artifacts
//	  + 0.05 × Fact Density            // Rich metadata artifacts
//	)
type ContextSelector struct {
	maxTokens int
	weights   ScoringWeights
}

// ScoringWeights defines the weight for each scoring component
type ScoringWeights struct {
	SemanticSimilarity float64 // Default: 0.40
	EntityOverlap      float64 // Default: 0.25
	TemporalProximity  float64 // Default: 0.20
	DocumentTypeMatch  float64 // Default: 0.10
	FactDensity        float64 // Default: 0.05
}

// DefaultScoringWeights returns the recommended scoring weights
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		SemanticSimilarity: 0.40,
		EntityOverlap:      0.25,
		TemporalProximity:  0.20,
		DocumentTypeMatch:  0.10,
		FactDensity:        0.05,
	}
}

// ArtifactCandidate represents a potential artifact to include in context
// with all the scoring components pre-computed
type ArtifactCandidate struct {
	// Core artifact data
	ArtifactID       uuid.UUID
	Filename         string
	Category         string
	Subcategory      string
	UploadedAt       time.Time
	ExecutiveSummary string
	Sentiment        string
	Priority         int

	// Entity data
	MentionedPeople []string
	PersonIDs       []uuid.UUID
	SharedPersonIDs []uuid.UUID // Intersection with target artifact

	// Topic data
	Topics   []string
	TopicIDs []uuid.UUID

	// Fact data
	FactCount int

	// Scoring components (0.0 - 1.0 each)
	SemanticScore  float64 // Cosine similarity from vector search
	EntityScore    float64 // Ratio of shared entities
	TemporalScore  float64 // Exponential decay based on time difference
	TypeScore      float64 // 1.0 if category matches, 0.0 otherwise
	DensityScore   float64 // Normalized fact count

	// Final computed score
	TotalScore float64

	// Token estimation
	EstimatedTokens int
}

// NewContextSelector creates a new context selector with the given configuration
func NewContextSelector(maxTokens int, weights ScoringWeights) *ContextSelector {
	return &ContextSelector{
		maxTokens: maxTokens,
		weights:   weights,
	}
}

// NewDefaultContextSelector creates a context selector with default settings
func NewDefaultContextSelector() *ContextSelector {
	return NewContextSelector(4000, DefaultScoringWeights())
}

// ScoreCandidate computes the total relevance score for a candidate artifact
// relative to the target artifact being analyzed
func (cs *ContextSelector) ScoreCandidate(
	candidate *ArtifactCandidate,
	targetArtifact *Artifact,
	targetPersonIDs []uuid.UUID,
) float64 {
	// Component 1: Semantic Similarity (already computed from vector search)
	semanticComponent := cs.weights.SemanticSimilarity * candidate.SemanticScore

	// Component 2: Entity Overlap
	entityComponent := cs.weights.EntityOverlap * cs.computeEntityScore(
		candidate.PersonIDs,
		targetPersonIDs,
	)
	candidate.EntityScore = cs.computeEntityScore(candidate.PersonIDs, targetPersonIDs)

	// Component 3: Temporal Proximity
	temporalComponent := cs.weights.TemporalProximity * cs.computeTemporalScore(
		candidate.UploadedAt,
		targetArtifact.UploadedAt,
	)
	candidate.TemporalScore = cs.computeTemporalScore(candidate.UploadedAt, targetArtifact.UploadedAt)

	// Component 4: Document Type Match
	targetCategory := ""
	if targetArtifact.ArtifactCategory.Valid {
		targetCategory = targetArtifact.ArtifactCategory.String
	}
	typeComponent := cs.weights.DocumentTypeMatch * cs.computeTypeScore(
		candidate.Category,
		targetCategory,
	)
	candidate.TypeScore = cs.computeTypeScore(candidate.Category, targetCategory)

	// Component 5: Fact Density
	densityComponent := cs.weights.FactDensity * cs.computeDensityScore(candidate.FactCount)
	candidate.DensityScore = cs.computeDensityScore(candidate.FactCount)

	// Compute total score
	totalScore := semanticComponent + entityComponent + temporalComponent +
		typeComponent + densityComponent

	candidate.TotalScore = totalScore

	return totalScore
}

// computeEntityScore calculates the entity overlap score (0-1)
// based on the ratio of shared people between candidate and target
func (cs *ContextSelector) computeEntityScore(
	candidatePersonIDs []uuid.UUID,
	targetPersonIDs []uuid.UUID,
) float64 {
	if len(targetPersonIDs) == 0 {
		return 0.0
	}

	// Count shared person IDs
	sharedCount := 0
	targetSet := make(map[uuid.UUID]bool)
	for _, id := range targetPersonIDs {
		targetSet[id] = true
	}

	for _, id := range candidatePersonIDs {
		if targetSet[id] {
			sharedCount++
		}
	}

	// Calculate overlap ratio (Jaccard similarity might be better, but this is simpler)
	maxPeople := math.Max(float64(len(candidatePersonIDs)), float64(len(targetPersonIDs)))
	if maxPeople == 0 {
		return 0.0
	}

	return float64(sharedCount) / maxPeople
}

// computeTemporalScore calculates temporal proximity score (0-1)
// using exponential decay with a 90-day half-life
func (cs *ContextSelector) computeTemporalScore(
	candidateTime time.Time,
	targetTime time.Time,
) float64 {
	// Calculate days difference
	daysDiff := math.Abs(candidateTime.Sub(targetTime).Hours() / 24)

	// Exponential decay: score = exp(-days / 90)
	// This gives ~0.5 score at 90 days, ~0.25 at 180 days, etc.
	decayConstant := 90.0
	score := math.Exp(-daysDiff / decayConstant)

	return score
}

// computeTypeScore returns 1.0 if categories match, 0.5 if related, 0.0 otherwise
func (cs *ContextSelector) computeTypeScore(
	candidateCategory string,
	targetCategory string,
) float64 {
	if candidateCategory == targetCategory {
		return 1.0
	}

	// Define related categories that get partial credit
	relatedCategories := map[string][]string{
		"contract":  {"invoice", "change_order", "amendment"},
		"invoice":   {"contract", "purchase_order"},
		"report":    {"meeting_notes", "status_report"},
		"email":     {"meeting_notes", "memo"},
		"amendment": {"contract", "change_order"},
	}

	if related, ok := relatedCategories[targetCategory]; ok {
		for _, rel := range related {
			if candidateCategory == rel {
				return 0.5 // Partial credit for related types
			}
		}
	}

	return 0.0
}

// computeDensityScore calculates fact density score (0-1)
// based on the number of extracted facts (capped at 20)
func (cs *ContextSelector) computeDensityScore(factCount int) float64 {
	// Normalize to 0-1 scale, capping at 20 facts
	maxFacts := 20.0
	normalizedCount := math.Min(float64(factCount), maxFacts)
	return normalizedCount / maxFacts
}

// EstimateTokens estimates the token count for including this artifact in context
func (cs *ContextSelector) EstimateTokens(candidate *ArtifactCandidate) int {
	// Token estimation formula:
	// - Filename: ~10 tokens
	// - Category + date: ~15 tokens
	// - Summary: length / 4 (rough approximation: 1 token ≈ 4 characters)
	// - People list: num_people * 8 tokens (name + role)
	// - Topics: num_topics * 3 tokens
	// - Base formatting: 20 tokens

	baseTokens := 20
	filenameTokens := 10
	metadataTokens := 15

	summaryTokens := len(candidate.ExecutiveSummary) / 4
	peopleTokens := len(candidate.MentionedPeople) * 8
	topicTokens := len(candidate.Topics) * 3

	total := baseTokens + filenameTokens + metadataTokens +
		summaryTokens + peopleTokens + topicTokens

	return total
}

// SelectTopN selects the top N candidates that fit within the token budget
// Candidates are sorted by TotalScore (highest first)
func (cs *ContextSelector) SelectTopN(
	candidates []ArtifactCandidate,
	tokenBudget int,
) []ArtifactCandidate {
	// Sort by total score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].TotalScore > candidates[j].TotalScore
	})

	// Select candidates within budget
	selected := []ArtifactCandidate{}
	tokensUsed := 0

	for _, candidate := range candidates {
		estimatedTokens := cs.EstimateTokens(&candidate)

		// Check if adding this candidate would exceed budget
		if tokensUsed+estimatedTokens <= tokenBudget {
			candidate.EstimatedTokens = estimatedTokens
			selected = append(selected, candidate)
			tokensUsed += estimatedTokens
		} else {
			// Budget exhausted, stop adding
			break
		}
	}

	return selected
}

// ScoreAndSelectContext is a convenience method that scores all candidates
// and selects the top ones within the token budget
func (cs *ContextSelector) ScoreAndSelectContext(
	candidates []ArtifactCandidate,
	targetArtifact *Artifact,
	targetPersonIDs []uuid.UUID,
	tokenBudget int,
) []ArtifactCandidate {
	// Score all candidates
	for i := range candidates {
		cs.ScoreCandidate(&candidates[i], targetArtifact, targetPersonIDs)
	}

	// Select within budget
	return cs.SelectTopN(candidates, tokenBudget)
}

// GetScoreBreakdown returns a detailed breakdown of the score components
// for debugging and analysis
func (cs *ContextSelector) GetScoreBreakdown(candidate *ArtifactCandidate) map[string]interface{} {
	return map[string]interface{}{
		"total_score": candidate.TotalScore,
		"components": map[string]interface{}{
			"semantic": map[string]float64{
				"score":  candidate.SemanticScore,
				"weight": cs.weights.SemanticSimilarity,
				"contribution": candidate.SemanticScore *
					cs.weights.SemanticSimilarity,
			},
			"entity": map[string]float64{
				"score":        candidate.EntityScore,
				"weight":       cs.weights.EntityOverlap,
				"contribution": candidate.EntityScore * cs.weights.EntityOverlap,
			},
			"temporal": map[string]float64{
				"score":        candidate.TemporalScore,
				"weight":       cs.weights.TemporalProximity,
				"contribution": candidate.TemporalScore * cs.weights.TemporalProximity,
			},
			"type": map[string]float64{
				"score":        candidate.TypeScore,
				"weight":       cs.weights.DocumentTypeMatch,
				"contribution": candidate.TypeScore * cs.weights.DocumentTypeMatch,
			},
			"density": map[string]float64{
				"score":        candidate.DensityScore,
				"weight":       cs.weights.FactDensity,
				"contribution": candidate.DensityScore * cs.weights.FactDensity,
			},
		},
		"estimated_tokens": candidate.EstimatedTokens,
	}
}

// ValidateWeights checks that scoring weights sum to approximately 1.0
func (sw *ScoringWeights) Validate() error {
	sum := sw.SemanticSimilarity + sw.EntityOverlap + sw.TemporalProximity +
		sw.DocumentTypeMatch + sw.FactDensity

	// Allow small floating point tolerance
	tolerance := 0.01
	if math.Abs(sum-1.0) > tolerance {
		return fmt.Errorf("scoring weights must sum to 1.0 (got %.4f)", sum)
	}

	// Check each weight is between 0 and 1
	weights := []struct {
		name  string
		value float64
	}{
		{"SemanticSimilarity", sw.SemanticSimilarity},
		{"EntityOverlap", sw.EntityOverlap},
		{"TemporalProximity", sw.TemporalProximity},
		{"DocumentTypeMatch", sw.DocumentTypeMatch},
		{"FactDensity", sw.FactDensity},
	}

	for _, w := range weights {
		if w.value < 0 || w.value > 1 {
			return fmt.Errorf("weight %s must be between 0 and 1 (got %.4f)",
				w.name, w.value)
		}
	}

	return nil
}

// FormatSelectedContext formats the selected candidates for inclusion in the AI prompt
func (cs *ContextSelector) FormatSelectedContext(
	selected []ArtifactCandidate,
) string {
	if len(selected) == 0 {
		return "No related artifacts found."
	}

	output := "Related Artifacts:\n\n"

	for i, candidate := range selected {
		// Calculate relative time
		relativeTime := cs.formatRelativeTime(candidate.UploadedAt)

		output += fmt.Sprintf("[%d] \"%s\" (%s, uploaded %s)\n",
			i+1, candidate.Filename, candidate.Category, relativeTime)

		if candidate.ExecutiveSummary != "" {
			// Truncate summary if too long
			summary := candidate.ExecutiveSummary
			if len(summary) > 200 {
				summary = summary[:197] + "..."
			}
			output += fmt.Sprintf("    Summary: %s\n", summary)
		}

		if len(candidate.SharedPersonIDs) > 0 {
			peopleList := ""
			maxPeople := 3
			for i, person := range candidate.MentionedPeople {
				if i >= maxPeople {
					peopleList += fmt.Sprintf(", +%d more", len(candidate.MentionedPeople)-maxPeople)
					break
				}
				if i > 0 {
					peopleList += ", "
				}
				peopleList += person
			}
			output += fmt.Sprintf("    Shared People: %s\n", peopleList)
		}

		if len(candidate.Topics) > 0 {
			topicList := ""
			maxTopics := 4
			for i, topic := range candidate.Topics {
				if i >= maxTopics {
					topicList += fmt.Sprintf(", +%d more", len(candidate.Topics)-maxTopics)
					break
				}
				if i > 0 {
					topicList += ", "
				}
				topicList += topic
			}
			output += fmt.Sprintf("    Topics: %s\n", topicList)
		}

		output += fmt.Sprintf("    Relevance Score: %.2f\n\n", candidate.TotalScore)
	}

	return output
}

// formatRelativeTime formats a timestamp as relative time (e.g., "2 weeks ago")
func (cs *ContextSelector) formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	days := int(diff.Hours() / 24)
	weeks := days / 7
	months := days / 30

	if days == 0 {
		return "today"
	} else if days == 1 {
		return "yesterday"
	} else if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	} else if weeks == 1 {
		return "1 week ago"
	} else if weeks < 4 {
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if months == 1 {
		return "1 month ago"
	} else if months < 12 {
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := months / 12
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
