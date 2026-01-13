-- Program Configuration Migration
-- Adds program-level configuration for company info, taxonomy, and stakeholder management

-- Extend programs table with JSONB configuration
ALTER TABLE programs ADD COLUMN IF NOT EXISTS configuration JSONB DEFAULT '{}';

-- Configuration stores:
-- {
--   "company": {
--     "name": "PING, Inc.",
--     "full_legal_name": "PING Incorporated",
--     "aliases": ["PING", "Ping Identity"]
--   },
--   "taxonomy": {
--     "risk_categories": ["technical", "financial", "vendor", "security"],
--     "spend_categories": ["software", "labor", "infrastructure", "consulting"],
--     "project_phases": ["planning", "design", "implementation", "testing"]
--   },
--   "vendors": [
--     {"name": "Infor (US), LLC", "type": "software_vendor"},
--     {"name": "TechCorp Solutions", "type": "consulting"}
--   ]
-- }

-- Index for JSONB queries
CREATE INDEX IF NOT EXISTS idx_programs_config ON programs USING gin(configuration);

-- Program stakeholders table
CREATE TABLE IF NOT EXISTS program_stakeholders (
    stakeholder_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Person identification
    person_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    role VARCHAR(255),
    organization VARCHAR(255),

    -- Classification
    stakeholder_type VARCHAR(50) NOT NULL, -- internal, external, vendor, partner, customer
    is_internal BOOLEAN DEFAULT FALSE,

    -- Engagement
    engagement_level VARCHAR(50), -- key, primary, secondary, observer
    department VARCHAR(100),

    -- Metadata
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT program_stakeholders_unique UNIQUE(program_id, person_name, deleted_at)
);

CREATE INDEX IF NOT EXISTS idx_stakeholders_program ON program_stakeholders(program_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_stakeholders_type ON program_stakeholders(stakeholder_type, is_internal);
CREATE INDEX IF NOT EXISTS idx_stakeholders_name ON program_stakeholders(person_name);

-- Trigger for stakeholders updated_at
DROP TRIGGER IF EXISTS update_stakeholders_updated_at ON program_stakeholders;
CREATE TRIGGER update_stakeholders_updated_at BEFORE UPDATE ON program_stakeholders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed default program configuration
UPDATE programs
SET configuration = jsonb_build_object(
    'company', jsonb_build_object(
        'name', program_name,
        'full_legal_name', program_name,
        'aliases', jsonb_build_array()
    ),
    'taxonomy', jsonb_build_object(
        'risk_categories', jsonb_build_array('technical', 'financial', 'schedule', 'resource', 'external'),
        'spend_categories', jsonb_build_array('labor', 'materials', 'software', 'travel', 'other'),
        'project_phases', jsonb_build_array()
    ),
    'vendors', jsonb_build_array()
)
WHERE configuration = '{}'::jsonb OR configuration IS NULL;
