-- Migration: 010_context_graph_enhancements.sql
-- Purpose: Add tables and views for full context graph artifact analysis
-- This enables AI analysis to include rich cross-artifact context including
-- related document summaries, entity relationships, and temporal sequences.

-- ============================================================================
-- Table 1: artifact_context_cache
-- Purpose: Cache table for pre-computed context graphs
-- Caches the enriched context for each artifact to avoid repeated queries
-- ============================================================================

CREATE TABLE artifact_context_cache (
    cache_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Cached context data (JSONB for flexibility)
    -- Contains: related artifacts, entity relationships, temporal context, aggregated facts
    context_data JSONB NOT NULL,

    -- Metadata
    token_count INTEGER NOT NULL,                    -- Estimated token count for this context
    artifacts_included UUID[] NOT NULL,               -- IDs of artifacts included in context
    cache_version INTEGER DEFAULT 1,                  -- Schema version for cache evolution

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,                 -- TTL: typically NOW() + 24 hours

    UNIQUE(artifact_id)
);

CREATE INDEX idx_context_cache_artifact ON artifact_context_cache(artifact_id);
CREATE INDEX idx_context_cache_program ON artifact_context_cache(program_id);
CREATE INDEX idx_context_cache_expires ON artifact_context_cache(expires_at);

COMMENT ON TABLE artifact_context_cache IS 'Caches pre-computed enriched context for artifact AI analysis';
COMMENT ON COLUMN artifact_context_cache.context_data IS 'JSONB containing related artifacts, entity graph, timeline, and facts';
COMMENT ON COLUMN artifact_context_cache.expires_at IS 'Cache expiration timestamp, typically 24 hours from creation';

-- ============================================================================
-- Table 2: artifact_entity_graph
-- Purpose: Entity co-occurrence tracking for faster relationship queries
-- Tracks which people/entities appear together across artifacts
-- ============================================================================

CREATE TABLE artifact_entity_graph (
    edge_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Person mentions that co-occur (ordered: person_id_1 < person_id_2)
    person_id_1 UUID NOT NULL REFERENCES artifact_persons(person_id) ON DELETE CASCADE,
    person_id_2 UUID NOT NULL REFERENCES artifact_persons(person_id) ON DELETE CASCADE,

    -- Co-occurrence statistics
    co_occurrence_count INTEGER DEFAULT 1,            -- Number of artifacts where both appear
    shared_artifact_ids UUID[] NOT NULL,              -- List of artifacts containing both people

    -- Relationship strength (0-1 scale, computed from co-occurrence patterns)
    relationship_strength DECIMAL(5,4) DEFAULT 0.5,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(person_id_1, person_id_2),
    CHECK (person_id_1 < person_id_2)  -- Ensure consistent ordering to avoid duplicates
);

CREATE INDEX idx_entity_graph_program ON artifact_entity_graph(program_id);
CREATE INDEX idx_entity_graph_person1 ON artifact_entity_graph(person_id_1);
CREATE INDEX idx_entity_graph_person2 ON artifact_entity_graph(person_id_2);
CREATE INDEX idx_entity_graph_strength ON artifact_entity_graph(relationship_strength DESC);
CREATE INDEX idx_entity_graph_cooccurrence ON artifact_entity_graph(co_occurrence_count DESC);

COMMENT ON TABLE artifact_entity_graph IS 'Tracks co-occurrence of people across artifacts for relationship analysis';
COMMENT ON COLUMN artifact_entity_graph.relationship_strength IS 'Computed relationship strength (0-1) based on co-occurrence patterns';
COMMENT ON COLUMN artifact_entity_graph.shared_artifact_ids IS 'Array of artifact IDs where both people are mentioned';

-- ============================================================================
-- Table 3: artifact_temporal_sequences
-- Purpose: Document timeline tracking and sequence detection
-- Groups related artifacts into temporal sequences (e.g., contract lifecycle)
-- ============================================================================

CREATE TABLE artifact_temporal_sequences (
    sequence_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Sequence metadata
    sequence_name VARCHAR(255) NOT NULL,              -- e.g., "Q4 Budget Reviews", "Vendor Negotiations"
    sequence_type VARCHAR(100),                       -- meeting_series, project_phase, contract_lifecycle, etc.
    description TEXT,                                 -- Optional description of the sequence

    -- Artifacts in sequence (ordered array)
    artifact_ids UUID[] NOT NULL,                     -- Chronologically ordered artifact IDs
    start_date DATE,                                  -- First artifact date in sequence
    end_date DATE,                                    -- Last artifact date in sequence

    -- Detection method and confidence
    detection_method VARCHAR(50) DEFAULT 'auto',     -- auto, manual, hybrid
    confidence_score DECIMAL(5,4),                   -- Confidence in this sequence grouping (0-1)

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_temporal_sequences_program ON artifact_temporal_sequences(program_id);
CREATE INDEX idx_temporal_sequences_dates ON artifact_temporal_sequences(start_date, end_date);
CREATE INDEX idx_temporal_sequences_type ON artifact_temporal_sequences(sequence_type);
CREATE INDEX idx_temporal_sequences_confidence ON artifact_temporal_sequences(confidence_score DESC);

COMMENT ON TABLE artifact_temporal_sequences IS 'Groups artifacts into temporal sequences for narrative understanding';
COMMENT ON COLUMN artifact_temporal_sequences.artifact_ids IS 'Chronologically ordered array of artifact IDs in this sequence';
COMMENT ON COLUMN artifact_temporal_sequences.detection_method IS 'How this sequence was identified: auto-detected, manually curated, or hybrid';

-- ============================================================================
-- Materialized View: artifact_context_summary
-- Purpose: Fast lookup view for context building
-- Pre-aggregates artifact metadata to avoid repeated joins during context building
-- ============================================================================

CREATE MATERIALIZED VIEW artifact_context_summary AS
SELECT
    a.artifact_id,
    a.program_id,
    a.filename,
    a.artifact_category,
    a.artifact_subcategory,
    a.uploaded_at,

    -- Summary data
    s.executive_summary,
    s.sentiment,
    s.priority,

    -- Aggregated metadata counts
    COUNT(DISTINCT ap.person_id) as person_count,
    COUNT(DISTINCT af.fact_id) as fact_count,
    COUNT(DISTINCT at.topic_id) as topic_count,
    COUNT(DISTINCT ai.insight_id) as insight_count,

    -- Arrays for quick access (filter NULL values)
    array_agg(DISTINCT ap.person_name ORDER BY ap.person_name)
        FILTER (WHERE ap.person_name IS NOT NULL) as mentioned_people,
    array_agg(DISTINCT ap.person_id ORDER BY ap.person_id)
        FILTER (WHERE ap.person_id IS NOT NULL) as person_ids,
    array_agg(DISTINCT at.topic_name ORDER BY at.topic_name)
        FILTER (WHERE at.topic_name IS NOT NULL) as topics,
    array_agg(DISTINCT at.topic_id ORDER BY at.topic_id)
        FILTER (WHERE at.topic_id IS NOT NULL) as topic_ids,

    -- Approximate token count (for budget management)
    -- Formula: summary_length/4 + 100 (base metadata)
    COALESCE((LENGTH(s.executive_summary) / 4)::INTEGER, 0) + 100 as estimated_tokens,

    -- Processing status for filtering
    a.processing_status

FROM artifacts a
LEFT JOIN artifact_summaries s ON a.artifact_id = s.artifact_id
LEFT JOIN artifact_persons ap ON a.artifact_id = ap.artifact_id
LEFT JOIN artifact_facts af ON a.artifact_id = af.artifact_id
LEFT JOIN artifact_topics at ON a.artifact_id = at.artifact_id
LEFT JOIN artifact_insights ai ON a.artifact_id = ai.artifact_id

WHERE a.deleted_at IS NULL
  AND a.processing_status = 'completed'

GROUP BY
    a.artifact_id,
    a.program_id,
    a.filename,
    a.artifact_category,
    a.artifact_subcategory,
    a.uploaded_at,
    a.processing_status,
    s.executive_summary,
    s.sentiment,
    s.priority;

-- Indexes on materialized view for fast lookups
CREATE UNIQUE INDEX idx_context_summary_artifact ON artifact_context_summary(artifact_id);
CREATE INDEX idx_context_summary_program ON artifact_context_summary(program_id);
CREATE INDEX idx_context_summary_uploaded ON artifact_context_summary(uploaded_at DESC);
CREATE INDEX idx_context_summary_category ON artifact_context_summary(artifact_category);
CREATE INDEX idx_context_summary_status ON artifact_context_summary(processing_status);

COMMENT ON MATERIALIZED VIEW artifact_context_summary IS 'Pre-aggregated artifact metadata for fast context building';
COMMENT ON COLUMN artifact_context_summary.estimated_tokens IS 'Approximate token count for this artifact in context prompts';

-- ============================================================================
-- Function: refresh_artifact_context_summary
-- Purpose: Helper function to refresh the materialized view
-- Call this after artifact processing completes
-- ============================================================================

CREATE OR REPLACE FUNCTION refresh_artifact_context_summary()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY artifact_context_summary;
END;
$$;

COMMENT ON FUNCTION refresh_artifact_context_summary IS 'Refreshes the artifact_context_summary materialized view (use after artifact processing)';

-- ============================================================================
-- Function: update_entity_graph_on_artifact
-- Purpose: Update entity co-occurrence graph when artifact is processed
-- This should be called after artifact_persons are extracted
-- ============================================================================

CREATE OR REPLACE FUNCTION update_entity_graph_on_artifact(p_artifact_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_program_id UUID;
    v_person1 RECORD;
    v_person2 RECORD;
BEGIN
    -- Get program ID for this artifact
    SELECT program_id INTO v_program_id
    FROM artifacts
    WHERE artifact_id = p_artifact_id;

    -- For each pair of people in this artifact, update the entity graph
    FOR v_person1 IN
        SELECT person_id, person_name
        FROM artifact_persons
        WHERE artifact_id = p_artifact_id
    LOOP
        FOR v_person2 IN
            SELECT person_id, person_name
            FROM artifact_persons
            WHERE artifact_id = p_artifact_id
              AND person_id > v_person1.person_id  -- Avoid duplicates and self-loops
        LOOP
            -- Insert or update the edge
            INSERT INTO artifact_entity_graph (
                program_id,
                person_id_1,
                person_id_2,
                co_occurrence_count,
                shared_artifact_ids,
                relationship_strength
            )
            VALUES (
                v_program_id,
                v_person1.person_id,
                v_person2.person_id,
                1,
                ARRAY[p_artifact_id],
                0.5
            )
            ON CONFLICT (person_id_1, person_id_2) DO UPDATE
            SET
                co_occurrence_count = artifact_entity_graph.co_occurrence_count + 1,
                shared_artifact_ids = array_append(
                    artifact_entity_graph.shared_artifact_ids,
                    p_artifact_id
                ),
                -- Increase relationship strength based on co-occurrence
                -- Formula: min(1.0, co_occurrences / 10)
                relationship_strength = LEAST(1.0, (artifact_entity_graph.co_occurrence_count + 1)::DECIMAL / 10),
                updated_at = NOW();
        END LOOP;
    END LOOP;
END;
$$;

COMMENT ON FUNCTION update_entity_graph_on_artifact IS 'Updates entity co-occurrence graph after artifact processing';

-- ============================================================================
-- Function: cleanup_expired_context_cache
-- Purpose: Remove expired cache entries (run periodically)
-- ============================================================================

CREATE OR REPLACE FUNCTION cleanup_expired_context_cache()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM artifact_context_cache
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;

    RETURN v_deleted_count;
END;
$$;

COMMENT ON FUNCTION cleanup_expired_context_cache IS 'Deletes expired context cache entries and returns count deleted';

-- ============================================================================
-- Trigger: auto_update_entity_graph
-- Purpose: Automatically update entity graph after artifact processing
-- ============================================================================

CREATE OR REPLACE FUNCTION trigger_update_entity_graph()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Only update if status changed to 'completed'
    IF NEW.processing_status = 'completed' AND OLD.processing_status != 'completed' THEN
        -- Update entity graph asynchronously (don't block the update)
        PERFORM update_entity_graph_on_artifact(NEW.artifact_id);
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER auto_update_entity_graph_trigger
AFTER UPDATE OF processing_status ON artifacts
FOR EACH ROW
WHEN (NEW.processing_status = 'completed')
EXECUTE FUNCTION trigger_update_entity_graph();

COMMENT ON TRIGGER auto_update_entity_graph_trigger ON artifacts IS 'Automatically updates entity graph when artifact processing completes';

-- ============================================================================
-- Initial Data / Backfill
-- ============================================================================

-- Backfill entity graph for existing artifacts
-- This can take a while for large datasets, so it's optional
-- Uncomment to run initial backfill:

/*
DO $$
DECLARE
    v_artifact RECORD;
BEGIN
    FOR v_artifact IN
        SELECT artifact_id
        FROM artifacts
        WHERE processing_status = 'completed'
          AND deleted_at IS NULL
        ORDER BY uploaded_at DESC
        LIMIT 1000  -- Limit to recent 1000 artifacts for initial backfill
    LOOP
        PERFORM update_entity_graph_on_artifact(v_artifact.artifact_id);
    END LOOP;
END;
$$;
*/

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Verify tables were created
-- SELECT table_name FROM information_schema.tables
-- WHERE table_schema = 'public'
--   AND table_name LIKE 'artifact_%'
-- ORDER BY table_name;

-- Verify materialized view
-- SELECT * FROM artifact_context_summary LIMIT 5;

-- Check entity graph
-- SELECT COUNT(*) as total_edges FROM artifact_entity_graph;

-- Check context cache
-- SELECT COUNT(*) as total_cached FROM artifact_context_cache;

-- Check temporal sequences
-- SELECT COUNT(*) as total_sequences FROM artifact_temporal_sequences;
