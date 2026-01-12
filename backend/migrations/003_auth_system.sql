-- Auth System Migration
-- Creates tables for invitations, refresh tokens, and password resets
-- Adds columns to users table for authentication features

-- User invitations table
CREATE TABLE user_invitations (
    invitation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'contributor', 'viewer')),
    is_admin BOOLEAN DEFAULT FALSE,
    invited_by UUID NOT NULL REFERENCES users(user_id),
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    user_id UUID REFERENCES users(user_id),
    token_hash VARCHAR(64) UNIQUE NOT NULL,

    CONSTRAINT invitations_email_program_unique UNIQUE(email, program_id, accepted_at)
);

CREATE INDEX idx_invitations_email ON user_invitations(email) WHERE accepted_at IS NULL;
CREATE INDEX idx_invitations_token ON user_invitations(token_hash);
CREATE INDEX idx_invitations_program ON user_invitations(program_id);
CREATE INDEX idx_invitations_expires ON user_invitations(expires_at) WHERE accepted_at IS NULL;

COMMENT ON TABLE user_invitations IS 'Tracks user invitations to programs';
COMMENT ON COLUMN user_invitations.token_hash IS 'SHA-256 hash of the invitation token';
COMMENT ON COLUMN user_invitations.is_admin IS 'Whether to grant global admin privileges upon acceptance';

-- Refresh tokens table (for JWT refresh token rotation)
CREATE TABLE refresh_tokens (
    token_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_expiry ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_token_id ON refresh_tokens(token_id) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'Tracks active refresh tokens for token rotation';
COMMENT ON COLUMN refresh_tokens.token_id IS 'Unique ID embedded in the JWT refresh token';
COMMENT ON COLUMN refresh_tokens.revoked_at IS 'When token was revoked (for logout or security)';

-- Password reset tokens table
CREATE TABLE password_reset_tokens (
    token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_password_reset_user ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_token ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_expiry ON password_reset_tokens(expires_at);

COMMENT ON TABLE password_reset_tokens IS 'Temporary tokens for password reset flow';
COMMENT ON COLUMN password_reset_tokens.token_hash IS 'SHA-256 hash of the reset token';

-- Add authentication-related columns to users table
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS failed_login_attempts INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_program_accessed UUID REFERENCES programs(program_id);

COMMENT ON COLUMN users.failed_login_attempts IS 'Counter for failed login attempts (resets on success)';
COMMENT ON COLUMN users.locked_until IS 'Account locked until this timestamp (for brute force protection)';
COMMENT ON COLUMN users.last_program_accessed IS 'Most recently accessed program (for token refresh)';

-- Create index for locked accounts
CREATE INDEX IF NOT EXISTS idx_users_locked ON users(locked_until) WHERE locked_until IS NOT NULL;

-- Function to automatically clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    -- Delete expired password reset tokens
    DELETE FROM password_reset_tokens
    WHERE expires_at < NOW();

    -- Delete expired refresh tokens
    DELETE FROM refresh_tokens
    WHERE expires_at < NOW();

    -- Delete old revoked refresh tokens (older than 30 days)
    DELETE FROM refresh_tokens
    WHERE revoked_at IS NOT NULL
    AND revoked_at < NOW() - INTERVAL '30 days';

    -- Delete old expired invitations (older than 30 days)
    DELETE FROM user_invitations
    WHERE expires_at < NOW()
    AND accepted_at IS NULL
    AND expires_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_tokens() IS 'Cleanup function for expired auth tokens (should be run periodically)';

-- Note: In production, schedule this function to run periodically
-- Example using pg_cron extension:
-- SELECT cron.schedule('cleanup-tokens', '0 2 * * *', 'SELECT cleanup_expired_tokens()');
