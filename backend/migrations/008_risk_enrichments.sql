-- Create risk_enrichments table
CREATE TABLE IF NOT EXISTS risk_enrichments (
    enrichment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_id UUID NOT NULL REFERENCES risks(risk_id),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id),
    source_insight_id UUID REFERENCES artifact_insights(insight_id),

    -- Enrichment content
    enrichment_type VARCHAR(100) NOT NULL, -- 'new_evidence', 'impact_update', 'mitigation_idea', 'status_change'
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,

    -- Matching metadata
    match_score DECIMAL(5,4) NOT NULL, -- 0.0000 to 1.0000
    match_method VARCHAR(50) NOT NULL, -- 'entity_overlap', 'category_match', 'semantic_similarity'

    -- User feedback
    is_relevant BOOLEAN DEFAULT NULL, -- null = pending review, true = accepted, false = rejected
    reviewed_by UUID REFERENCES users(user_id),
    reviewed_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_risk_enrichments_risk ON risk_enrichments(risk_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_risk_enrichments_pending ON risk_enrichments(risk_id, is_relevant) WHERE is_relevant IS NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_risk_enrichments_insight ON risk_enrichments(source_insight_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_risk_enrichments_artifact ON risk_enrichments(artifact_id) WHERE deleted_at IS NULL;
