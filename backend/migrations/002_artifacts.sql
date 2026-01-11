-- Artifacts Module Migration
-- Creates tables for artifact storage and AI-extracted metadata

-- Core artifacts table
CREATE TABLE artifacts (
    artifact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- File metadata
    filename VARCHAR(500) NOT NULL,
    storage_path TEXT NOT NULL,
    file_type VARCHAR(100),
    file_size_bytes BIGINT,
    mime_type VARCHAR(255),

    -- Content
    raw_content TEXT,
    content_hash VARCHAR(64),

    -- Classification
    artifact_category VARCHAR(100),
    artifact_subcategory VARCHAR(100),

    -- AI processing
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    processed_at TIMESTAMPTZ,
    ai_model_version VARCHAR(50),
    ai_processing_time_ms INTEGER,

    -- Temporal
    uploaded_by UUID NOT NULL REFERENCES users(user_id),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version_number INTEGER DEFAULT 1,
    superseded_by UUID REFERENCES artifacts(artifact_id),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT artifacts_content_hash_program_unique UNIQUE(program_id, content_hash)
);

CREATE INDEX idx_artifacts_program ON artifacts(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_type ON artifacts(file_type, artifact_category);
CREATE INDEX idx_artifacts_processing ON artifacts(processing_status)
    WHERE processing_status IN ('pending', 'processing');
CREATE INDEX idx_artifacts_uploaded_at ON artifacts(uploaded_at DESC);
CREATE INDEX idx_artifacts_content_hash ON artifacts(content_hash);
CREATE INDEX idx_artifacts_content_fts ON artifacts
    USING gin(to_tsvector('english', COALESCE(raw_content, '')));

-- AI-extracted topics
CREATE TABLE artifact_topics (
    topic_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    topic_name VARCHAR(255) NOT NULL,
    confidence_score DECIMAL(5,4) CHECK (confidence_score >= 0 AND confidence_score <= 1),

    -- Hierarchical taxonomy support
    parent_topic_id UUID REFERENCES artifact_topics(topic_id),
    topic_level INTEGER DEFAULT 1,

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id, topic_name)
);

CREATE INDEX idx_topics_artifact ON artifact_topics(artifact_id);
CREATE INDEX idx_topics_name ON artifact_topics(topic_name);
CREATE INDEX idx_topics_confidence ON artifact_topics(confidence_score DESC);

-- AI-extracted persons (Named Entity Recognition)
CREATE TABLE artifact_persons (
    person_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    person_name VARCHAR(255) NOT NULL,
    person_role VARCHAR(255),
    person_organization VARCHAR(255),

    -- Context from document
    mention_count INTEGER DEFAULT 1,
    context_snippets JSONB, -- Array of text snippets where person is mentioned
    confidence_score DECIMAL(5,4),

    -- Will link to stakeholders module in Phase 4
    stakeholder_id UUID,

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_artifact_persons_artifact ON artifact_persons(artifact_id);
CREATE INDEX idx_artifact_persons_name ON artifact_persons(person_name);
CREATE INDEX idx_artifact_persons_stakeholder ON artifact_persons(stakeholder_id);

-- AI-extracted facts (structured data)
CREATE TABLE artifact_facts (
    fact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    fact_type VARCHAR(100) NOT NULL, -- date, amount, metric, commitment, deadline
    fact_key VARCHAR(255) NOT NULL,
    fact_value TEXT NOT NULL,

    -- Normalized values for querying
    normalized_value_numeric DECIMAL(20,4),
    normalized_value_date DATE,
    normalized_value_boolean BOOLEAN,

    unit VARCHAR(50),
    confidence_score DECIMAL(5,4),
    context_snippet TEXT,

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id, fact_key, fact_value)
);

CREATE INDEX idx_facts_artifact ON artifact_facts(artifact_id);
CREATE INDEX idx_facts_type ON artifact_facts(fact_type);
CREATE INDEX idx_facts_key ON artifact_facts(fact_key);
CREATE INDEX idx_facts_numeric ON artifact_facts(normalized_value_numeric)
    WHERE normalized_value_numeric IS NOT NULL;
CREATE INDEX idx_facts_date ON artifact_facts(normalized_value_date)
    WHERE normalized_value_date IS NOT NULL;

-- AI-generated insights
CREATE TABLE artifact_insights (
    insight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    insight_type VARCHAR(100) NOT NULL, -- risk, opportunity, action_required, anomaly, decision
    insight_category VARCHAR(100),

    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(50), -- low, medium, high, critical

    suggested_action TEXT,
    impacted_modules VARCHAR(100)[], -- Which modules should see this insight

    confidence_score DECIMAL(5,4),

    -- User feedback
    user_rating INTEGER CHECK (user_rating BETWEEN 1 AND 5),
    user_feedback TEXT,
    is_dismissed BOOLEAN DEFAULT FALSE,

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dismissed_at TIMESTAMPTZ,
    dismissed_by UUID REFERENCES users(user_id)
);

CREATE INDEX idx_insights_artifact ON artifact_insights(artifact_id);
CREATE INDEX idx_insights_type ON artifact_insights(insight_type, severity);
CREATE INDEX idx_insights_dismissed ON artifact_insights(is_dismissed);
CREATE INDEX idx_insights_modules ON artifact_insights USING gin(impacted_modules);

-- Artifact document chunks (for AI processing and embeddings)
CREATE TABLE artifact_chunks (
    chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    chunk_index INTEGER NOT NULL,
    chunk_text TEXT NOT NULL,
    chunk_start_offset INTEGER,
    chunk_end_offset INTEGER,
    token_count INTEGER,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id, chunk_index)
);

CREATE INDEX idx_chunks_artifact ON artifact_chunks(artifact_id, chunk_index);

-- Vector embeddings for semantic search
CREATE TABLE artifact_embeddings (
    embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
    chunk_id UUID NOT NULL REFERENCES artifact_chunks(chunk_id) ON DELETE CASCADE,

    -- Vector embedding (1536 dimensions for Claude/OpenAI embeddings)
    embedding vector(1536),

    embedding_model VARCHAR(100), -- Model used to generate embedding
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(chunk_id)
);

CREATE INDEX idx_embeddings_artifact ON artifact_embeddings(artifact_id);
CREATE INDEX idx_embeddings_chunk ON artifact_embeddings(chunk_id);

-- Create IVFFlat index for fast vector similarity search
-- Lists parameter should be approximately sqrt(total_rows)
-- Start with 100, adjust as data grows
CREATE INDEX idx_embeddings_vector ON artifact_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- AI analysis summary (one per artifact)
CREATE TABLE artifact_summaries (
    summary_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    executive_summary TEXT NOT NULL,
    key_takeaways TEXT[],
    sentiment VARCHAR(50), -- positive, neutral, concern, negative
    priority INTEGER CHECK (priority BETWEEN 1 AND 5),

    confidence_score DECIMAL(5,4),
    ai_model VARCHAR(50),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id)
);

CREATE INDEX idx_summaries_artifact ON artifact_summaries(artifact_id);
CREATE INDEX idx_summaries_priority ON artifact_summaries(priority DESC);
CREATE INDEX idx_summaries_sentiment ON artifact_summaries(sentiment);
