# Cerberus Data Model

This document provides comprehensive database schema design for all 10 modules in Cerberus.

---

## Design Principles

### 1. Artifact-Centric Architecture
All entities can link to artifacts via `artifact_ids UUID[]` arrays. This provides:
- **Context:** Every risk, decision, invoice traces back to source documents
- **Traceability:** Full audit trail from insight to evidence
- **AI Leverage:** Artifacts provide context for AI analysis across modules

### 2. AI-First Schema Design
- **JSONB Columns:** Flexible storage for AI-extracted metadata that evolves
- **Confidence Scores:** Track AI confidence for all extractions
- **Vector Embeddings:** pgvector for semantic search
- **Structured + Unstructured:** Normalized tables for queries, JSONB for flexibility

### 3. Multi-Tenancy via Program Isolation
- **Every table has `program_id`** for data isolation
- Row-Level Security (RLS) enforces program boundaries
- No cross-program data leakage

### 4. Temporal Data & Audit Trails
- **Soft Deletes:** `deleted_at TIMESTAMPTZ` for recoverability
- **Versioning:** Track versions with `superseded_by` foreign keys
- **Audit Logs:** Universal `audit_logs` table tracks all changes
- **Timestamps:** `created_at`, `updated_at` on all core tables

### 5. Event Sourcing for Integration
- **Events Table:** All cross-module communication via events
- **Correlation IDs:** Track workflows end-to-end
- **Event Replay:** Can reconstruct state from event history

---

## PostgreSQL Extensions Required

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";      -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Cryptographic functions
CREATE EXTENSION IF NOT EXISTS "vector";         -- pgvector for embeddings
CREATE EXTENSION IF NOT EXISTS "btree_gin";      -- Composite GIN indexes
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Fuzzy text search
```

---

## Foundation Tables

### Programs (Multi-Tenancy Root)

```sql
CREATE TABLE programs (
    program_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_name VARCHAR(255) NOT NULL,
    program_code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    start_date DATE,
    end_date DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT programs_dates_check CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_programs_status ON programs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_code ON programs(program_code);
```

### Users

```sql
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
```

### Program Users (Access Control)

```sql
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
```

---

## Module 1: Program Artifacts (CORE)

### Artifacts

```sql
CREATE TABLE artifacts (
    artifact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- File metadata
    filename VARCHAR(500) NOT NULL,
    file_path TEXT NOT NULL,
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
    processing_status VARCHAR(50) DEFAULT 'pending',
    processed_at TIMESTAMPTZ,
    ai_model_version VARCHAR(50),

    -- Temporal
    uploaded_by UUID NOT NULL REFERENCES users(user_id),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version_number INTEGER DEFAULT 1,
    superseded_by UUID REFERENCES artifacts(artifact_id),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT artifacts_content_hash_unique UNIQUE(program_id, content_hash, deleted_at)
);

CREATE INDEX idx_artifacts_program ON artifacts(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_type ON artifacts(file_type, artifact_category);
CREATE INDEX idx_artifacts_processing ON artifacts(processing_status)
    WHERE processing_status IN ('pending', 'processing');
CREATE INDEX idx_artifacts_uploaded_at ON artifacts(uploaded_at DESC);
CREATE INDEX idx_artifacts_content_hash ON artifacts(content_hash);
CREATE INDEX idx_artifacts_content_fts ON artifacts
    USING gin(to_tsvector('english', raw_content));
```

### Artifact Topics (AI-Extracted Taxonomy)

```sql
CREATE TABLE artifact_topics (
    topic_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    topic_name VARCHAR(255) NOT NULL,
    confidence_score DECIMAL(5,4) CHECK (confidence_score >= 0 AND confidence_score <= 1),

    -- Hierarchical taxonomy
    parent_topic_id UUID REFERENCES artifact_topics(topic_id),
    topic_level INTEGER DEFAULT 1,

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id, topic_name)
);

CREATE INDEX idx_topics_artifact ON artifact_topics(artifact_id);
CREATE INDEX idx_topics_name ON artifact_topics(topic_name);
CREATE INDEX idx_topics_confidence ON artifact_topics(confidence_score DESC);
```

### Artifact Persons (Named Entity Recognition)

```sql
CREATE TABLE artifact_persons (
    person_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    person_name VARCHAR(255) NOT NULL,
    person_role VARCHAR(255),
    person_organization VARCHAR(255),

    -- Linking to stakeholder management
    stakeholder_id UUID,

    -- Context from document
    mention_count INTEGER DEFAULT 1,
    context_snippets JSONB,
    confidence_score DECIMAL(5,4),

    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_artifact_persons_artifact ON artifact_persons(artifact_id);
CREATE INDEX idx_artifact_persons_name ON artifact_persons(person_name);
CREATE INDEX idx_artifact_persons_stakeholder ON artifact_persons(stakeholder_id);
```

### Artifact Facts (Structured Data Extraction)

```sql
CREATE TABLE artifact_facts (
    fact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    fact_type VARCHAR(100) NOT NULL,
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
```

### Artifact Insights (AI Analysis)

```sql
CREATE TABLE artifact_insights (
    insight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    insight_type VARCHAR(100) NOT NULL,
    insight_category VARCHAR(100),

    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(50),

    suggested_action TEXT,
    impacted_modules VARCHAR(100)[],

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
```

### Artifact Embeddings (Vector Search)

```sql
CREATE TABLE artifact_embeddings (
    embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,

    -- Chunking strategy
    chunk_index INTEGER DEFAULT 0,
    chunk_text TEXT,
    chunk_start_offset INTEGER,
    chunk_end_offset INTEGER,

    -- Vector embedding
    embedding vector(1536),

    embedding_model VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(artifact_id, chunk_index)
);

CREATE INDEX idx_embeddings_artifact ON artifact_embeddings(artifact_id);
CREATE INDEX idx_embeddings_vector ON artifact_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

---

## Module 2: Program Financial Reporting

### Rate Cards

```sql
CREATE TABLE rate_cards (
    rate_card_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,
    effective_start_date DATE NOT NULL,
    effective_end_date DATE,

    currency VARCHAR(3) DEFAULT 'USD',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    version_number INTEGER DEFAULT 1,
    superseded_by UUID REFERENCES rate_cards(rate_card_id),

    deleted_at TIMESTAMPTZ,

    CONSTRAINT rate_cards_dates_check CHECK (
        effective_end_date IS NULL OR effective_end_date >= effective_start_date
    )
);

CREATE INDEX idx_rate_cards_program ON rate_cards(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_rate_cards_effective ON rate_cards(effective_start_date, effective_end_date);
```

### Rate Card Items

```sql
CREATE TABLE rate_card_items (
    item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_card_id UUID NOT NULL REFERENCES rate_cards(rate_card_id) ON DELETE CASCADE,

    role_title VARCHAR(255) NOT NULL,
    rate_type VARCHAR(50) NOT NULL,
    rate_amount DECIMAL(15,2) NOT NULL,

    seniority_level VARCHAR(50),
    skill_category VARCHAR(100),

    notes TEXT,

    UNIQUE(rate_card_id, role_title, rate_type)
);

CREATE INDEX idx_rate_card_items_card ON rate_card_items(rate_card_id);
CREATE INDEX idx_rate_card_items_role ON rate_card_items(role_title);
```

### Invoices

```sql
CREATE TABLE invoices (
    invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    invoice_number VARCHAR(100) NOT NULL,
    vendor_name VARCHAR(255) NOT NULL,
    vendor_id UUID,

    invoice_date DATE NOT NULL,
    due_date DATE,
    payment_date DATE,

    subtotal_amount DECIMAL(15,2) NOT NULL,
    tax_amount DECIMAL(15,2) DEFAULT 0,
    total_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    payment_status VARCHAR(50) DEFAULT 'pending',
    approval_status VARCHAR(50) DEFAULT 'pending',

    artifact_id UUID REFERENCES artifacts(artifact_id),
    rate_card_id UUID REFERENCES rate_cards(rate_card_id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ,

    CONSTRAINT invoices_amounts_check CHECK (total_amount = subtotal_amount + tax_amount),
    CONSTRAINT invoices_dates_check CHECK (due_date IS NULL OR due_date >= invoice_date)
);

CREATE INDEX idx_invoices_program ON invoices(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_vendor ON invoices(vendor_name);
CREATE INDEX idx_invoices_date ON invoices(invoice_date DESC);
CREATE INDEX idx_invoices_status ON invoices(payment_status, approval_status);
CREATE INDEX idx_invoices_artifact ON invoices(artifact_id);
```

### Invoice Line Items

```sql
CREATE TABLE invoice_line_items (
    line_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(invoice_id) ON DELETE CASCADE,

    line_number INTEGER NOT NULL,
    description TEXT NOT NULL,

    spend_category VARCHAR(100),
    spend_subcategory VARCHAR(100),

    quantity DECIMAL(15,4),
    unit_of_measure VARCHAR(50),
    unit_rate DECIMAL(15,4),
    line_amount DECIMAL(15,2) NOT NULL,

    budget_category_id UUID,

    UNIQUE(invoice_id, line_number)
);

CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_category ON invoice_line_items(spend_category, spend_subcategory);
CREATE INDEX idx_line_items_budget ON invoice_line_items(budget_category_id);
```

### Budget Categories

```sql
CREATE TABLE budget_categories (
    budget_category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    category_name VARCHAR(255) NOT NULL,
    category_code VARCHAR(50),
    parent_category_id UUID REFERENCES budget_categories(budget_category_id),

    budgeted_amount DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'USD',

    fiscal_year INTEGER,
    fiscal_quarter INTEGER CHECK (fiscal_quarter BETWEEN 1 AND 4),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, category_code, fiscal_year, fiscal_quarter)
);

CREATE INDEX idx_budget_categories_program ON budget_categories(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_budget_categories_parent ON budget_categories(parent_category_id);
CREATE INDEX idx_budget_categories_fiscal ON budget_categories(fiscal_year, fiscal_quarter);
```

---

## Module 3: Risk & Issue Management

### Risks

```sql
CREATE TABLE risks (
    risk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    risk_number VARCHAR(50),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    risk_category VARCHAR(100),
    risk_type VARCHAR(50),

    probability VARCHAR(50) NOT NULL,
    impact VARCHAR(50) NOT NULL,
    risk_score INTEGER GENERATED ALWAYS AS (
        (CASE probability
            WHEN 'very_low' THEN 1 WHEN 'low' THEN 2 WHEN 'medium' THEN 3
            WHEN 'high' THEN 4 WHEN 'very_high' THEN 5 END) *
        (CASE impact
            WHEN 'very_low' THEN 1 WHEN 'low' THEN 2 WHEN 'medium' THEN 3
            WHEN 'high' THEN 4 WHEN 'very_high' THEN 5 END)
    ) STORED,

    residual_probability VARCHAR(50),
    residual_impact VARCHAR(50),
    residual_risk_score INTEGER,

    mitigation_strategy TEXT,
    contingency_plan TEXT,

    owner_id UUID REFERENCES users(user_id),
    status VARCHAR(50) DEFAULT 'open',

    is_ai_suggested BOOLEAN DEFAULT FALSE,
    ai_confidence_score DECIMAL(5,4),
    ai_suggested_at TIMESTAMPTZ,

    artifact_ids UUID[],

    identified_date DATE DEFAULT CURRENT_DATE,
    target_closure_date DATE,
    actual_closure_date DATE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_risks_program ON risks(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_risks_status ON risks(status);
CREATE INDEX idx_risks_score ON risks(risk_score DESC);
CREATE INDEX idx_risks_owner ON risks(owner_id);
CREATE INDEX idx_risks_ai_suggested ON risks(is_ai_suggested, ai_confidence_score);
CREATE INDEX idx_risks_artifacts ON risks USING gin(artifact_ids);
```

### Issues

```sql
CREATE TABLE issues (
    issue_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    issue_number VARCHAR(50),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    issue_category VARCHAR(100),
    priority VARCHAR(50) NOT NULL,
    severity VARCHAR(50) NOT NULL,

    owner_id UUID REFERENCES users(user_id),
    status VARCHAR(50) DEFAULT 'open',
    resolution TEXT,

    related_risk_id UUID REFERENCES risks(risk_id),
    artifact_ids UUID[],

    identified_date DATE DEFAULT CURRENT_DATE,
    due_date DATE,
    resolved_date DATE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_issues_program ON issues(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_priority ON issues(priority, severity);
CREATE INDEX idx_issues_owner ON issues(owner_id);
CREATE INDEX idx_issues_risk ON issues(related_risk_id);
```

---

## Cross-Module: Conversation Threads

### Conversation Threads (Unified Pattern)

```sql
CREATE TABLE conversation_threads (
    thread_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,

    thread_title VARCHAR(500),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),

    is_locked BOOLEAN DEFAULT FALSE,
    locked_at TIMESTAMPTZ,
    locked_by UUID REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_threads_entity ON conversation_threads(entity_type, entity_id);
CREATE INDEX idx_threads_program ON conversation_threads(program_id);
```

### Conversation Messages

```sql
CREATE TABLE conversation_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES conversation_threads(thread_id) ON DELETE CASCADE,

    message_body TEXT NOT NULL,
    message_type VARCHAR(50) DEFAULT 'comment',

    mentioned_user_ids UUID[],
    artifact_ids UUID[],

    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_by UUID NOT NULL REFERENCES users(user_id),

    edited_at TIMESTAMPTZ,
    edited_by UUID REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(user_id)
);

CREATE INDEX idx_messages_thread ON conversation_messages(thread_id, posted_at);
CREATE INDEX idx_messages_user ON conversation_messages(posted_by);
CREATE INDEX idx_messages_mentions ON conversation_messages USING gin(mentioned_user_ids);
```

---

## Module 4: Communications

### Communication Plans

```sql
CREATE TABLE communication_plans (
    plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    plan_name VARCHAR(255) NOT NULL,
    description TEXT,

    frequency VARCHAR(50),
    next_scheduled_date DATE,

    status VARCHAR(50) DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comm_plans_program ON communication_plans(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_comm_plans_next_date ON communication_plans(next_scheduled_date)
    WHERE status = 'active' AND deleted_at IS NULL;
```

### Communication Templates

```sql
CREATE TABLE communication_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    template_name VARCHAR(255) NOT NULL,
    template_type VARCHAR(50),

    subject_template TEXT,
    body_template TEXT NOT NULL,

    template_variables JSONB,

    is_default BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comm_templates_program ON communication_templates(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_comm_templates_type ON communication_templates(template_type);
```

### Communications

```sql
CREATE TABLE communications (
    communication_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    plan_id UUID REFERENCES communication_plans(plan_id),
    template_id UUID REFERENCES communication_templates(template_id),

    communication_type VARCHAR(50) NOT NULL,
    subject TEXT,
    body TEXT NOT NULL,

    recipient_stakeholder_ids UUID[],
    recipient_emails TEXT[],

    status VARCHAR(50) DEFAULT 'draft',

    scheduled_send_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,

    artifact_ids UUID[],

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_communications_program ON communications(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_communications_status ON communications(status, scheduled_send_at);
CREATE INDEX idx_communications_plan ON communications(plan_id);
```

---

## Module 5: Stakeholder Management

### Stakeholders

```sql
CREATE TABLE stakeholders (
    stakeholder_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    organization VARCHAR(255),
    job_title VARCHAR(255),

    stakeholder_type VARCHAR(50),
    influence_level VARCHAR(50),
    interest_level VARCHAR(50),

    engagement_approach TEXT,
    communication_preferences JSONB,

    user_id UUID REFERENCES users(user_id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_stakeholders_program ON stakeholders(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_stakeholders_email ON stakeholders(email);
CREATE INDEX idx_stakeholders_user ON stakeholders(user_id);
CREATE INDEX idx_stakeholders_influence ON stakeholders(influence_level, interest_level);

ALTER TABLE artifact_persons
    ADD CONSTRAINT fk_artifact_persons_stakeholder
    FOREIGN KEY (stakeholder_id) REFERENCES stakeholders(stakeholder_id);
```

### Stakeholder Groups

```sql
CREATE TABLE stakeholder_groups (
    group_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    group_name VARCHAR(255) NOT NULL,
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, group_name)
);

CREATE INDEX idx_stakeholder_groups_program ON stakeholder_groups(program_id)
    WHERE deleted_at IS NULL;
```

### Stakeholder Group Members

```sql
CREATE TABLE stakeholder_group_members (
    member_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES stakeholder_groups(group_id) ON DELETE CASCADE,
    stakeholder_id UUID NOT NULL REFERENCES stakeholders(stakeholder_id) ON DELETE CASCADE,

    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID REFERENCES users(user_id),

    removed_at TIMESTAMPTZ,

    UNIQUE(group_id, stakeholder_id, removed_at)
);

CREATE INDEX idx_group_members_group ON stakeholder_group_members(group_id)
    WHERE removed_at IS NULL;
CREATE INDEX idx_group_members_stakeholder ON stakeholder_group_members(stakeholder_id);
```

### Communication Enrollments

```sql
CREATE TABLE communication_enrollments (
    enrollment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    plan_id UUID NOT NULL REFERENCES communication_plans(plan_id) ON DELETE CASCADE,
    stakeholder_id UUID REFERENCES stakeholders(stakeholder_id),
    group_id UUID REFERENCES stakeholder_groups(group_id),

    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enrolled_by UUID REFERENCES users(user_id),

    unsubscribed_at TIMESTAMPTZ,

    CONSTRAINT enrollment_target_check CHECK (
        (stakeholder_id IS NOT NULL AND group_id IS NULL) OR
        (stakeholder_id IS NULL AND group_id IS NOT NULL)
    )
);

CREATE INDEX idx_enrollments_plan ON communication_enrollments(plan_id)
    WHERE unsubscribed_at IS NULL;
CREATE INDEX idx_enrollments_stakeholder ON communication_enrollments(stakeholder_id);
```

---

## Module 6: Decision Log

### Decisions

```sql
CREATE TABLE decisions (
    decision_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    decision_number VARCHAR(50),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    decision_category VARCHAR(100),
    rationale TEXT NOT NULL,
    alternatives_considered TEXT,

    decision_maker_id UUID REFERENCES users(user_id),
    approved_by_ids UUID[],

    impact_description TEXT,
    impacted_modules VARCHAR(100)[],

    decision_status VARCHAR(50) DEFAULT 'proposed',

    decision_date DATE,
    effective_date DATE,
    review_date DATE,

    artifact_ids UUID[],
    related_risk_ids UUID[],
    related_change_request_ids UUID[],

    superseded_by UUID REFERENCES decisions(decision_id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_decisions_program ON decisions(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_decisions_status ON decisions(decision_status);
CREATE INDEX idx_decisions_date ON decisions(decision_date DESC);
CREATE INDEX idx_decisions_maker ON decisions(decision_maker_id);
CREATE INDEX idx_decisions_artifacts ON decisions USING gin(artifact_ids);
```

### Decision Participants

```sql
CREATE TABLE decision_participants (
    participant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id UUID NOT NULL REFERENCES decisions(decision_id) ON DELETE CASCADE,

    stakeholder_id UUID REFERENCES stakeholders(stakeholder_id),
    user_id UUID REFERENCES users(user_id),

    participation_role VARCHAR(50),

    approval_status VARCHAR(50),
    approval_date TIMESTAMPTZ,
    approval_notes TEXT,

    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT participant_identity_check CHECK (
        stakeholder_id IS NOT NULL OR user_id IS NOT NULL
    )
);

CREATE INDEX idx_decision_participants_decision ON decision_participants(decision_id);
CREATE INDEX idx_decision_participants_stakeholder ON decision_participants(stakeholder_id);
CREATE INDEX idx_decision_participants_user ON decision_participants(user_id);
```

---

## Module 7: Executive Dashboard

### KPI Definitions

```sql
CREATE TABLE kpi_definitions (
    kpi_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    kpi_name VARCHAR(255) NOT NULL,
    kpi_code VARCHAR(50) NOT NULL,
    description TEXT,

    unit_of_measure VARCHAR(50),
    calculation_method TEXT,

    target_value DECIMAL(15,4),
    threshold_red DECIMAL(15,4),
    threshold_yellow DECIMAL(15,4),
    threshold_green DECIMAL(15,4),

    higher_is_better BOOLEAN DEFAULT TRUE,

    measurement_frequency VARCHAR(50),

    dashboard_order INTEGER,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, kpi_code)
);

CREATE INDEX idx_kpi_defs_program ON kpi_definitions(program_id)
    WHERE deleted_at IS NULL AND is_active = TRUE;
CREATE INDEX idx_kpi_defs_order ON kpi_definitions(dashboard_order);
```

### KPI Measurements

```sql
CREATE TABLE kpi_measurements (
    measurement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kpi_id UUID NOT NULL REFERENCES kpi_definitions(kpi_id) ON DELETE CASCADE,

    measurement_date DATE NOT NULL,
    measured_value DECIMAL(15,4) NOT NULL,

    status VARCHAR(50),

    notes TEXT,
    data_source VARCHAR(255),
    artifact_id UUID REFERENCES artifacts(artifact_id),

    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_by UUID REFERENCES users(user_id),

    UNIQUE(kpi_id, measurement_date)
);

CREATE INDEX idx_kpi_measurements_kpi ON kpi_measurements(kpi_id, measurement_date DESC);
CREATE INDEX idx_kpi_measurements_status ON kpi_measurements(status);
```

### Health Scores

```sql
CREATE TABLE health_scores (
    health_score_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    score_date DATE NOT NULL,

    overall_score INTEGER CHECK (overall_score BETWEEN 0 AND 100),
    overall_status VARCHAR(50),

    schedule_score INTEGER CHECK (schedule_score BETWEEN 0 AND 100),
    budget_score INTEGER CHECK (budget_score BETWEEN 0 AND 100),
    risk_score INTEGER CHECK (risk_score BETWEEN 0 AND 100),
    quality_score INTEGER CHECK (quality_score BETWEEN 0 AND 100),

    trend VARCHAR(50),

    ai_summary TEXT,
    key_concerns TEXT[],

    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(program_id, score_date)
);

CREATE INDEX idx_health_scores_program ON health_scores(program_id, score_date DESC);
CREATE INDEX idx_health_scores_status ON health_scores(overall_status);
```

---

## Module 8: Milestone & Phase Management

### Phases

```sql
CREATE TABLE phases (
    phase_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    phase_name VARCHAR(255) NOT NULL,
    phase_number INTEGER NOT NULL,
    description TEXT,

    planned_start_date DATE,
    planned_end_date DATE,
    actual_start_date DATE,
    actual_end_date DATE,

    status VARCHAR(50) DEFAULT 'planned',

    predecessor_phase_id UUID REFERENCES phases(phase_id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, phase_number)
);

CREATE INDEX idx_phases_program ON phases(program_id, phase_number)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_phases_dates ON phases(planned_start_date, planned_end_date);
CREATE INDEX idx_phases_status ON phases(status);
```

### Milestones

```sql
CREATE TABLE milestones (
    milestone_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    phase_id UUID REFERENCES phases(phase_id),

    milestone_name VARCHAR(500) NOT NULL,
    description TEXT,

    milestone_type VARCHAR(50),

    planned_date DATE NOT NULL,
    actual_date DATE,

    status VARCHAR(50) DEFAULT 'pending',

    success_criteria TEXT,
    deliverables TEXT[],

    owner_id UUID REFERENCES users(user_id),

    artifact_ids UUID[],

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_milestones_program ON milestones(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_milestones_phase ON milestones(phase_id);
CREATE INDEX idx_milestones_date ON milestones(planned_date);
CREATE INDEX idx_milestones_status ON milestones(status);
CREATE INDEX idx_milestones_owner ON milestones(owner_id);
```

### Gates

```sql
CREATE TABLE gates (
    gate_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    phase_id UUID REFERENCES phases(phase_id),

    gate_name VARCHAR(255) NOT NULL,
    gate_type VARCHAR(50),

    scheduled_date DATE NOT NULL,
    actual_date DATE,

    entry_criteria TEXT,
    exit_criteria TEXT,

    gate_status VARCHAR(50) DEFAULT 'pending',
    decision_id UUID REFERENCES decisions(decision_id),

    artifact_ids UUID[],

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_gates_program ON gates(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_gates_phase ON gates(phase_id);
CREATE INDEX idx_gates_date ON gates(scheduled_date);
CREATE INDEX idx_gates_status ON gates(gate_status);
```

---

## Module 9: Governance Framework

### Governance Cadences

```sql
CREATE TABLE governance_cadences (
    cadence_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    cadence_name VARCHAR(255) NOT NULL,
    description TEXT,

    frequency VARCHAR(50),
    day_of_week INTEGER CHECK (day_of_week BETWEEN 0 AND 6),
    day_of_month INTEGER CHECK (day_of_month BETWEEN 1 AND 31),

    meeting_duration_minutes INTEGER,

    required_participant_ids UUID[],
    optional_participant_ids UUID[],

    agenda_template TEXT,

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_cadences_program ON governance_cadences(program_id)
    WHERE deleted_at IS NULL AND is_active = TRUE;
```

### Governance Meetings

```sql
CREATE TABLE governance_meetings (
    meeting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cadence_id UUID NOT NULL REFERENCES governance_cadences(cadence_id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    meeting_date DATE NOT NULL,
    actual_start_time TIMESTAMPTZ,
    actual_end_time TIMESTAMPTZ,

    attendee_user_ids UUID[],
    absent_user_ids UUID[],

    artifact_id UUID REFERENCES artifacts(artifact_id),

    status VARCHAR(50) DEFAULT 'scheduled',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_meetings_cadence ON governance_meetings(cadence_id, meeting_date DESC);
CREATE INDEX idx_meetings_program ON governance_meetings(program_id);
CREATE INDEX idx_meetings_status ON governance_meetings(status);
```

### Compliance Requirements

```sql
CREATE TABLE compliance_requirements (
    requirement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    requirement_code VARCHAR(50) NOT NULL,
    requirement_name VARCHAR(255) NOT NULL,
    description TEXT,

    compliance_domain VARCHAR(100),
    requirement_category VARCHAR(100),

    validation_frequency VARCHAR(50),
    next_review_date DATE,

    evidence_requirements TEXT,

    status VARCHAR(50) DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, requirement_code)
);

CREATE INDEX idx_compliance_reqs_program ON compliance_requirements(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_compliance_reqs_domain ON compliance_requirements(compliance_domain);
CREATE INDEX idx_compliance_reqs_review ON compliance_requirements(next_review_date)
    WHERE status = 'active';
```

### Compliance Evidence

```sql
CREATE TABLE compliance_evidence (
    evidence_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id UUID NOT NULL REFERENCES compliance_requirements(requirement_id) ON DELETE CASCADE,

    evidence_date DATE NOT NULL,
    evidence_description TEXT,

    artifact_ids UUID[] NOT NULL,

    validated_by UUID REFERENCES users(user_id),
    validated_at TIMESTAMPTZ,
    validation_status VARCHAR(50),
    validation_notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_compliance_evidence_req ON compliance_evidence(requirement_id, evidence_date DESC);
CREATE INDEX idx_compliance_evidence_status ON compliance_evidence(validation_status);
```

---

## Module 10: Change Control Board

### Change Requests

```sql
CREATE TABLE change_requests (
    change_request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    change_number VARCHAR(50),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    change_category VARCHAR(100),
    change_type VARCHAR(50),
    priority VARCHAR(50),

    business_justification TEXT NOT NULL,
    alternatives_considered TEXT,

    schedule_impact_days INTEGER,
    budget_impact_amount DECIMAL(15,2),
    resource_impact TEXT,
    risk_impact TEXT,

    impacted_milestones UUID[],

    requestor_id UUID NOT NULL REFERENCES users(user_id),
    sponsor_id UUID REFERENCES users(user_id),

    status VARCHAR(50) DEFAULT 'submitted',

    submitted_date DATE DEFAULT CURRENT_DATE,
    review_date DATE,
    approval_date DATE,
    implementation_date DATE,

    decision_id UUID REFERENCES decisions(decision_id),
    approval_notes TEXT,
    rejection_reason TEXT,

    artifact_ids UUID[],

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_change_requests_program ON change_requests(program_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_change_requests_status ON change_requests(status);
CREATE INDEX idx_change_requests_priority ON change_requests(priority, change_type);
CREATE INDEX idx_change_requests_requestor ON change_requests(requestor_id);
CREATE INDEX idx_change_requests_decision ON change_requests(decision_id);
```

### Change Request Approvers

```sql
CREATE TABLE change_request_approvers (
    approver_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_request_id UUID NOT NULL REFERENCES change_requests(change_request_id) ON DELETE CASCADE,

    user_id UUID NOT NULL REFERENCES users(user_id),
    approval_level INTEGER DEFAULT 1,

    approval_status VARCHAR(50) DEFAULT 'pending',
    approval_date TIMESTAMPTZ,
    approval_notes TEXT,

    required BOOLEAN DEFAULT TRUE,

    notified_at TIMESTAMPTZ,

    UNIQUE(change_request_id, user_id)
);

CREATE INDEX idx_cr_approvers_request ON change_request_approvers(change_request_id);
CREATE INDEX idx_cr_approvers_user ON change_request_approvers(user_id, approval_status);
```

---

## Cross-Module: System Tables

### Events (Module Integration)

```sql
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
```

### Audit Logs (Universal)

```sql
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
```

### AI Usage Tracking

```sql
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
```

---

## Entity Relationship Diagram (Simplified)

```
programs (root)
    ├── artifacts ⭐ (CORE HUB)
    │   ├── artifact_topics
    │   ├── artifact_persons → stakeholders
    │   ├── artifact_facts
    │   ├── artifact_insights
    │   └── artifact_embeddings
    │
    ├── invoices → artifacts
    │   └── invoice_line_items → budget_categories
    │
    ├── risks → artifacts[]
    │   └── conversation_threads → conversation_messages
    │
    ├── decisions → artifacts[], risks[]
    │   └── decision_participants → stakeholders
    │
    ├── communications → artifacts[], stakeholders[]
    │   ├── communication_plans
    │   └── communication_templates
    │
    ├── stakeholders
    │   ├── stakeholder_groups
    │   └── communication_enrollments
    │
    ├── milestones → artifacts[]
    │   ├── phases
    │   └── gates → decisions
    │
    ├── change_requests → artifacts[], milestones[]
    │   ├── decision
    │   └── change_request_approvers
    │
    └── compliance_requirements → artifacts[]
        └── compliance_evidence
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-10
**Document Owner:** Technical Lead
**Status:** Approved for Implementation
