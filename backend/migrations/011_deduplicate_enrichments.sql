-- Deduplicate Risk Enrichments Migration
-- Converts risk_enrichments from duplicated per-risk model to shared enrichments with many-to-many links

-- Step 1: Create new enrichments table (shared across risks)
CREATE TABLE IF NOT EXISTS enrichments (
    enrichment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id),
    source_insight_id UUID REFERENCES artifact_insights(insight_id),

    enrichment_type VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Ensure one enrichment per source insight
    UNIQUE(source_insight_id)
);

CREATE INDEX IF NOT EXISTS idx_enrichments_artifact ON enrichments(artifact_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_enrichments_insight ON enrichments(source_insight_id) WHERE deleted_at IS NULL;

-- Step 2: Create risk-enrichment links table (many-to-many)
CREATE TABLE IF NOT EXISTS risk_enrichment_links (
    link_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_id UUID NOT NULL REFERENCES risks(risk_id) ON DELETE CASCADE,
    enrichment_id UUID NOT NULL REFERENCES enrichments(enrichment_id) ON DELETE CASCADE,

    -- Risk-specific fields (each risk can independently review)
    match_score NUMERIC(5,4) NOT NULL,
    match_method VARCHAR(50) NOT NULL,
    is_relevant BOOLEAN,  -- NULL = pending review, TRUE = relevant, FALSE = not relevant

    -- Review tracking
    reviewed_by UUID REFERENCES users(user_id),
    reviewed_at TIMESTAMPTZ,

    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(risk_id, enrichment_id)
);

CREATE INDEX IF NOT EXISTS idx_risk_enrichment_links_risk ON risk_enrichment_links(risk_id);
CREATE INDEX IF NOT EXISTS idx_risk_enrichment_links_enrichment ON risk_enrichment_links(enrichment_id);
CREATE INDEX IF NOT EXISTS idx_risk_enrichment_links_pending ON risk_enrichment_links(risk_id, is_relevant)
    WHERE is_relevant IS NULL;

-- Step 3: Migrate unique enrichments (deduplicate by source_insight_id)
INSERT INTO enrichments (enrichment_id, artifact_id, source_insight_id, enrichment_type, title, content, created_at, updated_at)
SELECT DISTINCT ON (source_insight_id)
    enrichment_id,
    artifact_id,
    source_insight_id,
    enrichment_type,
    title,
    content,
    created_at,
    created_at as updated_at
FROM risk_enrichments
WHERE source_insight_id IS NOT NULL
  AND deleted_at IS NULL
ORDER BY source_insight_id, created_at ASC;

-- Step 4: Create risk-enrichment links from old enrichments
INSERT INTO risk_enrichment_links (risk_id, enrichment_id, match_score, match_method, is_relevant, reviewed_by, reviewed_at, linked_at)
SELECT
    re.risk_id,
    e.enrichment_id,
    re.match_score,
    re.match_method,
    re.is_relevant,
    re.reviewed_by,
    re.reviewed_at,
    re.created_at
FROM risk_enrichments re
JOIN enrichments e ON re.source_insight_id = e.source_insight_id
WHERE re.deleted_at IS NULL
ON CONFLICT (risk_id, enrichment_id) DO NOTHING;

-- Step 5: Rename old table for safety (don't drop yet)
ALTER TABLE risk_enrichments RENAME TO risk_enrichments_old;

-- Step 6: Add comments for documentation
COMMENT ON TABLE enrichments IS 'Unique enrichments extracted from documents, shared across multiple risks';
COMMENT ON TABLE risk_enrichment_links IS 'Many-to-many links between risks and enrichments, allows independent review per risk';
COMMENT ON COLUMN enrichments.source_insight_id IS 'Reference to the artifact insight this enrichment was derived from';
COMMENT ON COLUMN risk_enrichment_links.is_relevant IS 'NULL = pending review, TRUE = user confirmed relevant, FALSE = user dismissed';
