-- Seed Data Migration
-- Creates default system user and program for initial setup
-- Uses specific UUIDs that are hardcoded in the application

-- Default system user
-- UUID: 00000000-0000-0000-0000-000000000001
-- This is hardcoded in backend/internal/modules/artifacts/handlers.go:63
INSERT INTO users (
    user_id,
    email,
    full_name,
    is_active,
    is_admin,
    created_at,
    auth_provider
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'system@cerberus.local',
    'System Administrator',
    true,
    true,
    NOW(),
    'system'
)
ON CONFLICT (user_id) DO NOTHING;

-- Default program
-- UUID: 00000000-0000-0000-0000-000000000001
-- This is hardcoded in frontend/src/App.tsx:22
INSERT INTO programs (
    program_id,
    program_name,
    program_code,
    description,
    status,
    created_at,
    created_by,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Cerberus Demo Program',
    'DEMO-001',
    'Default program for Cerberus system demonstration and initial setup. This program is used for testing artifact uploads and AI analysis features.',
    'active',
    NOW(),
    '00000000-0000-0000-0000-000000000001'::uuid,
    NOW()
)
ON CONFLICT (program_id) DO NOTHING;

-- Grant system user admin access to the default program
INSERT INTO program_users (
    program_id,
    user_id,
    role,
    granted_at,
    granted_by
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'admin',
    NOW(),
    '00000000-0000-0000-0000-000000000001'::uuid
)
ON CONFLICT (program_id, user_id, revoked_at) DO NOTHING;

-- Log that seed data has been created
-- This helps with auditing and troubleshooting
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM users
        WHERE user_id = '00000000-0000-0000-0000-000000000001'::uuid
    ) AND EXISTS (
        SELECT 1 FROM programs
        WHERE program_id = '00000000-0000-0000-0000-000000000001'::uuid
    ) THEN
        RAISE NOTICE 'Seed data verified: System user and default program are ready';
    END IF;
END $$;
