package auth

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	UserID               uuid.UUID  `json:"user_id"`
	Email                string     `json:"email"`
	FullName             string     `json:"full_name"`
	PasswordHash         *string    `json:"-"` // Never expose password hash
	AuthProvider         *string    `json:"auth_provider,omitempty"`
	AuthProviderID       *string    `json:"auth_provider_id,omitempty"`
	IsActive             bool       `json:"is_active"`
	IsAdmin              bool       `json:"is_admin"`
	CreatedAt            time.Time  `json:"created_at"`
	LastLoginAt          *time.Time `json:"last_login_at,omitempty"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty"`
	FailedLoginAttempts  int        `json:"-"`
	LockedUntil          *time.Time `json:"-"`
	LastProgramAccessed  *uuid.UUID `json:"-"`
}

// ProgramUser represents a user's access to a program
type ProgramUser struct {
	ProgramUserID uuid.UUID  `json:"program_user_id"`
	ProgramID     uuid.UUID  `json:"program_id"`
	UserID        uuid.UUID  `json:"user_id"`
	Role          string     `json:"role"` // 'admin', 'contributor', 'viewer'
	GrantedAt     time.Time  `json:"granted_at"`
	GrantedBy     *uuid.UUID `json:"granted_by,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

// ProgramAccess represents a program a user has access to
type ProgramAccess struct {
	ProgramID   uuid.UUID `json:"program_id"`
	ProgramName string    `json:"program_name"`
	ProgramCode string    `json:"program_code"`
	Role        string    `json:"role"`
	GrantedAt   time.Time `json:"granted_at"`
}

// Organization represents a top-level tenant organization
type Organization struct {
	OrganizationID   uuid.UUID       `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	OrganizationCode string          `json:"organization_code"`
	Status           string          `json:"status"`
	Settings         json.RawMessage `json:"settings"`
	PlanTier         string          `json:"plan_tier"`
	MaxPrograms      int             `json:"max_programs"`
	MaxUsers         int             `json:"max_users"`
	CreatedAt        time.Time       `json:"created_at"`
	CreatedBy        *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty"`
}

// OrganizationUser represents a user's membership in an organization
type OrganizationUser struct {
	OrganizationUserID uuid.UUID  `json:"organization_user_id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	UserID             uuid.UUID  `json:"user_id"`
	OrgRole            string     `json:"org_role"` // 'owner', 'admin', 'member'
	GrantedAt          time.Time  `json:"granted_at"`
	GrantedBy          *uuid.UUID `json:"granted_by,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
}

// OrganizationInfo represents organization information in responses
type OrganizationInfo struct {
	OrganizationID   uuid.UUID `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	OrganizationCode string    `json:"organization_code"`
	OrgRole          string    `json:"org_role"` // User's role in org
}

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	TokenID        uuid.UUID  `json:"token_id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"organization_id"` // NEW
	ProgramID      uuid.UUID  `json:"program_id"`
	IssuedAt       time.Time  `json:"issued_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	IPAddress      *string    `json:"ip_address,omitempty"`
	UserAgent      *string    `json:"user_agent,omitempty"`
}

// Invitation represents a user invitation
type Invitation struct {
	InvitationID   uuid.UUID  `json:"invitation_id"`
	Email          string     `json:"email"`
	OrganizationID uuid.UUID  `json:"organization_id"` // NEW
	ProgramID      uuid.UUID  `json:"program_id"`
	OrgRole        string     `json:"org_role"`  // NEW: Organization-level role
	Role           string     `json:"role"`      // Program-level role
	IsAdmin        bool       `json:"is_admin"`
	InvitedBy      uuid.UUID  `json:"invited_by"`
	InvitedAt      time.Time  `json:"invited_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Token          string     `json:"token,omitempty"` // Only populated when creating
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	TokenID   uuid.UUID `json:"token_id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"` // Never expose
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// No ProgramID required - backend will determine it
}

// LoginResponse represents a login response
type LoginResponse struct {
	User           UserInfo          `json:"user"`
	Organization   OrganizationInfo  `json:"organization"`      // NEW
	CurrentProgram *ProgramAccess    `json:"current_program"`   // NEW - may be nil if no program access
	Tokens         *TokenPair        `json:"tokens,omitempty"`  // Nil if no program access
	Programs       []ProgramAccess   `json:"programs"`          // All programs user can access
}

// UserInfo represents basic user information
type UserInfo struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	IsAdmin  bool      `json:"is_admin"`
}

// TokenPair represents an access token and refresh token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents a refresh token response with full session data
type RefreshResponse struct {
	User           UserInfo        `json:"user"`
	Organization   OrganizationInfo `json:"organization"`
	CurrentProgram *ProgramAccess  `json:"current_program"`
	Programs       []ProgramAccess `json:"programs"`
	AccessToken    string          `json:"access_token"`
	RefreshToken   string          `json:"-"` // Not sent in JSON, used for cookie
	ExpiresIn      int64           `json:"expires_in"` // seconds
}

// SwitchProgramRequest represents a program switch request
type SwitchProgramRequest struct {
	ProgramID uuid.UUID `json:"program_id"`
}

// CreateInvitationRequest represents an invitation creation request
type CreateInvitationRequest struct {
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id"` // NEW
	ProgramID      uuid.UUID `json:"program_id"`
	OrgRole        string    `json:"org_role"` // NEW: 'owner', 'admin', 'member'
	Role           string    `json:"role"`     // Program-level: 'admin', 'contributor', 'viewer'
	IsAdmin        bool      `json:"is_admin"`
}

// AcceptInvitationRequest represents an invitation acceptance request
type AcceptInvitationRequest struct {
	Token    string `json:"token"`
	FullName string `json:"full_name"`
	Password string `json:"password"` // Required for password-based auth
}

// InitiatePasswordResetRequest represents a password reset initiation request
type InitiatePasswordResetRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// OAuthUserInfo represents user information from OAuth provider
type OAuthUserInfo struct {
	ProviderID string `json:"provider_id"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
