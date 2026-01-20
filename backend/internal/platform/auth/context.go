package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetUserClaims extracts user claims from context
func GetUserClaims(ctx context.Context) (*AccessTokenClaims, bool) {
	claims, ok := ctx.Value(ContextKeyUserClaims).(*AccessTokenClaims)
	return claims, ok
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return uuid.Nil, fmt.Errorf("user claims not found in context")
	}
	return claims.UserID, nil
}

// GetProgramID extracts program ID from context
func GetProgramID(ctx context.Context) (uuid.UUID, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return uuid.Nil, fmt.Errorf("user claims not found in context")
	}
	return claims.ProgramID, nil
}

// GetOrganizationID extracts organization ID from context
func GetOrganizationID(ctx context.Context) (uuid.UUID, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return uuid.Nil, fmt.Errorf("user claims not found in context")
	}
	return claims.OrganizationID, nil
}

// GetOrgRole extracts organization role from context
func GetOrgRole(ctx context.Context) (string, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return "", fmt.Errorf("user claims not found in context")
	}
	return claims.OrgRole, nil
}

// GetProgramRole extracts program role from context
func GetProgramRole(ctx context.Context) (string, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return "", fmt.Errorf("user claims not found in context")
	}
	return claims.ProgramRole, nil
}

// GetUserRole is deprecated, use GetProgramRole instead
func GetUserRole(ctx context.Context) (string, error) {
	return GetProgramRole(ctx)
}

// IsAdmin checks if user is a global admin
func IsAdmin(ctx context.Context) bool {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return false
	}
	return claims.IsAdmin
}
