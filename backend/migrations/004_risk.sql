-- Risk & Issue Management Module Migration
-- Creates tables for risk register, suggestions, mitigations, and conversations

-- Main risks table
CREATE TABLE risks (
    risk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Risk identification
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    -- Risk assessment
    probability VARCHAR(50) NOT NULL, -- very_low, low, medium, high, very_high
    impact VARCHAR(50) NOT NULL, -- very_low, low, medium, high, very_high
    severity VARCHAR(50) NOT NULL, -- calculated: low, medium, high, critical

    -- Categorization
    category VARCHAR(100) NOT NULL, -- technical, financial, schedule, resource, external

    -- Status workflow
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- suggested, open, monitoring, mitigated, closed, realized

    -- Ownership
    owner_user_id UUID REFERENCES users(user_id),
    owner_name VARCHAR(255),

    -- Temporal
    identified_date DATE NOT NULL DEFAULT CURRENT_DATE,
    target_resolution_date DATE,
    closed_date DATE,
    realized_date DATE,

    -- AI metadata
    ai_confidence_score DECIMAL(5,4),
    ai_detected_at TIMESTAMPTZ,

    -- User actions
    created_by UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT risks_probability_check CHECK (probability IN ('very_low', 'low', 'medium', 'high', 'very_high')),
    CONSTRAINT risks_impact_check CHECK (impact IN ('very_low', 'low', 'medium', 'high', 'very_high')),
    CONSTRAINT risks_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT risks_category_check CHECK (category IN ('technical', 'financial', 'schedule', 'resource', 'external')),
    CONSTRAINT risks_status_check CHECK (status IN ('suggested', 'open', 'monitoring', 'mitigated', 'closed', 'realized'))
);

CREATE INDEX idx_risks_program ON risks(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_risks_status ON risks(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_risks_severity ON risks(severity, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_risks_category ON risks(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_risks_owner ON risks(owner_user_id) WHERE deleted_at IS NULL;

-- Risk suggestions (AI-detected risks awaiting user approval)
CREATE TABLE risk_suggestions (
    suggestion_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Suggested risk details
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    rationale TEXT NOT NULL, -- Why AI thinks this is a risk

    -- Risk assessment
    suggested_probability VARCHAR(50) NOT NULL,
    suggested_impact VARCHAR(50) NOT NULL,
    suggested_severity VARCHAR(50) NOT NULL,
    suggested_category VARCHAR(100) NOT NULL,

    -- Source tracking
    source_type VARCHAR(100) NOT NULL, -- artifact_insight, financial_variance, schedule_delay
    source_artifact_ids UUID[], -- Artifacts that support this risk
    source_insight_id UUID, -- Link to artifact_insights if applicable
    source_variance_id UUID, -- Link to financial_variances if applicable

    -- AI metadata
    ai_confidence_score DECIMAL(5,4),
    ai_detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- User actions
    is_approved BOOLEAN DEFAULT FALSE,
    is_dismissed BOOLEAN DEFAULT FALSE,
    approved_by UUID REFERENCES users(user_id),
    approved_at TIMESTAMPTZ,
    dismissed_by UUID REFERENCES users(user_id),
    dismissed_at TIMESTAMPTZ,
    dismissal_reason TEXT,
    created_risk_id UUID REFERENCES risks(risk_id) ON DELETE SET NULL,

    CONSTRAINT risk_suggestions_probability_check CHECK (suggested_probability IN ('very_low', 'low', 'medium', 'high', 'very_high')),
    CONSTRAINT risk_suggestions_impact_check CHECK (suggested_impact IN ('very_low', 'low', 'medium', 'high', 'very_high')),
    CONSTRAINT risk_suggestions_severity_check CHECK (suggested_severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT risk_suggestions_category_check CHECK (suggested_category IN ('technical', 'financial', 'schedule', 'resource', 'external'))
);

CREATE INDEX idx_risk_suggestions_program ON risk_suggestions(program_id);
CREATE INDEX idx_risk_suggestions_pending ON risk_suggestions(program_id)
    WHERE is_approved = FALSE AND is_dismissed = FALSE;
CREATE INDEX idx_risk_suggestions_source_artifacts ON risk_suggestions USING gin(source_artifact_ids);
CREATE INDEX idx_risk_suggestions_severity ON risk_suggestions(suggested_severity);

-- Risk mitigations (actions to reduce or eliminate risks)
CREATE TABLE risk_mitigations (
    mitigation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_id UUID NOT NULL REFERENCES risks(risk_id) ON DELETE CASCADE,

    -- Mitigation details
    strategy VARCHAR(100) NOT NULL, -- avoid, transfer, mitigate, accept
    action_description TEXT NOT NULL,

    -- Effectiveness tracking
    expected_probability_reduction VARCHAR(50), -- Target probability after mitigation
    expected_impact_reduction VARCHAR(50), -- Target impact after mitigation
    effectiveness_rating INTEGER CHECK (effectiveness_rating BETWEEN 1 AND 5),

    -- Execution
    status VARCHAR(50) NOT NULL DEFAULT 'planned', -- planned, in_progress, completed, abandoned
    assigned_to UUID REFERENCES users(user_id),
    target_completion_date DATE,
    actual_completion_date DATE,

    -- Cost tracking
    estimated_cost DECIMAL(15,2),
    actual_cost DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'USD',

    -- Metadata
    created_by UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT risk_mitigations_strategy_check CHECK (strategy IN ('avoid', 'transfer', 'mitigate', 'accept')),
    CONSTRAINT risk_mitigations_status_check CHECK (status IN ('planned', 'in_progress', 'completed', 'abandoned'))
);

CREATE INDEX idx_risk_mitigations_risk ON risk_mitigations(risk_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_risk_mitigations_status ON risk_mitigations(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_risk_mitigations_assigned ON risk_mitigations(assigned_to) WHERE deleted_at IS NULL;

-- Risk artifact links (many-to-many relationship)
CREATE TABLE risk_artifact_links (
    link_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_id UUID NOT NULL REFERENCES risks(risk_id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    -- Link metadata
    link_type VARCHAR(100) NOT NULL, -- evidence, impact_analysis, mitigation_plan, related
    description TEXT,

    created_by UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(risk_id, artifact_id, link_type)
);

CREATE INDEX idx_risk_artifact_links_risk ON risk_artifact_links(risk_id);
CREATE INDEX idx_risk_artifact_links_artifact ON risk_artifact_links(artifact_id);

-- Conversation threads (for risk discussions)
CREATE TABLE conversation_threads (
    thread_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_id UUID NOT NULL REFERENCES risks(risk_id) ON DELETE CASCADE,

    -- Thread metadata
    title VARCHAR(500) NOT NULL,
    thread_type VARCHAR(100) DEFAULT 'discussion', -- discussion, status_update, decision, escalation

    -- Status
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(user_id),

    -- Tracking
    message_count INTEGER DEFAULT 0,
    last_message_at TIMESTAMPTZ,

    created_by UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_conversation_threads_risk ON conversation_threads(risk_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversation_threads_unresolved ON conversation_threads(risk_id, is_resolved)
    WHERE deleted_at IS NULL AND is_resolved = FALSE;

-- Conversation messages (thread messages with markdown support)
CREATE TABLE conversation_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES conversation_threads(thread_id) ON DELETE CASCADE,

    -- Message content
    message_text TEXT NOT NULL,
    message_format VARCHAR(50) DEFAULT 'markdown', -- markdown, plain_text

    -- Mentions
    mentioned_user_ids UUID[], -- Array of user IDs mentioned in message

    -- Metadata
    created_by UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_conversation_messages_thread ON conversation_messages(thread_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversation_messages_created_at ON conversation_messages(thread_id, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversation_messages_mentions ON conversation_messages USING gin(mentioned_user_ids);

-- Trigger to update risks updated_at timestamp
CREATE OR REPLACE FUNCTION update_risk_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_risks_updated_at
    BEFORE UPDATE ON risks
    FOR EACH ROW
    EXECUTE FUNCTION update_risk_updated_at();

CREATE TRIGGER trigger_risk_mitigations_updated_at
    BEFORE UPDATE ON risk_mitigations
    FOR EACH ROW
    EXECUTE FUNCTION update_risk_updated_at();

-- Trigger to update conversation thread message count and last message time
CREATE OR REPLACE FUNCTION update_conversation_thread_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE conversation_threads
        SET message_count = message_count + 1,
            last_message_at = NEW.created_at
        WHERE thread_id = NEW.thread_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE conversation_threads
        SET message_count = message_count - 1
        WHERE thread_id = OLD.thread_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_conversation_message_stats
    AFTER INSERT OR DELETE ON conversation_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_conversation_thread_stats();

-- Function to calculate risk severity from probability and impact
CREATE OR REPLACE FUNCTION calculate_risk_severity(
    probability VARCHAR(50),
    impact VARCHAR(50)
) RETURNS VARCHAR(50) AS $$
DECLARE
    prob_score INTEGER;
    impact_score INTEGER;
    total_score INTEGER;
BEGIN
    -- Convert probability to numeric score
    prob_score := CASE probability
        WHEN 'very_low' THEN 1
        WHEN 'low' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'high' THEN 4
        WHEN 'very_high' THEN 5
        ELSE 0
    END;

    -- Convert impact to numeric score
    impact_score := CASE impact
        WHEN 'very_low' THEN 1
        WHEN 'low' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'high' THEN 4
        WHEN 'very_high' THEN 5
        ELSE 0
    END;

    -- Calculate total score (multiply for risk matrix)
    total_score := prob_score * impact_score;

    -- Map to severity
    RETURN CASE
        WHEN total_score <= 4 THEN 'low'
        WHEN total_score <= 9 THEN 'medium'
        WHEN total_score <= 16 THEN 'high'
        ELSE 'critical'
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger to automatically calculate severity on insert/update
CREATE OR REPLACE FUNCTION set_risk_severity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.severity := calculate_risk_severity(NEW.probability, NEW.impact);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_risks_severity
    BEFORE INSERT OR UPDATE OF probability, impact ON risks
    FOR EACH ROW
    EXECUTE FUNCTION set_risk_severity();

-- Similarly for risk suggestions
CREATE OR REPLACE FUNCTION set_risk_suggestion_severity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.suggested_severity := calculate_risk_severity(NEW.suggested_probability, NEW.suggested_impact);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_risk_suggestions_severity
    BEFORE INSERT OR UPDATE OF suggested_probability, suggested_impact ON risk_suggestions
    FOR EACH ROW
    EXECUTE FUNCTION set_risk_suggestion_severity();
