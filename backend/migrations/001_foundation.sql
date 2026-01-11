-- Foundation Migration
-- Creates core tables: users, programs, program_users

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Users table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    auth_provider VARCHAR(50),
    auth_provider_id VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_auth ON users(auth_provider, auth_provider_id);

-- Programs table
CREATE TABLE programs (
    program_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_name VARCHAR(255) NOT NULL,
    program_code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    start_date DATE,
    end_date DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(user_id),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT programs_dates_check CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_programs_status ON programs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_code ON programs(program_code);
CREATE INDEX idx_programs_created_by ON programs(created_by);

-- Program users (access control)
CREATE TABLE program_users (
    program_user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- 'admin', 'contributor', 'viewer'
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(user_id),
    revoked_at TIMESTAMPTZ,

    UNIQUE(program_id, user_id, revoked_at)
);

CREATE INDEX idx_program_users_lookup ON program_users(program_id, user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_program_users_user ON program_users(user_id);

-- Audit logs table
CREATE TABLE audit_logs (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    changed_fields JSONB,
    changed_by UUID NOT NULL REFERENCES users(user_id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    change_reason TEXT,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, changed_at DESC);
CREATE INDEX idx_audit_logs_program ON audit_logs(program_id, changed_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs(changed_by);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- Events table (for NATS persistence/tracking)
CREATE TABLE events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    source_module VARCHAR(100) NOT NULL,
    correlation_id UUID,
    payload JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    ai_generated BOOLEAN DEFAULT FALSE,
    ai_confidence DECIMAL(5,4),
    artifact_refs UUID[],
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_program ON events(program_id);
CREATE INDEX idx_events_published ON events(published_at DESC);
CREATE INDEX idx_events_processed ON events(processed, published_at) WHERE NOT processed;
CREATE INDEX idx_events_correlation ON events(correlation_id);

-- AI usage tracking
CREATE TABLE ai_usage (
    usage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    module VARCHAR(100) NOT NULL,
    job_type VARCHAR(100),
    model VARCHAR(50) NOT NULL,
    tokens_input INTEGER NOT NULL,
    tokens_output INTEGER NOT NULL,
    tokens_cached INTEGER DEFAULT 0,
    tokens_total INTEGER NOT NULL,
    cost_usd DECIMAL(10,4) NOT NULL,
    duration_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_program ON ai_usage(program_id);
CREATE INDEX idx_ai_usage_date ON ai_usage(created_at DESC);
CREATE INDEX idx_ai_usage_module ON ai_usage(module, created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for programs table
CREATE TRIGGER update_programs_updated_at BEFORE UPDATE ON programs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
