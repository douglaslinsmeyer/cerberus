-- ====================================================================
-- MULTI-TENANT ORGANIZATION MIGRATION
-- WARNING: This migration DELETES ALL EXISTING DATA
-- ====================================================================

-- Step 1: Delete all existing data (preserve table structures)
DELETE FROM conversation_messages;
DELETE FROM conversation_threads;
DELETE FROM risk_enrichment_links;
DELETE FROM enrichments;
DELETE FROM risk_mitigations;
DELETE FROM risk_artifact_links;
DELETE FROM risk_suggestions;
DELETE FROM risks;
DELETE FROM financial_variances;
DELETE FROM invoice_line_items;
DELETE FROM invoices;
DELETE FROM budget_categories;
DELETE FROM rate_card_items;
DELETE FROM rate_cards;
DELETE FROM artifact_summaries;
DELETE FROM artifact_embeddings;
DELETE FROM artifact_chunks;
DELETE FROM artifact_insights;
DELETE FROM artifact_facts;
DELETE FROM artifact_persons;
DELETE FROM artifact_topics;
DELETE FROM artifacts;
DELETE FROM ai_usage;
DELETE FROM events;
DELETE FROM audit_logs;
DELETE FROM password_reset_tokens;
DELETE FROM refresh_tokens;
DELETE FROM user_invitations;
DELETE FROM program_users;
DELETE FROM program_stakeholders;
DELETE FROM programs;
DELETE FROM users;

-- Step 2: Create organizations table
CREATE TABLE organizations (
    organization_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_name VARCHAR(255) NOT NULL,
    organization_code VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    plan_tier VARCHAR(50) DEFAULT 'free',
    max_programs INTEGER DEFAULT 10,
    max_users INTEGER DEFAULT 50,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,  -- Nullable for bootstrap
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT org_status_check CHECK (status IN ('active', 'suspended', 'archived')),
    CONSTRAINT org_code_format CHECK (organization_code ~ '^[A-Z0-9-]+$')
);

CREATE INDEX idx_orgs_code ON organizations(organization_code) WHERE deleted_at IS NULL;
CREATE INDEX idx_orgs_status ON organizations(status) WHERE deleted_at IS NULL;

-- Step 3: Create organization_users table
CREATE TABLE organization_users (
    organization_user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(organization_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    org_role VARCHAR(50) NOT NULL CHECK (org_role IN ('owner', 'admin', 'member')),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(user_id),
    revoked_at TIMESTAMPTZ,
    UNIQUE(organization_id, user_id, revoked_at)
);

CREATE INDEX idx_org_users_lookup ON organization_users(organization_id, user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_org_users_user ON organization_users(user_id)
    WHERE revoked_at IS NULL;

-- Step 4: Modify programs table
ALTER TABLE programs
    ADD COLUMN organization_id UUID REFERENCES organizations(organization_id) ON DELETE CASCADE;

CREATE INDEX idx_programs_org ON programs(organization_id) WHERE deleted_at IS NULL;

-- Step 5: Modify refresh_tokens table
ALTER TABLE refresh_tokens
    ADD COLUMN organization_id UUID REFERENCES organizations(organization_id) ON DELETE CASCADE;

CREATE INDEX idx_refresh_tokens_org ON refresh_tokens(organization_id)
    WHERE revoked_at IS NULL;

-- Step 6: Modify user_invitations table
ALTER TABLE user_invitations
    ADD COLUMN organization_id UUID REFERENCES organizations(organization_id) ON DELETE CASCADE,
    ADD COLUMN org_role VARCHAR(50) DEFAULT 'member'
        CHECK (org_role IN ('owner', 'admin', 'member'));

CREATE INDEX idx_invitations_organization ON user_invitations(organization_id);

-- Step 7: Seed bootstrap admin user
INSERT INTO users (
    user_id,
    email,
    full_name,
    password_hash,
    is_active,
    is_admin,
    created_at
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'admin@cerberus.local',
    'System Administrator',
    '$argon2id$v=19$m=65536,t=3,p=4$j5FG0wgeiF+t3EpX4UlcqA$281EGRP3md/V6WdKRIk+Y0O+PNLWpYVNWi1qWGgGeY0',
    TRUE,
    TRUE,
    NOW()
);

-- Step 8: Seed default organization
INSERT INTO organizations (
    organization_id,
    organization_name,
    organization_code,
    status,
    created_at
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Demo Organization',
    'DEMO',
    'active',
    NOW()
);

-- Step 9: Link admin to organization as owner
INSERT INTO organization_users (
    organization_id,
    user_id,
    org_role,
    granted_at
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'owner',
    NOW()
);

-- Step 10: Create demo program
INSERT INTO programs (
    program_id,
    organization_id,
    program_name,
    program_code,
    description,
    status,
    internal_organization,
    created_at,
    created_by
) VALUES (
    '00000000-0000-0000-0000-000000000002'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Demo Program',
    'DEMO-001',
    'Default program for demonstration',
    'active',
    'Demo Corp',
    NOW(),
    '00000000-0000-0000-0000-000000000001'::uuid
);

-- Step 11: Grant admin program access
INSERT INTO program_users (
    program_id,
    user_id,
    role,
    granted_at,
    granted_by
) VALUES (
    '00000000-0000-0000-0000-000000000002'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'admin',
    NOW(),
    '00000000-0000-0000-0000-000000000001'::uuid
);

-- Step 12: Add triggers
CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Step 13: Add comments
COMMENT ON TABLE organizations IS 'Top-level tenant entity. Users belong to one organization.';
COMMENT ON TABLE organization_users IS 'User membership in organizations with org-level roles.';
COMMENT ON COLUMN organization_users.org_role IS 'owner: full control, admin: manage users/programs, member: regular access';
COMMENT ON COLUMN programs.organization_id IS 'Organization that owns this program.';

-- Migration complete
DO $$
BEGIN
    RAISE NOTICE 'Multi-tenant migration complete!';
    RAISE NOTICE 'Bootstrap Credentials:';
    RAISE NOTICE '  Email: admin@cerberus.local';
    RAISE NOTICE '  Password: Admin123!';
    RAISE NOTICE '  Organization: Demo Organization (DEMO)';
    RAISE NOTICE '  Program: Demo Program (DEMO-001)';
END $$;
