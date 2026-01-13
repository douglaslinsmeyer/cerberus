-- Fix artifacts unique constraint to allow re-upload of soft-deleted artifacts
-- The current constraint blocks duplicates even when original is deleted

-- Drop the old constraint
ALTER TABLE artifacts DROP CONSTRAINT IF EXISTS artifacts_content_hash_program_unique;

-- Create a partial unique index that only applies to non-deleted artifacts
CREATE UNIQUE INDEX artifacts_content_hash_program_active_unique
    ON artifacts(program_id, content_hash)
    WHERE deleted_at IS NULL;

-- This allows:
-- 1. Same content to be re-uploaded if original was soft-deleted
-- 2. Smart deduplication logic to work properly
-- 3. Multiple versions with same content (as long as old ones are deleted)
