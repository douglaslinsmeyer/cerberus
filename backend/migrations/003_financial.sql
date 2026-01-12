-- Financial Module Migration
-- Creates tables for rate cards, invoices, budget tracking, and variance detection

-- Rate cards (approved billing rates)
CREATE TABLE rate_cards (
    rate_card_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,
    effective_start_date DATE NOT NULL,
    effective_end_date DATE,
    currency VARCHAR(3) DEFAULT 'USD',
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(user_id),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT rate_cards_dates_check CHECK (effective_end_date IS NULL OR effective_end_date >= effective_start_date)
);

CREATE INDEX idx_rate_cards_program ON rate_cards(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_rate_cards_active ON rate_cards(is_active, effective_start_date);

-- Rate card items (individual rates per person/role)
CREATE TABLE rate_card_items (
    item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_card_id UUID NOT NULL REFERENCES rate_cards(rate_card_id) ON DELETE CASCADE,

    -- Person or role identification
    person_name VARCHAR(255),
    role_title VARCHAR(255),
    seniority_level VARCHAR(50), -- junior, mid, senior, principal

    -- Rate information
    rate_type VARCHAR(50) NOT NULL, -- hourly, daily, fixed, monthly
    rate_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Allocation expectations
    expected_hours_per_week DECIMAL(5,2),
    expected_hours_per_month DECIMAL(6,2),

    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT rate_card_items_amount_check CHECK (rate_amount >= 0)
);

CREATE INDEX idx_rate_card_items_card ON rate_card_items(rate_card_id);
CREATE INDEX idx_rate_card_items_person ON rate_card_items(person_name);
CREATE INDEX idx_rate_card_items_role ON rate_card_items(role_title, seniority_level);

-- Invoices (links to artifacts)
CREATE TABLE invoices (
    invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    artifact_id UUID REFERENCES artifacts(artifact_id) ON DELETE SET NULL,

    -- Invoice header
    invoice_number VARCHAR(100),
    vendor_name VARCHAR(255) NOT NULL,
    vendor_id VARCHAR(100),
    invoice_date DATE NOT NULL,
    due_date DATE,
    period_start_date DATE,
    period_end_date DATE,

    -- Amounts
    subtotal_amount DECIMAL(15,2),
    tax_amount DECIMAL(15,2),
    total_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Processing
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, validated, approved, rejected
    payment_status VARCHAR(50) DEFAULT 'unpaid', -- unpaid, paid, partial, overdue

    -- AI analysis
    ai_model_version VARCHAR(50),
    ai_confidence_score DECIMAL(5,4),
    ai_processing_time_ms INTEGER,

    -- Workflow
    submitted_by UUID REFERENCES users(user_id),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by UUID REFERENCES users(user_id),
    approved_at TIMESTAMPTZ,
    rejected_reason TEXT,

    deleted_at TIMESTAMPTZ,

    CONSTRAINT invoices_amounts_check CHECK (total_amount >= 0)
);

CREATE INDEX idx_invoices_program ON invoices(program_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_artifact ON invoices(artifact_id);
CREATE INDEX idx_invoices_vendor ON invoices(vendor_name);
CREATE INDEX idx_invoices_status ON invoices(processing_status, payment_status);
CREATE INDEX idx_invoices_date ON invoices(invoice_date DESC);

-- Invoice line items
CREATE TABLE invoice_line_items (
    line_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(invoice_id) ON DELETE CASCADE,

    line_number INTEGER NOT NULL,
    description TEXT NOT NULL,

    -- Extracted amounts
    quantity DECIMAL(15,4),
    unit_rate DECIMAL(15,4),
    line_amount DECIMAL(15,2) NOT NULL,

    -- Identified person/role (from AI extraction)
    person_name VARCHAR(255),
    role_description VARCHAR(255),

    -- Rate card reference (if matched)
    matched_rate_card_item_id UUID REFERENCES rate_card_items(item_id),
    expected_rate DECIMAL(15,4),
    rate_variance_amount DECIMAL(15,2),
    rate_variance_percentage DECIMAL(6,2),

    -- Hours validation
    billed_hours DECIMAL(10,2),
    expected_hours DECIMAL(10,2),
    hours_variance DECIMAL(10,2),

    -- Categorization
    spend_category VARCHAR(100), -- labor, materials, software, travel, other
    budget_category_id UUID,

    -- Flags
    has_variance BOOLEAN DEFAULT FALSE,
    variance_severity VARCHAR(50), -- low, medium, high, critical
    needs_review BOOLEAN DEFAULT FALSE,
    review_notes TEXT,

    ai_confidence_score DECIMAL(5,4),

    CONSTRAINT line_items_amount_check CHECK (line_amount >= 0)
);

CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_person ON invoice_line_items(person_name);
CREATE INDEX idx_line_items_category ON invoice_line_items(spend_category);
CREATE INDEX idx_line_items_variance ON invoice_line_items(has_variance, variance_severity);
CREATE INDEX idx_line_items_review ON invoice_line_items(needs_review);

-- Budget categories
CREATE TABLE budget_categories (
    category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    category_name VARCHAR(255) NOT NULL,
    description TEXT,
    budgeted_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Time period
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER CHECK (fiscal_quarter BETWEEN 1 AND 4),

    -- Tracking
    actual_spend DECIMAL(15,2) DEFAULT 0,
    committed_spend DECIMAL(15,2) DEFAULT 0,
    variance_amount DECIMAL(15,2) DEFAULT 0,
    variance_percentage DECIMAL(6,2) DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(program_id, category_name, fiscal_year, fiscal_quarter)
);

CREATE INDEX idx_budget_categories_program ON budget_categories(program_id);
CREATE INDEX idx_budget_categories_period ON budget_categories(fiscal_year, fiscal_quarter);
CREATE INDEX idx_budget_categories_variance ON budget_categories(variance_percentage DESC);

-- Financial variances (AI-detected discrepancies)
CREATE TABLE financial_variances (
    variance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES invoices(invoice_id) ON DELETE CASCADE,
    line_item_id UUID REFERENCES invoice_line_items(line_item_id) ON DELETE CASCADE,

    variance_type VARCHAR(100) NOT NULL, -- rate_overage, hours_overage, cross_document_conflict, budget_exceeded
    severity VARCHAR(50) NOT NULL, -- low, medium, high, critical
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,

    -- Variance details
    expected_value DECIMAL(15,2),
    actual_value DECIMAL(15,2),
    variance_amount DECIMAL(15,2),
    variance_percentage DECIMAL(6,2),

    -- Evidence
    source_artifact_ids UUID[], -- Artifacts that support this variance finding
    conflicting_values JSONB, -- Store conflicting values from different sources

    -- AI metadata
    ai_confidence_score DECIMAL(5,4),
    ai_detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- User actions
    is_dismissed BOOLEAN DEFAULT FALSE,
    dismissed_by UUID REFERENCES users(user_id),
    dismissed_at TIMESTAMPTZ,
    dismissal_reason TEXT,

    resolution_notes TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(user_id),

    CONSTRAINT variance_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical'))
);

CREATE INDEX idx_variances_program ON financial_variances(program_id);
CREATE INDEX idx_variances_invoice ON financial_variances(invoice_id);
CREATE INDEX idx_variances_severity ON financial_variances(severity, is_dismissed);
CREATE INDEX idx_variances_type ON financial_variances(variance_type);
CREATE INDEX idx_variances_artifacts ON financial_variances USING gin(source_artifact_ids);

-- Trigger for budget category updates
CREATE OR REPLACE FUNCTION update_budget_category_variance()
RETURNS TRIGGER AS $$
BEGIN
    NEW.variance_amount = NEW.actual_spend - NEW.budgeted_amount;
    NEW.variance_percentage = CASE
        WHEN NEW.budgeted_amount > 0 THEN
            (NEW.variance_amount / NEW.budgeted_amount) * 100
        ELSE 0
    END;
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_budget_variance_calculation
    BEFORE UPDATE OF actual_spend, budgeted_amount ON budget_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_budget_category_variance();

-- Trigger for rate cards updated_at
CREATE TRIGGER update_rate_cards_updated_at BEFORE UPDATE ON rate_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
