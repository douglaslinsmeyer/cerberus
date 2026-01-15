-- Person Merge Groups Migration
-- Adds tables for grouping and merging AI-detected person mentions across documents

-- Person merge groups table
-- Tracks groups of similar person mentions that can be merged into single stakeholder
CREATE TABLE IF NOT EXISTS person_merge_groups (
    group_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,

    -- Grouping metadata
    suggested_name VARCHAR(255) NOT NULL, -- Most common/confident name variant
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, confirmed, rejected, merged

    -- Conflict tracking
    has_role_conflicts BOOLEAN DEFAULT FALSE,
    has_org_conflicts BOOLEAN DEFAULT FALSE,

    -- Selected values after user review
    resolved_name VARCHAR(255),
    resolved_role VARCHAR(255),
    resolved_organization VARCHAR(255),

    -- Merge result
    merged_stakeholder_id UUID REFERENCES program_stakeholders(stakeholder_id) ON DELETE SET NULL,
    merged_at TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_status CHECK (status IN ('pending', 'confirmed', 'rejected', 'merged'))
);

-- Indexes for person_merge_groups
CREATE INDEX IF NOT EXISTS idx_merge_groups_program ON person_merge_groups(program_id);
CREATE INDEX IF NOT EXISTS idx_merge_groups_status ON person_merge_groups(status, program_id);
CREATE INDEX IF NOT EXISTS idx_merge_groups_merged_stakeholder ON person_merge_groups(merged_stakeholder_id) WHERE merged_stakeholder_id IS NOT NULL;

-- Person merge group members table
-- Links individual person_ids to merge groups
CREATE TABLE IF NOT EXISTS person_merge_group_members (
    member_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES person_merge_groups(group_id) ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES artifact_persons(person_id) ON DELETE CASCADE,

    -- Why this person was added to group
    similarity_score DECIMAL(5,4) NOT NULL, -- 0.0000 to 1.0000
    matching_method VARCHAR(50), -- exact_name, fuzzy_name, same_org, manual

    -- Metadata
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(group_id, person_id),
    CONSTRAINT valid_similarity CHECK (similarity_score >= 0 AND similarity_score <= 1),
    CONSTRAINT valid_matching_method CHECK (matching_method IN ('exact_name', 'fuzzy_name', 'same_org', 'manual'))
);

-- Indexes for person_merge_group_members
CREATE INDEX IF NOT EXISTS idx_merge_members_group ON person_merge_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_merge_members_person ON person_merge_group_members(person_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_merge_group_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER trigger_update_merge_group_updated_at
    BEFORE UPDATE ON person_merge_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_merge_group_updated_at();

-- Add comment for documentation
COMMENT ON TABLE person_merge_groups IS 'Groups of similar person mentions that can be merged into a single stakeholder';
COMMENT ON TABLE person_merge_group_members IS 'Links individual person_ids to merge groups';
COMMENT ON COLUMN person_merge_groups.suggested_name IS 'Most common or highest confidence name variant in the group';
COMMENT ON COLUMN person_merge_groups.status IS 'pending=awaiting review, confirmed=user reviewed, rejected=not same person, merged=stakeholder created';
COMMENT ON COLUMN person_merge_group_members.similarity_score IS 'pg_trgm similarity score between this person and the group anchor';
COMMENT ON COLUMN person_merge_group_members.matching_method IS 'How this person was matched to the group';
