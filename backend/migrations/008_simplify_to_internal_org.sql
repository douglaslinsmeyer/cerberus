-- Simplify program configuration to just internal organization name
-- This is all AI needs to classify internal vs external people

-- Add simple internal_organization field
ALTER TABLE programs ADD COLUMN IF NOT EXISTS internal_organization VARCHAR(255);

-- Seed from existing configuration or program name
UPDATE programs
SET internal_organization = COALESCE(
    configuration->'company'->>'name',
    program_name
)
WHERE internal_organization IS NULL;

-- Set default for future programs
ALTER TABLE programs ALTER COLUMN internal_organization SET DEFAULT 'Organization';

-- Comment explaining the field
COMMENT ON COLUMN programs.internal_organization IS 'The internal company/organization name. AI uses this to classify people as internal (if their organization matches) or external (otherwise).';

-- Note: We keep the configuration JSONB column for potential future use,
-- but the system no longer relies on complex nested structures.
-- Just internal_organization is sufficient for AI classification.
