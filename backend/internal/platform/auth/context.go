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

// GetUserRole extracts user role from context
func GetUserRole(ctx context.Context) (string, error) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return "", fmt.Errorf("user claims not found in context")
	}
	return claims.Role, nil
}

// IsAdmin checks if user is a global admin
func IsAdmin(ctx context.Context) bool {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return false
	}
	return claims.IsAdmin
}
