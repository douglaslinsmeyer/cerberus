package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ContextCache provides multi-level caching for enriched context
// Level 1: In-memory cache (optional, fast but limited)
// Level 2: Redis cache (persistent, 24-hour TTL)
// Level 3: Database cache table (long-term, 7-day TTL)
type ContextCache struct {
	redis  *redis.Client
	repo   RepositoryInterface
	ttl    time.Duration
	prefix string
}

// NewContextCache creates a new context cache
func NewContextCache(
	redisClient *redis.Client,
	repo RepositoryInterface,
	ttl time.Duration,
) *ContextCache {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24-hour TTL
	}

	return &ContextCache{
		redis:  redisClient,
		repo:   repo,
		ttl:    ttl,
		prefix: "artifact:context:",
	}
}

// GetCachedContext retrieves cached context for an artifact
// Checks Redis first, then database cache table
func (cc *ContextCache) GetCachedContext(
	ctx context.Context,
	artifactID uuid.UUID,
) (*EnrichedContext, error) {
	// Try Redis first (Level 2 cache)
	if cc.redis != nil {
		enrichedCtx, err := cc.getFromRedis(ctx, artifactID)
		if err == nil && enrichedCtx != nil {
			log.Printf("Redis cache hit for artifact %s", artifactID)
			return enrichedCtx, nil
		}
		// Cache miss or error, continue to database
	}

	// Try database cache table (Level 3 cache)
	if cc.repo != nil {
		enrichedCtx, err := cc.getFromDatabase(ctx, artifactID)
		if err == nil && enrichedCtx != nil {
			log.Printf("Database cache hit for artifact %s", artifactID)

			// Populate Redis cache for next time
			if cc.redis != nil {
				go cc.setInRedis(context.Background(), artifactID, enrichedCtx)
			}

			return enrichedCtx, nil
		}
	}

	// Cache miss at all levels
	return nil, fmt.Errorf("cache miss for artifact %s", artifactID)
}

// CacheContext stores enriched context in cache
// Writes to both Redis and database cache table
func (cc *ContextCache) CacheContext(
	ctx context.Context,
	artifactID uuid.UUID,
	enrichedCtx *EnrichedContext,
) error {
	// Write to Redis (Level 2)
	if cc.redis != nil {
		if err := cc.setInRedis(ctx, artifactID, enrichedCtx); err != nil {
			log.Printf("Warning: failed to write to Redis cache: %v", err)
			// Continue to database cache
		}
	}

	// Write to database cache table (Level 3)
	if cc.repo != nil {
		if err := cc.setInDatabase(ctx, artifactID, enrichedCtx); err != nil {
			log.Printf("Warning: failed to write to database cache: %v", err)
		}
	}

	return nil
}

// InvalidateArtifactCache invalidates cache for a specific artifact
func (cc *ContextCache) InvalidateArtifactCache(
	ctx context.Context,
	artifactID uuid.UUID,
) error {
	// Delete from Redis
	if cc.redis != nil {
		key := cc.redisKey(artifactID)
		if err := cc.redis.Del(ctx, key).Err(); err != nil {
			log.Printf("Warning: failed to delete from Redis: %v", err)
		}
	}

	// Delete from database
	if cc.repo != nil {
		if err := cc.repo.DeleteContextCache(ctx, artifactID); err != nil {
			log.Printf("Warning: failed to delete from database cache: %v", err)
		}
	}

	return nil
}

// InvalidateProgramCache invalidates cache for all artifacts in a program
// This is called when new artifacts are uploaded or program settings change
func (cc *ContextCache) InvalidateProgramCache(
	ctx context.Context,
	programID uuid.UUID,
) error {
	log.Printf("Invalidating cache for program %s", programID)

	// Get all artifact IDs in this program
	artifactIDs, err := cc.repo.GetArtifactIDsByProgram(ctx, programID)
	if err != nil {
		return fmt.Errorf("failed to get artifact IDs: %w", err)
	}

	// Invalidate each artifact (could be optimized with batch operations)
	for _, artifactID := range artifactIDs {
		if err := cc.InvalidateArtifactCache(ctx, artifactID); err != nil {
			log.Printf("Warning: failed to invalidate cache for artifact %s: %v",
				artifactID, err)
		}
	}

	return nil
}

// CleanupExpiredCache removes expired cache entries
// Should be called periodically (e.g., every hour)
func (cc *ContextCache) CleanupExpiredCache(ctx context.Context) (int, error) {
	if cc.repo == nil {
		return 0, nil
	}

	deletedCount, err := cc.repo.CleanupExpiredContextCache(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired cache: %w", err)
	}

	log.Printf("Cleaned up %d expired cache entries", deletedCount)
	return deletedCount, nil
}

// GetCacheStats returns statistics about cache usage
func (cc *ContextCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	if cc.repo != nil {
		// Database cache stats
		dbStats, err := cc.repo.GetContextCacheStats(ctx)
		if err == nil {
			stats["database"] = dbStats
		}
	}

	if cc.redis != nil {
		// Redis cache stats (approximate)
		keys, err := cc.redis.Keys(ctx, cc.prefix+"*").Result()
		if err == nil {
			stats["redis_entries"] = len(keys)
		}
	}

	return stats, nil
}

// Redis operations

func (cc *ContextCache) redisKey(artifactID uuid.UUID) string {
	return cc.prefix + artifactID.String()
}

func (cc *ContextCache) getFromRedis(
	ctx context.Context,
	artifactID uuid.UUID,
) (*EnrichedContext, error) {
	key := cc.redisKey(artifactID)

	// Get from Redis
	data, err := cc.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss")
	} else if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	// Deserialize
	var enrichedCtx EnrichedContext
	if err := json.Unmarshal(data, &enrichedCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &enrichedCtx, nil
}

func (cc *ContextCache) setInRedis(
	ctx context.Context,
	artifactID uuid.UUID,
	enrichedCtx *EnrichedContext,
) error {
	key := cc.redisKey(artifactID)

	// Serialize
	data, err := json.Marshal(enrichedCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Set with TTL
	if err := cc.redis.Set(ctx, key, data, cc.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Database operations

func (cc *ContextCache) getFromDatabase(
	ctx context.Context,
	artifactID uuid.UUID,
) (*EnrichedContext, error) {
	// Query artifact_context_cache table
	cacheEntry, err := cc.repo.GetContextCacheEntry(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	// Check if expired
	if cacheEntry.ExpiresAt.Before(time.Now()) {
		// Expired, delete and return miss
		cc.repo.DeleteContextCache(ctx, artifactID)
		return nil, fmt.Errorf("cache entry expired")
	}

	// Deserialize context_data JSONB
	var enrichedCtx EnrichedContext
	if err := json.Unmarshal(cacheEntry.ContextData, &enrichedCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &enrichedCtx, nil
}

func (cc *ContextCache) setInDatabase(
	ctx context.Context,
	artifactID uuid.UUID,
	enrichedCtx *EnrichedContext,
) error {
	// Serialize context
	contextData, err := json.Marshal(enrichedCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Extract artifact IDs for artifacts_included array
	artifactIDs := []uuid.UUID{enrichedCtx.TargetArtifactID}
	for _, related := range enrichedCtx.RelatedArtifacts {
		artifactIDs = append(artifactIDs, related.ArtifactID)
	}

	// Calculate expiration (7 days for database cache)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Insert or update cache entry
	cacheEntry := &ContextCacheEntry{
		CacheID:           uuid.New(),
		ArtifactID:        artifactID,
		ContextData:       contextData,
		TokenCount:        enrichedCtx.EstimatedTokens,
		ArtifactsIncluded: artifactIDs,
		CacheVersion:      1,
		CreatedAt:         time.Now(),
		ExpiresAt:         expiresAt,
	}

	if err := cc.repo.UpsertContextCacheEntry(ctx, cacheEntry); err != nil {
		return fmt.Errorf("database insert error: %w", err)
	}

	return nil
}

// ContextCacheEntry represents a row in artifact_context_cache table
type ContextCacheEntry struct {
	CacheID           uuid.UUID
	ArtifactID        uuid.UUID
	ProgramID         uuid.UUID
	ContextData       []byte // JSONB serialized EnrichedContext
	TokenCount        int
	ArtifactsIncluded []uuid.UUID
	CacheVersion      int
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

// WarmCache pre-computes and caches context for pending artifacts
// This can be called in the background to improve user experience
func (cc *ContextCache) WarmCache(
	ctx context.Context,
	builder *ContextGraphBuilder,
	programID uuid.UUID,
	limit int,
) error {
	// Get recently completed artifacts that don't have cache entries
	artifacts, err := cc.repo.GetRecentArtifactsWithoutCache(ctx, programID, limit)
	if err != nil {
		return fmt.Errorf("failed to get artifacts: %w", err)
	}

	log.Printf("Warming cache for %d artifacts in program %s", len(artifacts), programID)

	// Build context for each (in background)
	for _, artifact := range artifacts {
		go func(art Artifact) {
			_, err := builder.BuildEnrichedContext(context.Background(), &art, 4000)
			if err != nil {
				log.Printf("Warning: failed to build context for warm cache: %v", err)
				return
			}

			// Cache will be written by BuildEnrichedContext internally
			log.Printf("Warmed cache for artifact %s", art.ArtifactID)
		}(artifact)
	}

	return nil
}

// RefreshMaterializedView refreshes the artifact_context_summary materialized view
// This should be called after artifact processing completes
func (cc *ContextCache) RefreshMaterializedView(ctx context.Context) error {
	if cc.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	if err := cc.repo.RefreshContextSummaryView(ctx); err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}

	log.Println("Refreshed artifact_context_summary materialized view")
	return nil
}

// ShouldInvalidateOnEvent determines if cache should be invalidated for an event
func (cc *ContextCache) ShouldInvalidateOnEvent(eventType string) bool {
	invalidatingEvents := map[string]bool{
		"artifact.uploaded":           true,
		"artifact.reprocessed":        true,
		"artifact.deleted":            true,
		"stakeholder.merged":          true,
		"program.config.updated":      true,
		"program.taxonomy.updated":    true,
		"risk.created":                true,
		"risk.updated":                true,
	}

	return invalidatingEvents[eventType]
}
