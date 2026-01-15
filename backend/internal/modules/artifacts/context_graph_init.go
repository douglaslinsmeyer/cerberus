package artifacts

import (
	"log"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/redis/go-redis/v9"
)

// InitializeContextGraphServices creates and wires up all context graph services
// This is the main initialization function for enabling enriched context
func InitializeContextGraphServices(
	repo RepositoryInterface,
	redisClient *redis.Client,
) (*ContextGraphBuilder, error) {
	log.Println("Initializing context graph services for enriched AI analysis...")

	// 1. Create ContextSelector with default scoring weights
	contextSelector := NewDefaultContextSelector()
	log.Println("✓ Context selector initialized with default weights")

	// 2. Create EntityGraphQuery
	entityGraph := NewEntityGraphQuery(repo)
	log.Println("✓ Entity graph query service initialized")

	// 3. Create TemporalOrganizer
	temporal := NewTemporalOrganizer(repo)
	log.Println("✓ Temporal organizer initialized")

	// 4. Create FactAggregator
	factAggregator := NewFactAggregator(repo)
	log.Println("✓ Fact aggregator initialized")

	// 5. Create ContextCache with 24-hour TTL
	var cache *ContextCache
	if redisClient != nil {
		cache = NewContextCache(redisClient, repo, 24*time.Hour)
		log.Println("✓ Context cache initialized with Redis")
	} else {
		log.Println("⚠ Warning: Redis not available, caching disabled")
	}

	// 6. Create ContextGraphBuilder (the main orchestrator)
	contextGraphBuilder := NewContextGraphBuilder(
		repo,
		contextSelector,
		entityGraph,
		temporal,
		factAggregator,
		cache,
	)
	log.Println("✓ Context graph builder initialized")

	log.Println("✅ Context graph services ready for enriched AI analysis")

	return contextGraphBuilder, nil
}

// InitializeAIAnalyzerWithContext creates an AI analyzer with enriched context support
// This is the recommended way to create an AI analyzer for production use
func InitializeAIAnalyzerWithContext(
	aiClient *ai.Client,
	repo RepositoryInterface,
	redisClient *redis.Client,
) (*AIAnalyzer, error) {
	// Initialize context graph services
	contextBuilder, err := InitializeContextGraphServices(repo, redisClient)
	if err != nil {
		log.Printf("Warning: Failed to initialize context graph: %v", err)
		// Fall back to basic analyzer
		return NewAIAnalyzer(aiClient, repo), nil
	}

	// Create analyzer with context support
	analyzer := NewAIAnalyzerWithContext(aiClient, repo, contextBuilder)
	log.Println("✅ AI Analyzer initialized with enriched context support")

	return analyzer, nil
}

// ContextGraphConfig allows customization of context graph behavior
type ContextGraphConfig struct {
	// EnableEnrichedContext toggles enriched context on/off
	EnableEnrichedContext bool

	// TokenBudget is the maximum tokens to allocate for context
	TokenBudget int

	// ScoringWeights allows customization of the scoring algorithm
	ScoringWeights *ScoringWeights

	// CacheTTL is the cache time-to-live duration
	CacheTTL time.Duration

	// EnableCaching toggles Redis caching
	EnableCaching bool
}

// DefaultContextGraphConfig returns the default configuration
func DefaultContextGraphConfig() *ContextGraphConfig {
	return &ContextGraphConfig{
		EnableEnrichedContext: true,
		TokenBudget:          4000,
		ScoringWeights:       nil, // Will use defaults
		CacheTTL:             24 * time.Hour,
		EnableCaching:        true,
	}
}

// InitializeWithConfig creates context graph services with custom configuration
func InitializeWithConfig(
	repo RepositoryInterface,
	redisClient *redis.Client,
	config *ContextGraphConfig,
) (*ContextGraphBuilder, error) {
	if !config.EnableEnrichedContext {
		log.Println("Enriched context disabled by configuration")
		return nil, nil
	}

	// Create ContextSelector with custom or default weights
	var contextSelector *ContextSelector
	if config.ScoringWeights != nil {
		// Validate custom weights
		if err := config.ScoringWeights.Validate(); err != nil {
			log.Printf("Warning: Invalid scoring weights: %v, using defaults", err)
			contextSelector = NewDefaultContextSelector()
		} else {
			contextSelector = NewContextSelector(config.TokenBudget, *config.ScoringWeights)
			log.Println("✓ Context selector initialized with custom weights")
		}
	} else {
		contextSelector = NewDefaultContextSelector()
		log.Println("✓ Context selector initialized with default weights")
	}

	// Create other services
	entityGraph := NewEntityGraphQuery(repo)
	temporal := NewTemporalOrganizer(repo)
	factAggregator := NewFactAggregator(repo)

	// Create cache if enabled
	var cache *ContextCache
	if config.EnableCaching && redisClient != nil {
		cache = NewContextCache(redisClient, repo, config.CacheTTL)
		log.Printf("✓ Context cache initialized (TTL: %s)", config.CacheTTL)
	}

	// Create builder
	builder := NewContextGraphBuilder(
		repo,
		contextSelector,
		entityGraph,
		temporal,
		factAggregator,
		cache,
	)

	log.Println("✅ Context graph services initialized with custom config")
	return builder, nil
}

// Example usage in worker or main:
//
// func main() {
//     // Initialize database and Redis
//     database := db.Connect(...)
//     redisClient := redis.NewClient(...)
//     aiClient := ai.NewClient(...)
//
//     // Initialize repository
//     repo := artifacts.NewRepository(database)
//
//     // Option 1: Simple initialization with defaults
//     analyzer, err := artifacts.InitializeAIAnalyzerWithContext(aiClient, repo, redisClient)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     // Option 2: Custom configuration
//     config := &artifacts.ContextGraphConfig{
//         EnableEnrichedContext: true,
//         TokenBudget:          5000,  // Larger budget
//         ScoringWeights: &artifacts.ScoringWeights{
//             SemanticSimilarity: 0.50,  // Prioritize semantic similarity
//             EntityOverlap:      0.20,
//             TemporalProximity:  0.15,
//             DocumentTypeMatch:  0.10,
//             FactDensity:        0.05,
//         },
//         CacheTTL:      48 * time.Hour,  // Longer cache
//         EnableCaching: true,
//     }
//
//     contextBuilder, err := artifacts.InitializeWithConfig(repo, redisClient, config)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     analyzer := artifacts.NewAIAnalyzerWithContext(aiClient, repo, contextBuilder)
//
//     // Use analyzer to process artifacts
//     result, err := analyzer.AnalyzeArtifact(ctx, artifact, programContext)
// }

// MigrateToEnrichedContext is a helper for gradually migrating to enriched context
// It allows you to enable enriched context for specific programs or as a percentage
func MigrateToEnrichedContext(
	analyzer *AIAnalyzer,
	contextBuilder *ContextGraphBuilder,
	enableForProgramIDs []string,
	rolloutPercentage int,
) {
	// This is a placeholder for a gradual rollout strategy
	// In production, you might:
	// 1. Enable for specific programs first
	// 2. Gradually increase percentage of artifacts analyzed with context
	// 3. Monitor quality improvements and costs
	// 4. Full rollout once validated

	if contextBuilder != nil {
		analyzer.SetContextGraphBuilder(contextBuilder)
		analyzer.EnableEnrichedContext(true)
		log.Printf("✅ Enriched context enabled (rollout: %d%%)", rolloutPercentage)
	}
}

// BackgroundJobs contains maintenance jobs for context graph
type BackgroundJobs struct {
	contextCache *ContextCache
	entityGraph  *EntityGraphQuery
	temporal     *TemporalOrganizer
}

// NewBackgroundJobs creates background job manager
func NewBackgroundJobs(cache *ContextCache, entityGraph *EntityGraphQuery, temporal *TemporalOrganizer) *BackgroundJobs {
	return &BackgroundJobs{
		contextCache: cache,
		entityGraph:  entityGraph,
		temporal:     temporal,
	}
}

// RunCacheCleanup should run periodically (e.g., every hour)
func (bj *BackgroundJobs) RunCacheCleanup() {
	if bj.contextCache == nil {
		return
	}

	// Clean up expired cache entries
	// This would typically be called from a cron job or background worker
	log.Println("Running cache cleanup...")
	// Implementation would call contextCache.CleanupExpiredCache()
}

// RunEntityGraphRebuild should run daily to recompute entity relationships
func (bj *BackgroundJobs) RunEntityGraphRebuild() {
	if bj.entityGraph == nil {
		return
	}

	log.Println("Rebuilding entity graph...")
	// This would recompute co-occurrences and relationship strengths
	// Implementation would iterate through programs and rebuild their graphs
}

// RunSequenceDetection should run daily to detect temporal sequences
func (bj *BackgroundJobs) RunSequenceDetection() {
	if bj.temporal == nil {
		return
	}

	log.Println("Detecting temporal sequences...")
	// This would detect patterns like weekly meetings, monthly reports, etc.
	// Implementation would call temporal.DetectSequences() for each program
}
