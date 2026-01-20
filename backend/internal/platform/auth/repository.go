package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// Repository handles database operations for authentication
type Repository struct {
	db *db.DB
}

// NewRepository creates a new auth repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ==================== User Operations ====================

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT user_id, email, full_name, password_hash, auth_provider, auth_provider_id,
		       is_active, is_admin, created_at, last_login_at, deleted_at,
		       failed_login_attempts, locked_until, last_program_accessed
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.UserID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.AuthProvider,
		&user.AuthProviderID,
		&user.IsActive,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.LastLoginAt,
		&user.DeletedAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastProgramAccessed,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	query := `
		SELECT user_id, email, full_name, password_hash, auth_provider, auth_provider_id,
		       is_active, is_admin, created_at, last_login_at, deleted_at,
		       failed_login_attempts, locked_until, last_program_accessed
		FROM users
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.AuthProvider,
		&user.AuthProviderID,
		&user.IsActive,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.LastLoginAt,
		&user.DeletedAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastProgramAccessed,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByOAuthProvider retrieves a user by OAuth provider and provider ID
func (r *Repository) GetUserByOAuthProvider(ctx context.Context, provider, providerID string) (*User, error) {
	query := `
		SELECT user_id, email, full_name, password_hash, auth_provider, auth_provider_id,
		       is_active, is_admin, created_at, last_login_at, deleted_at,
		       failed_login_attempts, locked_until, last_program_accessed
		FROM users
		WHERE auth_provider = $1 AND auth_provider_id = $2 AND deleted_at IS NULL
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, provider, providerID).Scan(
		&user.UserID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.AuthProvider,
		&user.AuthProviderID,
		&user.IsActive,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.LastLoginAt,
		&user.DeletedAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastProgramAccessed,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (user_id, email, full_name, password_hash, auth_provider,
		                   auth_provider_id, is_active, is_admin, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.UserID,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.AuthProvider,
		user.AuthProviderID,
		user.IsActive,
		user.IsAdmin,
		user.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password hash
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1
		WHERE user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_login_at = NOW()
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// IncrementFailedAttempts increments the failed login attempts counter
func (r *Repository) IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes'
		        ELSE NULL
		    END
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment failed attempts: %w", err)
	}

	return nil
}

// ResetFailedAttempts resets the failed login attempts counter
func (r *Repository) ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until = NULL
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to reset failed attempts: %w", err)
	}

	return nil
}

// LinkOAuthProvider links an OAuth provider to an existing user
func (r *Repository) LinkOAuthProvider(ctx context.Context, userID uuid.UUID, provider, providerID string) error {
	query := `
		UPDATE users
		SET auth_provider = $1,
		    auth_provider_id = $2
		WHERE user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, provider, providerID, userID)
	if err != nil {
		return fmt.Errorf("failed to link OAuth provider: %w", err)
	}

	return nil
}

// UpdateLastProgramAccessed updates the last accessed program
func (r *Repository) UpdateLastProgramAccessed(ctx context.Context, userID, programID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_program_accessed = $1
		WHERE user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, programID, userID)
	if err != nil {
		return fmt.Errorf("failed to update last program accessed: %w", err)
	}

	return nil
}

// ==================== Organization Operations ====================

// GetOrganizationByID retrieves an organization by ID
func (r *Repository) GetOrganizationByID(ctx context.Context, organizationID uuid.UUID) (*Organization, error) {
	query := `
		SELECT organization_id, organization_name, organization_code, status, settings,
		       plan_tier, max_programs, max_users, created_at, created_by, updated_at, deleted_at
		FROM organizations
		WHERE organization_id = $1 AND deleted_at IS NULL
	`

	var org Organization
	err := r.db.QueryRowContext(ctx, query, organizationID).Scan(
		&org.OrganizationID,
		&org.OrganizationName,
		&org.OrganizationCode,
		&org.Status,
		&org.Settings,
		&org.PlanTier,
		&org.MaxPrograms,
		&org.MaxUsers,
		&org.CreatedAt,
		&org.CreatedBy,
		&org.UpdatedAt,
		&org.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// GetUserOrganization retrieves the organization a user belongs to
func (r *Repository) GetUserOrganization(ctx context.Context, userID uuid.UUID) (*Organization, error) {
	query := `
		SELECT o.organization_id, o.organization_name, o.organization_code, o.status, o.settings,
		       o.plan_tier, o.max_programs, o.max_users, o.created_at, o.created_by, o.updated_at, o.deleted_at
		FROM organizations o
		JOIN organization_users ou ON o.organization_id = ou.organization_id
		WHERE ou.user_id = $1 AND ou.revoked_at IS NULL AND o.deleted_at IS NULL
		LIMIT 1
	`

	var org Organization
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&org.OrganizationID,
		&org.OrganizationName,
		&org.OrganizationCode,
		&org.Status,
		&org.Settings,
		&org.PlanTier,
		&org.MaxPrograms,
		&org.MaxUsers,
		&org.CreatedAt,
		&org.CreatedBy,
		&org.UpdatedAt,
		&org.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not linked to organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user organization: %w", err)
	}

	return &org, nil
}

// GetOrganizationUser retrieves a user's membership in an organization
func (r *Repository) GetOrganizationUser(ctx context.Context, organizationID, userID uuid.UUID) (*OrganizationUser, error) {
	query := `
		SELECT organization_user_id, organization_id, user_id, org_role,
		       granted_at, granted_by, revoked_at
		FROM organization_users
		WHERE organization_id = $1 AND user_id = $2 AND revoked_at IS NULL
	`

	var orgUser OrganizationUser
	err := r.db.QueryRowContext(ctx, query, organizationID, userID).Scan(
		&orgUser.OrganizationUserID,
		&orgUser.OrganizationID,
		&orgUser.UserID,
		&orgUser.OrgRole,
		&orgUser.GrantedAt,
		&orgUser.GrantedBy,
		&orgUser.RevokedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization membership not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization user: %w", err)
	}

	return &orgUser, nil
}

// CreateOrganization creates a new organization
func (r *Repository) CreateOrganization(ctx context.Context, org *Organization) error {
	query := `
		INSERT INTO organizations (organization_id, organization_name, organization_code, status,
		                           settings, plan_tier, max_programs, max_users, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		org.OrganizationID,
		org.OrganizationName,
		org.OrganizationCode,
		org.Status,
		org.Settings,
		org.PlanTier,
		org.MaxPrograms,
		org.MaxUsers,
		org.CreatedAt,
		org.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

// CreateOrganizationUser links a user to an organization
func (r *Repository) CreateOrganizationUser(ctx context.Context, orgUser *OrganizationUser) error {
	query := `
		INSERT INTO organization_users (organization_user_id, organization_id, user_id, org_role,
		                                 granted_at, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		orgUser.OrganizationUserID,
		orgUser.OrganizationID,
		orgUser.UserID,
		orgUser.OrgRole,
		orgUser.GrantedAt,
		orgUser.GrantedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create organization user: %w", err)
	}

	return nil
}

// Program represents minimal program info for auth checks
type Program struct {
	ProgramID      uuid.UUID
	ProgramName    string
	OrganizationID uuid.UUID
}

// ==================== Program Access Operations ====================

// GetProgram retrieves basic program information (for auth checks)
func (r *Repository) GetProgram(ctx context.Context, programID uuid.UUID) (*Program, error) {
	query := `
		SELECT program_id, program_name, organization_id
		FROM programs
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	var prog Program
	err := r.db.QueryRowContext(ctx, query, programID).Scan(
		&prog.ProgramID,
		&prog.ProgramName,
		&prog.OrganizationID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("program not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	return &prog, nil
}

// GetProgramUser retrieves a user's access to a specific program
func (r *Repository) GetProgramUser(ctx context.Context, programID, userID uuid.UUID) (*ProgramUser, error) {
	query := `
		SELECT program_user_id, program_id, user_id, role, granted_at, granted_by, revoked_at
		FROM program_users
		WHERE program_id = $1 AND user_id = $2 AND revoked_at IS NULL
		ORDER BY granted_at DESC
		LIMIT 1
	`

	var pu ProgramUser
	err := r.db.QueryRowContext(ctx, query, programID, userID).Scan(
		&pu.ProgramUserID,
		&pu.ProgramID,
		&pu.UserID,
		&pu.Role,
		&pu.GrantedAt,
		&pu.GrantedBy,
		&pu.RevokedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("program access not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get program user: %w", err)
	}

	return &pu, nil
}

// GetUserPrograms retrieves all programs a user has access to within an organization
func (r *Repository) GetUserPrograms(ctx context.Context, userID, organizationID uuid.UUID) ([]ProgramAccess, error) {
	query := `
		SELECT p.program_id, p.program_name, p.program_code, pu.role, pu.granted_at
		FROM program_users pu
		JOIN programs p ON pu.program_id = p.program_id
		WHERE pu.user_id = $1 AND p.organization_id = $2
		      AND pu.revoked_at IS NULL AND p.deleted_at IS NULL
		ORDER BY pu.granted_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user programs: %w", err)
	}
	defer rows.Close()

	programs := make([]ProgramAccess, 0)
	for rows.Next() {
		var pa ProgramAccess
		err := rows.Scan(
			&pa.ProgramID,
			&pa.ProgramName,
			&pa.ProgramCode,
			&pa.Role,
			&pa.GrantedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan program access: %w", err)
		}
		programs = append(programs, pa)
	}

	return programs, nil
}

// CreateProgramUser grants a user access to a program
func (r *Repository) CreateProgramUser(ctx context.Context, pu ProgramUser) error {
	query := `
		INSERT INTO program_users (program_user_id, program_id, user_id, role, granted_at, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		pu.ProgramUserID,
		pu.ProgramID,
		pu.UserID,
		pu.Role,
		pu.GrantedAt,
		pu.GrantedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create program user: %w", err)
	}

	return nil
}

// UpdateProgramUserRole updates a user's role in a program
func (r *Repository) UpdateProgramUserRole(ctx context.Context, programID, userID uuid.UUID, role string) error {
	query := `
		UPDATE program_users
		SET role = $1
		WHERE program_id = $2 AND user_id = $3 AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, role, programID, userID)
	if err != nil {
		return fmt.Errorf("failed to update program user role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("program user not found or already revoked")
	}

	return nil
}

// RevokeProgramAccess revokes a user's access to a program
func (r *Repository) RevokeProgramAccess(ctx context.Context, programID, userID uuid.UUID) error {
	query := `
		UPDATE program_users
		SET revoked_at = NOW()
		WHERE program_id = $1 AND user_id = $2 AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, programID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke program access: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("program access not found or already revoked")
	}

	return nil
}

// ==================== Refresh Token Operations ====================

// StoreRefreshToken stores a new refresh token
func (r *Repository) StoreRefreshToken(ctx context.Context, token RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (token_id, user_id, organization_id, program_id, issued_at, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.TokenID,
		token.UserID,
		token.OrganizationID,
		token.ProgramID,
		token.IssuedAt,
		token.ExpiresAt,
		token.IPAddress,
		token.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by ID
func (r *Repository) GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (*RefreshToken, error) {
	query := `
		SELECT token_id, user_id, organization_id, program_id, issued_at, expires_at, revoked_at,
		       last_used_at, ip_address, user_agent
		FROM refresh_tokens
		WHERE token_id = $1
	`

	var token RefreshToken
	err := r.db.QueryRowContext(ctx, query, tokenID).Scan(
		&token.TokenID,
		&token.UserID,
		&token.OrganizationID,
		&token.ProgramID,
		&token.IssuedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.LastUsedAt,
		&token.IPAddress,
		&token.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllRefreshTokens revokes all refresh tokens for a user
func (r *Repository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
	}

	return nil
}

// UpdateRefreshTokenLastUsed updates the last used timestamp
func (r *Repository) UpdateRefreshTokenLastUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET last_used_at = NOW()
		WHERE token_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to update refresh token last used: %w", err)
	}

	return nil
}

// ==================== Invitation Operations ====================

// CreateInvitation creates a new invitation
func (r *Repository) CreateInvitation(ctx context.Context, invitation *Invitation, tokenHash string) error {
	query := `
		INSERT INTO user_invitations (invitation_id, email, program_id, role, is_admin,
		                              invited_by, invited_at, expires_at, token_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		invitation.InvitationID,
		invitation.Email,
		invitation.ProgramID,
		invitation.Role,
		invitation.IsAdmin,
		invitation.InvitedBy,
		invitation.InvitedAt,
		invitation.ExpiresAt,
		tokenHash,
	)

	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	return nil
}

// GetInvitationByToken retrieves an invitation by token hash
func (r *Repository) GetInvitationByToken(ctx context.Context, tokenHash string) (*Invitation, error) {
	query := `
		SELECT invitation_id, email, program_id, role, is_admin, invited_by,
		       invited_at, expires_at, accepted_at, user_id
		FROM user_invitations
		WHERE token_hash = $1
	`

	var inv Invitation
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&inv.InvitationID,
		&inv.Email,
		&inv.ProgramID,
		&inv.Role,
		&inv.IsAdmin,
		&inv.InvitedBy,
		&inv.InvitedAt,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.UserID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return &inv, nil
}

// GetPendingInvitation retrieves a pending invitation by email
func (r *Repository) GetPendingInvitation(ctx context.Context, email string) (*Invitation, error) {
	query := `
		SELECT invitation_id, email, program_id, role, is_admin, invited_by,
		       invited_at, expires_at, accepted_at, user_id
		FROM user_invitations
		WHERE email = $1 AND accepted_at IS NULL AND expires_at > NOW()
		ORDER BY invited_at DESC
		LIMIT 1
	`

	var inv Invitation
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&inv.InvitationID,
		&inv.Email,
		&inv.ProgramID,
		&inv.Role,
		&inv.IsAdmin,
		&inv.InvitedBy,
		&inv.InvitedAt,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.UserID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pending invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pending invitation: %w", err)
	}

	return &inv, nil
}

// AcceptInvitation marks an invitation as accepted
func (r *Repository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID) error {
	query := `
		UPDATE user_invitations
		SET accepted_at = NOW(),
		    user_id = $1
		WHERE invitation_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, userID, invitationID)
	if err != nil {
		return fmt.Errorf("failed to accept invitation: %w", err)
	}

	return nil
}

// ==================== Password Reset Operations ====================

// StorePasswordResetToken stores a password reset token
func (r *Repository) StorePasswordResetToken(ctx context.Context, token PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (token_id, user_id, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.TokenID,
		token.UserID,
		token.TokenHash,
		token.CreatedAt,
		token.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}

	return nil
}

// GetPasswordResetToken retrieves a password reset token by hash
func (r *Repository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	query := `
		SELECT token_id, user_id, token_hash, created_at, expires_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`

	var token PasswordResetToken
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.TokenID,
		&token.UserID,
		&token.TokenHash,
		&token.CreatedAt,
		&token.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("password reset token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	return &token, nil
}

// DeletePasswordResetToken deletes a password reset token
func (r *Repository) DeletePasswordResetToken(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		DELETE FROM password_reset_tokens
		WHERE token_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to delete password reset token: %w", err)
	}

	return nil
}

// DeletePasswordResetTokenByHash deletes a password reset token by hash
func (r *Repository) DeletePasswordResetTokenByHash(ctx context.Context, tokenHash string) error {
	query := `
		DELETE FROM password_reset_tokens
		WHERE token_hash = $1
	`

	_, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete password reset token: %w", err)
	}

	return nil
}

// GetLastAccessedProgram retrieves the last program a user accessed
func (r *Repository) GetLastAccessedProgram(ctx context.Context, userID uuid.UUID) (*ProgramAccess, error) {
	// First try to get from last_program_accessed
	query := `
		SELECT p.program_id, p.program_name, p.program_code, pu.role, pu.granted_at
		FROM users u
		JOIN programs p ON u.last_program_accessed = p.program_id
		JOIN program_users pu ON p.program_id = pu.program_id AND pu.user_id = u.user_id
		WHERE u.user_id = $1 AND pu.revoked_at IS NULL AND p.deleted_at IS NULL
	`

	var pa ProgramAccess
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&pa.ProgramID,
		&pa.ProgramName,
		&pa.ProgramCode,
		&pa.Role,
		&pa.GrantedAt,
	)

	if err == nil {
		return &pa, nil
	}

	// If not found, get user's organization and then get first program
	org, err := r.GetUserOrganization(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not linked to organization: %w", err)
	}

	// Get the first program they have access to in their org
	programs, err := r.GetUserPrograms(ctx, userID, org.OrganizationID)
	if err != nil {
		return nil, err
	}

	if len(programs) == 0 {
		return nil, fmt.Errorf("user has no program access")
	}

	return &programs[0], nil
}
