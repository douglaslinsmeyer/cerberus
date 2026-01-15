package artifacts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// EnrichedContext represents the full context graph for an artifact
// This is what gets passed to the AI analysis prompt
type EnrichedContext struct {
	// Target artifact being analyzed
	TargetArtifactID uuid.UUID

	// Selected related artifacts
	RelatedArtifacts []ArtifactCandidate

	// Entity relationship information
	EntityGraph *EntityGraphContext

	// Temporal context (timeline)
	Timeline *TimelineContext

	// Aggregated facts from related artifacts
	AggregatedFacts *AggregatedFactsContext

	// Token budget tracking
	EstimatedTokens     int
	TokenBudget         int
	ComponentBreakdown  map[string]int
	WasContextTruncated bool
}

// EntityGraphContext contains entity relationship information
type EntityGraphContext struct {
	KeyPeople          []PersonContext
	EstimatedTokens    int
	TotalRelationships int
}

// PersonContext represents a person's context across artifacts
type PersonContext struct {
	PersonID          uuid.UUID
	Name              string
	Role              string
	Organization      string
	Classification    string // INTERNAL, EXTERNAL
	MentionCount      int
	ArtifactCount     int
	CoOccursWith      []string // Names of people they frequently appear with
	RecentContext     string   // Summary of recent mentions
}

// TimelineContext contains temporal sequencing information
type TimelineContext struct {
	PrecedingArtifacts []TimelineEntry
	FollowingArtifacts []TimelineEntry
	EstimatedTokens    int
}

// TimelineEntry represents an artifact in the timeline
type TimelineEntry struct {
	ArtifactID       uuid.UUID
	Filename         string
	Category         string
	Summary          string
	UploadedAt       time.Time
	RelativeTime     string // "2 weeks ago", "same day", etc.
	RelationToTarget string // "before", "after", "same_day"
}

// AggregatedFactsContext contains cross-artifact fact aggregations
type AggregatedFactsContext struct {
	FinancialFacts []AggregatedFact
	DateFacts      []AggregatedFact
	MetricFacts    []AggregatedFact
	Conflicts      []FactConflict
	EstimatedTokens int
}

// AggregatedFact represents a fact with its sources
type AggregatedFact struct {
	FactKey         string
	FactValue       string
	FactType        string
	SourceArtifacts []string // Filenames
	OccurrenceCount int
	ConfidenceScore float64
}

// FactConflict represents conflicting facts across artifacts
type FactConflict struct {
	FactKey          string
	ConflictingValues []ConflictingValue
	Severity         string // "minor", "major"
}

// ConflictingValue represents one value in a conflict
type ConflictingValue struct {
	Value            string
	SourceArtifactID uuid.UUID
	SourceFilename   string
	ConfidenceScore  float64
}

// ContextGraphBuilder orchestrates the building of enriched context
// It coordinates calls to all component services
type ContextGraphBuilder struct {
	repo            RepositoryInterface
	contextSelector *ContextSelector
	entityGraph     *EntityGraphQuery
	temporal        *TemporalOrganizer
	factAggregator  *FactAggregator
	cache           *ContextCache
}

// NewContextGraphBuilder creates a new context graph builder
func NewContextGraphBuilder(
	repo RepositoryInterface,
	contextSelector *ContextSelector,
	entityGraph *EntityGraphQuery,
	temporal *TemporalOrganizer,
	factAggregator *FactAggregator,
	cache *ContextCache,
) *ContextGraphBuilder {
	return &ContextGraphBuilder{
		repo:            repo,
		contextSelector: contextSelector,
		entityGraph:     entityGraph,
		temporal:        temporal,
		factAggregator:  factAggregator,
		cache:           cache,
	}
}

// BuildEnrichedContext builds the full enriched context for an artifact
// This is the main entry point for context graph construction
func (cgb *ContextGraphBuilder) BuildEnrichedContext(
	ctx context.Context,
	artifact *Artifact,
	tokenBudget int,
) (*EnrichedContext, error) {
	// Check cache first
	if cgb.cache != nil {
		cachedContext, err := cgb.cache.GetCachedContext(ctx, artifact.ArtifactID)
		if err == nil && cachedContext != nil {
			log.Printf("Context cache hit for artifact %s", artifact.ArtifactID)
			return cachedContext, nil
		}
	}

	// Initialize enriched context
	enrichedCtx := &EnrichedContext{
		TargetArtifactID:   artifact.ArtifactID,
		TokenBudget:        tokenBudget,
		ComponentBreakdown: make(map[string]int),
	}

	// Allocate token budget across components
	// Priority: Related artifacts > Entity graph > Timeline > Facts
	budgetAllocation := cgb.allocateTokenBudget(tokenBudget)

	// Step 1: Find and score related artifacts
	relatedArtifacts, err := cgb.findRelatedArtifacts(ctx, artifact, budgetAllocation["related_artifacts"])
	if err != nil {
		log.Printf("Warning: failed to find related artifacts: %v", err)
		// Continue with empty related artifacts
	} else {
		enrichedCtx.RelatedArtifacts = relatedArtifacts
		tokens := cgb.calculateRelatedArtifactsTokens(relatedArtifacts)
		enrichedCtx.ComponentBreakdown["related_artifacts"] = tokens
		enrichedCtx.EstimatedTokens += tokens
	}

	// Step 2: Build entity relationship graph
	entityCtx, err := cgb.buildEntityContext(ctx, artifact, budgetAllocation["entity_graph"])
	if err != nil {
		log.Printf("Warning: failed to build entity context: %v", err)
	} else {
		enrichedCtx.EntityGraph = entityCtx
		enrichedCtx.ComponentBreakdown["entity_graph"] = entityCtx.EstimatedTokens
		enrichedCtx.EstimatedTokens += entityCtx.EstimatedTokens
	}

	// Step 3: Build temporal timeline
	timelineCtx, err := cgb.buildTimelineContext(ctx, artifact, budgetAllocation["timeline"])
	if err != nil {
		log.Printf("Warning: failed to build timeline context: %v", err)
	} else {
		enrichedCtx.Timeline = timelineCtx
		enrichedCtx.ComponentBreakdown["timeline"] = timelineCtx.EstimatedTokens
		enrichedCtx.EstimatedTokens += timelineCtx.EstimatedTokens
	}

	// Step 4: Aggregate facts from related artifacts
	if len(enrichedCtx.RelatedArtifacts) > 0 {
		relatedIDs := make([]uuid.UUID, len(enrichedCtx.RelatedArtifacts))
		for i, ra := range enrichedCtx.RelatedArtifacts {
			relatedIDs[i] = ra.ArtifactID
		}

		factsCtx, err := cgb.aggregateFacts(ctx, artifact.ArtifactID, relatedIDs, budgetAllocation["facts"])
		if err != nil {
			log.Printf("Warning: failed to aggregate facts: %v", err)
		} else {
			enrichedCtx.AggregatedFacts = factsCtx
			enrichedCtx.ComponentBreakdown["facts"] = factsCtx.EstimatedTokens
			enrichedCtx.EstimatedTokens += factsCtx.EstimatedTokens
		}
	}

	// Check if we exceeded budget (shouldn't happen, but safety check)
	if enrichedCtx.EstimatedTokens > tokenBudget {
		enrichedCtx.WasContextTruncated = true
		log.Printf("Warning: context exceeded budget (%d > %d), truncation occurred",
			enrichedCtx.EstimatedTokens, tokenBudget)
	}

	// Cache the result
	if cgb.cache != nil {
		if err := cgb.cache.CacheContext(ctx, artifact.ArtifactID, enrichedCtx); err != nil {
			log.Printf("Warning: failed to cache context: %v", err)
		}
	}

	return enrichedCtx, nil
}

// allocateTokenBudget divides the token budget across components
func (cgb *ContextGraphBuilder) allocateTokenBudget(totalBudget int) map[string]int {
	// Allocation strategy (based on importance):
	// Related artifacts: 50% (1200 tokens for 3-4 artifacts)
	// Entity graph: 25% (800 tokens for ~10-15 people)
	// Timeline: 15% (400 tokens for temporal context)
	// Facts: 10% (600 tokens for aggregated facts)

	return map[string]int{
		"related_artifacts": int(float64(totalBudget) * 0.50),
		"entity_graph":      int(float64(totalBudget) * 0.25),
		"timeline":          int(float64(totalBudget) * 0.15),
		"facts":             int(float64(totalBudget) * 0.10),
	}
}

// findRelatedArtifacts finds and scores artifacts related to the target
func (cgb *ContextGraphBuilder) findRelatedArtifacts(
	ctx context.Context,
	artifact *Artifact,
	tokenBudget int,
) ([]ArtifactCandidate, error) {
	// Get person IDs for target artifact (for entity overlap scoring)
	targetPersonIDs, err := cgb.repo.GetPersonIDsByArtifact(ctx, artifact.ArtifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get person IDs: %w", err)
	}

	// Find candidates using multiple strategies
	candidates := []ArtifactCandidate{}

	// Strategy 1: Semantic similarity (vector search)
	if artifact.ProcessingStatus == "completed" {
		semanticCandidates, err := cgb.findSemanticallyRelatedArtifacts(ctx, artifact, 10)
		if err != nil {
			log.Printf("Warning: semantic search failed: %v", err)
		} else {
			candidates = append(candidates, semanticCandidates...)
		}
	}

	// Strategy 2: Shared entities
	if len(targetPersonIDs) > 0 {
		entityCandidates, err := cgb.findArtifactsWithSharedEntities(ctx, artifact, targetPersonIDs, 10)
		if err != nil {
			log.Printf("Warning: entity search failed: %v", err)
		} else {
			// Merge with existing candidates (avoid duplicates)
			candidates = cgb.mergeCandidates(candidates, entityCandidates)
		}
	}

	// Strategy 3: Temporal proximity
	temporalCandidates, err := cgb.findTemporallyRelatedArtifacts(ctx, artifact, 10)
	if err != nil {
		log.Printf("Warning: temporal search failed: %v", err)
	} else {
		candidates = cgb.mergeCandidates(candidates, temporalCandidates)
	}

	// Score and select the best candidates within budget
	if len(candidates) == 0 {
		return []ArtifactCandidate{}, nil
	}

	selected := cgb.contextSelector.ScoreAndSelectContext(
		candidates,
		artifact,
		targetPersonIDs,
		tokenBudget,
	)

	return selected, nil
}

// findSemanticallyRelatedArtifacts uses vector similarity search
func (cgb *ContextGraphBuilder) findSemanticallyRelatedArtifacts(
	ctx context.Context,
	artifact *Artifact,
	limit int,
) ([]ArtifactCandidate, error) {
	// This will be implemented in the repository
	// For now, return empty (to be filled in repository.go)
	return cgb.repo.FindSemanticallyRelatedArtifacts(ctx, artifact.ArtifactID, artifact.ProgramID, limit)
}

// findArtifactsWithSharedEntities finds artifacts with overlapping people
func (cgb *ContextGraphBuilder) findArtifactsWithSharedEntities(
	ctx context.Context,
	artifact *Artifact,
	personIDs []uuid.UUID,
	limit int,
) ([]ArtifactCandidate, error) {
	return cgb.repo.FindArtifactsWithSharedEntities(ctx, artifact.ArtifactID, artifact.ProgramID, personIDs, limit)
}

// findTemporallyRelatedArtifacts finds artifacts close in time
func (cgb *ContextGraphBuilder) findTemporallyRelatedArtifacts(
	ctx context.Context,
	artifact *Artifact,
	limit int,
) ([]ArtifactCandidate, error) {
	return cgb.repo.FindTemporallyRelatedArtifacts(ctx, artifact.ArtifactID, artifact.ProgramID, artifact.UploadedAt, limit)
}

// mergeCandidates merges two candidate lists, avoiding duplicates
func (cgb *ContextGraphBuilder) mergeCandidates(
	existing []ArtifactCandidate,
	newCandidates []ArtifactCandidate,
) []ArtifactCandidate {
	// Create map of existing IDs
	existingIDs := make(map[uuid.UUID]bool)
	for _, candidate := range existing {
		existingIDs[candidate.ArtifactID] = true
	}

	// Add new candidates if not duplicate
	for _, candidate := range newCandidates {
		if !existingIDs[candidate.ArtifactID] {
			existing = append(existing, candidate)
		}
	}

	return existing
}

// buildEntityContext builds the entity relationship context
func (cgb *ContextGraphBuilder) buildEntityContext(
	ctx context.Context,
	artifact *Artifact,
	tokenBudget int,
) (*EntityGraphContext, error) {
	if cgb.entityGraph == nil {
		return &EntityGraphContext{}, nil
	}

	// Get key people from the target artifact
	persons, err := cgb.repo.GetPersonsByArtifact(ctx, artifact.ArtifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get persons: %w", err)
	}

	// Build context for each person (limit to fit budget)
	keyPeople := []PersonContext{}
	tokensUsed := 0
	tokensPerPerson := 50 // Estimate

	for _, person := range persons {
		if tokensUsed+tokensPerPerson > tokenBudget {
			break
		}

		personCtx := PersonContext{
			PersonID:     person.PersonID,
			Name:         person.PersonName,
			MentionCount: person.MentionCount,
		}

		if person.PersonRole.Valid {
			personCtx.Role = person.PersonRole.String
		}
		if person.PersonOrganization.Valid {
			personCtx.Organization = person.PersonOrganization.String
		}

		// Get co-occurrences (people this person appears with)
		coOccurrences, err := cgb.entityGraph.GetPersonCoOccurrences(ctx, person.PersonID, artifact.ProgramID)
		if err == nil && len(coOccurrences) > 0 {
			// Limit to top 3
			for i := 0; i < len(coOccurrences) && i < 3; i++ {
				// Determine which person name to use (the other person in the relationship)
				coOccurName := coOccurrences[i].Person2Name
				if coOccurrences[i].Person2ID == person.PersonID {
					coOccurName = coOccurrences[i].Person1Name
				}
				personCtx.CoOccursWith = append(personCtx.CoOccursWith, coOccurName)
			}
		}

		keyPeople = append(keyPeople, personCtx)
		tokensUsed += tokensPerPerson
	}

	return &EntityGraphContext{
		KeyPeople:       keyPeople,
		EstimatedTokens: tokensUsed,
	}, nil
}

// buildTimelineContext builds the temporal timeline context
func (cgb *ContextGraphBuilder) buildTimelineContext(
	ctx context.Context,
	artifact *Artifact,
	tokenBudget int,
) (*TimelineContext, error) {
	if cgb.temporal == nil {
		return &TimelineContext{}, nil
	}

	timeline, err := cgb.temporal.BuildTimeline(ctx, artifact.ProgramID, artifact)
	if err != nil {
		return nil, fmt.Errorf("failed to build timeline: %w", err)
	}

	return timeline, nil
}

// aggregateFacts aggregates facts from related artifacts
func (cgb *ContextGraphBuilder) aggregateFacts(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	relatedArtifactIDs []uuid.UUID,
	tokenBudget int,
) (*AggregatedFactsContext, error) {
	if cgb.factAggregator == nil {
		return &AggregatedFactsContext{}, nil
	}

	factsCtx, err := cgb.factAggregator.AggregateRelatedFacts(ctx, targetArtifactID, relatedArtifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate facts: %w", err)
	}

	return factsCtx, nil
}

// calculateRelatedArtifactsTokens estimates tokens for related artifacts
func (cgb *ContextGraphBuilder) calculateRelatedArtifactsTokens(artifacts []ArtifactCandidate) int {
	total := 0
	for _, artifact := range artifacts {
		total += artifact.EstimatedTokens
	}
	return total
}
