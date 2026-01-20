package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenClaims represents the claims in an access token
type AccessTokenClaims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id"` // NEW: Organization context
	ProgramID      uuid.UUID `json:"program_id"`      // Program context (within org)
	OrgRole        string    `json:"org_role"`        // NEW: Organization-level role (owner/admin/member)
	ProgramRole    string    `json:"program_role"`    // Program-level role (admin/contributor/viewer)
	IsAdmin        bool      `json:"is_admin"`        // Global system admin flag
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents the claims in a refresh token
type RefreshTokenClaims struct {
	UserID  uuid.UUID `json:"user_id"`
	TokenID uuid.UUID `json:"token_id"` // Unique ID for this refresh token
	jwt.RegisteredClaims
}

// ContextKey type for context keys
type ContextKey string

const (
	// ContextKeyUserClaims is the context key for user claims
	ContextKeyUserClaims ContextKey = "user_claims"
)
