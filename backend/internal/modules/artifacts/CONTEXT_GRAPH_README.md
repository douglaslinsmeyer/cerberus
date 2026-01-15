# Context Graph: Full Intelligence Artifact Analysis

## Overview

The Context Graph system provides **enriched cross-artifact intelligence** for AI analysis. Instead of analyzing each artifact in isolation, the system builds a comprehensive knowledge graph that includes:

- **Related artifacts** with semantic similarity scoring
- **Entity relationships** tracking who works with whom across documents
- **Document timelines** showing temporal sequences and narrative flow
- **Aggregated facts** with conflict detection across artifacts
- **Multi-level caching** for performance optimization

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  AI Analysis Pipeline                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Artifact Upload → ContextGraphBuilder                   │
│       ↓                                                      │
│  2. Build Enriched Context (4 components):                  │
│     • Semantic similarity search (vector embeddings)        │
│     • Entity overlap detection (shared people)              │
│     • Temporal proximity (timeline)                         │
│     • Fact aggregation (cross-doc facts)                    │
│       ↓                                                      │
│  3. Score & Select Top Related Artifacts                    │
│     Weighted formula:                                        │
│       40% Semantic + 25% Entity + 20% Temporal +           │
│       10% Type + 5% Density                                 │
│       ↓                                                      │
│  4. Format Context for AI Prompt                            │
│       ↓                                                      │
│  5. Claude Sonnet 4 Analysis                                │
│     (with enriched context)                                  │
│       ↓                                                      │
│  6. Enhanced Results with Cross-References                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Database Schema

### New Tables

**1. `artifact_context_cache`**
```sql
-- Caches pre-computed enriched context
-- TTL: 24 hours (Redis), 7 days (Database)
CREATE TABLE artifact_context_cache (
    cache_id UUID PRIMARY KEY,
    artifact_id UUID UNIQUE NOT NULL,
    program_id UUID NOT NULL,
    context_data JSONB NOT NULL,
    token_count INTEGER NOT NULL,
    artifacts_included UUID[] NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
```

**2. `artifact_entity_graph`**
```sql
-- Tracks person co-occurrences across artifacts
CREATE TABLE artifact_entity_graph (
    edge_id UUID PRIMARY KEY,
    program_id UUID NOT NULL,
    person_id_1 UUID NOT NULL,
    person_id_2 UUID NOT NULL,
    co_occurrence_count INTEGER DEFAULT 1,
    shared_artifact_ids UUID[] NOT NULL,
    relationship_strength DECIMAL(5,4),
    UNIQUE(person_id_1, person_id_2)
);
```

**3. `artifact_temporal_sequences`**
```sql
-- Groups artifacts into temporal sequences
CREATE TABLE artifact_temporal_sequences (
    sequence_id UUID PRIMARY KEY,
    program_id UUID NOT NULL,
    sequence_name VARCHAR(255) NOT NULL,
    sequence_type VARCHAR(100),  -- meeting_series, project_phase, etc.
    artifact_ids UUID[] NOT NULL,
    start_date DATE,
    end_date DATE,
    confidence_score DECIMAL(5,4)
);
```

**4. Materialized View: `artifact_context_summary`**
```sql
-- Pre-aggregated metadata for fast context building
CREATE MATERIALIZED VIEW artifact_context_summary AS
SELECT
    a.artifact_id,
    a.program_id,
    a.filename,
    s.executive_summary,
    array_agg(DISTINCT ap.person_name) as mentioned_people,
    array_agg(DISTINCT at.topic_name) as topics,
    (LENGTH(s.executive_summary) / 4 + 100) as estimated_tokens
FROM artifacts a
LEFT JOIN artifact_summaries s ON a.artifact_id = s.artifact_id
LEFT JOIN artifact_persons ap ON a.artifact_id = ap.artifact_id
LEFT JOIN artifact_topics at ON a.artifact_id = at.artifact_id
WHERE a.deleted_at IS NULL AND a.processing_status = 'completed'
GROUP BY a.artifact_id, ...;
```

## Core Services

### 1. ContextGraphBuilder
**Purpose:** Main orchestrator that coordinates all context gathering

**Key Methods:**
- `BuildEnrichedContext()` - Builds full context graph for an artifact
- Token budget allocation across components
- Cache integration

### 2. ContextSelector
**Purpose:** Scores and ranks candidate artifacts for inclusion

**Scoring Algorithm:**
```go
Relevance Score = (
    0.40 × Semantic Similarity      // Vector cosine similarity
  + 0.25 × Entity Overlap          // Shared people/stakeholders
  + 0.20 × Temporal Proximity      // Recent documents prioritized
  + 0.10 × Document Type Match     // Same category artifacts
  + 0.05 × Fact Density            // Rich metadata artifacts
)
```

**Key Methods:**
- `ScoreCandidate()` - Computes relevance score
- `SelectTopN()` - Selects top candidates within token budget
- `EstimateTokens()` - Estimates token cost for each artifact

### 3. EntityGraphQuery
**Purpose:** Tracks entity relationships and co-occurrences

**Key Methods:**
- `GetPersonRelationships()` - Finds who works with whom
- `FindArtifactsWithSharedPeople()` - Finds docs with entity overlap
- `BuildEntityGraphForArtifact()` - Updates graph after processing

### 4. TemporalOrganizer
**Purpose:** Builds document timelines and detects sequences

**Key Methods:**
- `BuildTimeline()` - Creates before/after timeline for artifact
- `DetectSequences()` - Finds patterns (weekly meetings, monthly reports)
- `FormatTimelineForPrompt()` - Formats timeline for AI

### 5. FactAggregator
**Purpose:** Aggregates facts and detects conflicts

**Key Methods:**
- `AggregateRelatedFacts()` - Collects facts from related artifacts
- `detectConflicts()` - Identifies contradictory facts
- `CompareNumericFacts()` - Statistical analysis (mean, outliers)

### 6. ContextCache
**Purpose:** Multi-level caching for performance

**Cache Levels:**
1. **In-Memory** - 5 min TTL, 100 entries, LRU eviction
2. **Redis** - 24 hour TTL, persistent
3. **Database** - 7 day TTL, long-term storage

**Key Methods:**
- `GetCachedContext()` - Retrieves cached context (checks all levels)
- `CacheContext()` - Stores context in cache
- `InvalidateProgramCache()` - Clears cache when program changes

## Installation & Setup

### Step 1: Run Database Migration

```bash
cd backend/migrations
psql -d cerberus -f 010_context_graph_enhancements.sql
```

This creates:
- 3 new tables
- 1 materialized view
- 5 helper functions
- 1 automatic trigger

### Step 2: Initialize Services in Worker

```go
package main

import (
    "github.com/cerberus/backend/internal/modules/artifacts"
    "github.com/cerberus/backend/internal/platform/ai"
    "github.com/go-redis/redis/v8"
)

func main() {
    // Initialize dependencies
    db := connectDatabase()
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    aiClient := ai.NewClient(apiKey)

    // Initialize repository
    repo := artifacts.NewRepository(db)

    // Initialize AI Analyzer with enriched context
    analyzer, err := artifacts.InitializeAIAnalyzerWithContext(
        aiClient,
        repo,
        redisClient,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Process artifacts with enriched context
    result, err := analyzer.AnalyzeArtifact(ctx, artifact, programContext)
}
```

### Step 3: Enable for Specific Programs (Optional)

```go
// Gradual rollout approach
config := &artifacts.ContextGraphConfig{
    EnableEnrichedContext: true,
    TokenBudget:          4000,
    ScoringWeights:       nil,  // Use defaults
    CacheTTL:             24 * time.Hour,
    EnableCaching:        true,
}

contextBuilder, err := artifacts.InitializeWithConfig(repo, redisClient, config)

// Enable for specific programs
analyzer.SetContextGraphBuilder(contextBuilder)
analyzer.EnableEnrichedContext(true)
```

## Usage Examples

### Example 1: Analyzing Invoice with Context

```go
// Upload invoice
artifact := &artifacts.Artifact{
    ArtifactID: uuid.New(),
    ProgramID:  programID,
    Filename:   "Invoice_Jan_2026.pdf",
    RawContent: sql.NullString{String: invoiceText, Valid: true},
}

// Analyze with enriched context
result, err := analyzer.AnalyzeArtifact(ctx, artifact, programContext)

// Result will include:
// - Cross-references to related invoices
// - Comparison with historical amounts
// - Detection of unusual patterns
// - Entity resolution (vendor names matched across docs)
```

**Sample Enhanced Output:**
```json
{
  "facts": [
    {
      "type": "amount",
      "key": "Invoice Amount",
      "value": "$65,000",
      "cross_reference": "15% above average of prior 6 invoices ($56,500)"
    }
  ],
  "insights": [
    {
      "type": "risk",
      "title": "Higher than average invoice amount",
      "description": "This invoice is 15% above historical average. However, aligns with Amendment #3 which increased hourly rates from $150 to $175.",
      "cross_artifact_pattern": "Rate change documented in Contract_Amendment_003.pdf"
    }
  ],
  "document_relationships": [
    {
      "type": "confirms",
      "related_document": "Contract_Amendment_003.pdf",
      "description": "Invoice reflects new rates from amendment"
    }
  ]
}
```

### Example 2: Detecting Conflicts

When analyzing artifacts, the system automatically detects conflicts:

```
Aggregated Facts:
- Team Size: 50 (Budget_Report.pdf, confidence: 0.95)

⚠️ CONFLICTS DETECTED:
- "Team Size" has 2 conflicting values:
  • 50 (from Budget_Report.pdf, confidence: 0.95)
  • 45 (from Status_Report.pdf, confidence: 0.92)
```

### Example 3: Entity Resolution

**Before (Isolated Analysis):**
```
Extracted: "J. Smith" from "Acme" → New Person
Extracted: "John Smith" from "Acme Corp" → ANOTHER New Person (duplicate!)
```

**After (With Context):**
```
Known Stakeholders:
- John Smith (Project Manager, Acme Corp) - INTERNAL

Extracted: "J. Smith" from "Acme"
  → Matched to John Smith (confidence: 0.92)
  → Appears in 12 related documents
  → Co-occurs with: Jane Doe (PM), Bob Johnson (CFO)
```

## Performance Characteristics

### Token Usage

**Without Context (v1):**
- Average: ~2,500 tokens per analysis
- Cost: ~$0.008 per artifact

**With Context (v2):**
- Average: ~6,500 tokens per analysis (+4,000 context)
- Cost: ~$0.020 per artifact
- **Value:** 3-5x better entity resolution, pattern detection, conflict identification

### Cache Performance

**Expected Hit Rates:**
- First artifact in program: 0% (cold start)
- Subsequent artifacts: 60-80% (partial cache reuse)
- Re-analysis: 95% (full cache hit)

**Performance Impact:**
- Without cache: ~3-5 seconds context building
- With cache: ~100-300ms context retrieval

### Scaling

**Per 1,000 artifacts/month:**
- Token cost increase: ~$12 ($20 vs $8)
- Storage: ~500MB (cached contexts)
- Query load: ~5,000 additional queries (with caching)

## Monitoring & Maintenance

### Key Metrics to Track

1. **Context Quality:**
   - Average relevance score of selected artifacts
   - Cross-artifact reference rate (% of analyses mentioning related docs)
   - Conflict detection rate

2. **Performance:**
   - Cache hit rate (target: >70%)
   - Context build time (target: <5s uncached, <300ms cached)
   - Token budget adherence (must be 100%)

3. **Cost:**
   - Token usage per artifact
   - Monthly cost vs baseline
   - Cost per quality improvement

### Background Jobs

Run these periodically:

**1. Cache Cleanup (Hourly)**
```sql
DELETE FROM artifact_context_cache WHERE expires_at < NOW();
```

**2. Materialized View Refresh (After each artifact)**
```sql
REFRESH MATERIALIZED VIEW CONCURRENTLY artifact_context_summary;
```

**3. Entity Graph Rebuild (Daily)**
- Recompute co-occurrence counts
- Update relationship strengths
- Remove stale edges

### Health Checks

```go
// Check cache stats
stats, _ := cache.GetCacheStats(ctx)
fmt.Printf("Cache entries: %d, Hit rate: %.2f%%\n",
    stats["total_entries"], stats["hit_rate"])

// Check entity graph
entityStats, _ := entityGraph.GetEntityStats(ctx, programID)
fmt.Printf("Total people: %d, Relationships: %d\n",
    entityStats["total_unique_people"], entityStats["total_relationships"])

// Check fact conflicts
factStats, _ := factAggregator.GetFactStats(ctx, programID)
fmt.Printf("Total conflicts: %d, Major: %d\n",
    factStats["total_conflicts"], factStats["major_conflicts"])
```

## Troubleshooting

### Issue: Context build times are slow (>10s)

**Causes:**
- Large program (1000+ artifacts)
- Missing database indexes
- Expensive queries not cached

**Solutions:**
```sql
-- Check if indexes exist
\d artifact_embeddings
\d artifact_persons
\d artifact_entity_graph

-- Rebuild materialized view
REFRESH MATERIALIZED VIEW CONCURRENTLY artifact_context_summary;

-- Check cache hit rate
SELECT COUNT(*) FROM artifact_context_cache WHERE expires_at > NOW();
```

### Issue: Cache misses are high (>50%)

**Causes:**
- Cache invalidation too aggressive
- TTL too short
- Redis connection issues

**Solutions:**
```go
// Increase TTL
cache := NewContextCache(redisClient, repo, 48*time.Hour)

// Warm cache for recent artifacts
cache.WarmCache(ctx, builder, programID, 20)

// Check Redis connection
pong, err := redisClient.Ping(ctx).Result()
```

### Issue: Token budget exceeded

**Causes:**
- Too many related artifacts selected
- Summaries too long
- Context formatting inefficient

**Solutions:**
```go
// Reduce token budget
contextSelector := NewContextSelector(3000, DefaultScoringWeights())

// Adjust scoring to be more selective
weights := ScoringWeights{
    SemanticSimilarity: 0.60,  // Higher threshold
    EntityOverlap:      0.15,
    TemporalProximity:  0.15,
    DocumentTypeMatch:  0.05,
    FactDensity:        0.05,
}
```

## Future Enhancements

### Phase 2 Ideas

1. **Smart Caching Strategies:**
   - Predictive cache warming
   - Program-level context caching
   - Incremental context updates

2. **Advanced Entity Resolution:**
   - Fuzzy name matching
   - Organization hierarchy tracking
   - Automatic stakeholder linking

3. **Pattern Detection:**
   - Anomaly detection (unusual amounts, dates)
   - Trend analysis (cost trajectories)
   - Risk prediction models

4. **Visualization:**
   - Entity relationship graphs
   - Document timeline views
   - Fact conflict dashboards

## Contributing

When adding new features to the context graph system:

1. **Add new repository methods** in `repository_context.go`
2. **Update scoring algorithm** if adding new signals
3. **Add tests** for new components
4. **Update this README** with examples
5. **Monitor token usage** to ensure budgets are respected

## Support

For questions or issues:
- Check logs for context build warnings
- Review cache stats and hit rates
- Validate database indexes exist
- Ensure Redis is connected and healthy
