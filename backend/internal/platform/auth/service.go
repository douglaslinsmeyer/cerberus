package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuthService handles authentication business logic
type AuthService struct {
	repo           *Repository
	tokenService   *TokenService
	passwordHasher *PasswordHasher
}

// NewService creates a new authentication service
func NewService(repo *Repository, tokenService *TokenService) *AuthService {
	return &AuthService{
		repo:           repo,
		tokenService:   tokenService,
		passwordHasher: NewPasswordHasher(),
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// 1. Fetch user by email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// 2. Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// 3. Check if user uses password auth (not OAuth)
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("please login with OAuth provider")
	}

	// 4. Check account lockout
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, fmt.Errorf("account locked due to multiple failed attempts. Try again later")
	}

	// 5. Verify password
	valid, err := s.passwordHasher.Verify(req.Password, *user.PasswordHash)
	if err != nil || !valid {
		// Increment failed login attempts
		_ = s.repo.IncrementFailedAttempts(ctx, user.UserID)
		return nil, fmt.Errorf("invalid credentials")
	}

	// 6. Verify user has access to requested program
	programUser, err := s.repo.GetProgramUser(ctx, req.ProgramID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("no access to this program")
	}

	if programUser.RevokedAt != nil {
		return nil, fmt.Errorf("program access has been revoked")
	}

	// 7. Reset failed attempts on successful login
	_ = s.repo.ResetFailedAttempts(ctx, user.UserID)

	// 8. Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		req.ProgramID,
		user.Email,
		programUser.Role,
		user.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, tokenID, err := s.tokenService.GenerateRefreshToken(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 9. Store refresh token in database
	err = s.repo.StoreRefreshToken(ctx, RefreshToken{
		TokenID:   tokenID,
		UserID:    user.UserID,
		ProgramID: req.ProgramID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// 10. Update last login and last program accessed
	_ = s.repo.UpdateLastLogin(ctx, user.UserID)
	_ = s.repo.UpdateLastProgramAccessed(ctx, user.UserID, req.ProgramID)

	// 11. Fetch all programs user has access to
	programs, err := s.repo.GetUserPrograms(ctx, user.UserID)
	if err != nil {
		programs = []ProgramAccess{} // Return empty if failed
	}

	return &LoginResponse{
		User: UserInfo{
			UserID:   user.UserID,
			Email:    user.Email,
			FullName: user.FullName,
			IsAdmin:  user.IsAdmin,
		},
		Tokens: TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int64(s.tokenService.GetAccessTokenExpiry().Seconds()),
		},
		Programs: programs,
	}, nil
}

// RefreshToken generates new tokens using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*TokenPair, error) {
	// 1. Validate refresh token structure
	claims, err := s.tokenService.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// 2. Check if token exists in database and not revoked
	storedToken, err := s.repo.GetRefreshToken(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found")
	}

	if storedToken.RevokedAt != nil {
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// 3. Get user and verify active
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// 4. Get user's program access for the stored program
	programUser, err := s.repo.GetProgramUser(ctx, storedToken.ProgramID, user.UserID)
	if err != nil {
		// If program access lost, get their last accessed or first available program
		lastProgram, err := s.repo.GetLastAccessedProgram(ctx, user.UserID)
		if err != nil {
			return nil, fmt.Errorf("no program access available")
		}
		storedToken.ProgramID = lastProgram.ProgramID
		programUser = &ProgramUser{
			ProgramID: lastProgram.ProgramID,
			Role:      lastProgram.Role,
		}
	}

	// 5. Revoke old refresh token (rotation)
	err = s.repo.RevokeRefreshToken(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// 6. Generate new tokens
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		storedToken.ProgramID,
		user.Email,
		programUser.Role,
		user.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, newTokenID, err := s.tokenService.GenerateRefreshToken(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 7. Store new refresh token
	err = s.repo.StoreRefreshToken(ctx, RefreshToken{
		TokenID:   newTokenID,
		UserID:    user.UserID,
		ProgramID: storedToken.ProgramID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.tokenService.GetAccessTokenExpiry().Seconds()),
	}, nil
}

// SwitchProgram generates a new access token for a different program
func (s *AuthService) SwitchProgram(ctx context.Context, userID, programID uuid.UUID) (*TokenPair, error) {
	// 1. Verify user has access to the program
	programUser, err := s.repo.GetProgramUser(ctx, programID, userID)
	if err != nil {
		return nil, fmt.Errorf("no access to this program")
	}

	if programUser.RevokedAt != nil {
		return nil, fmt.Errorf("program access has been revoked")
	}

	// 2. Get user details
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// 3. Generate new access token for this program
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		programID,
		user.Email,
		programUser.Role,
		user.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 4. Update last program accessed
	_ = s.repo.UpdateLastProgramAccessed(ctx, userID, programID)

	// Note: Refresh token remains the same - no need to regenerate
	return &TokenPair{
		AccessToken: accessToken,
		ExpiresIn:   int64(s.tokenService.GetAccessTokenExpiry().Seconds()),
	}, nil
}

// Logout revokes a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	// 1. Validate refresh token structure
	claims, err := s.tokenService.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		// If token is invalid, consider logout successful (already logged out)
		return nil
	}

	// 2. Revoke the refresh token
	err = s.repo.RevokeRefreshToken(ctx, claims.TokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// GetUserPrograms returns all programs a user has access to
func (s *AuthService) GetUserPrograms(ctx context.Context, userID uuid.UUID) ([]ProgramAccess, error) {
	return s.repo.GetUserPrograms(ctx, userID)
}

// ValidateAccessToken validates an access token and returns claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	return s.tokenService.ValidateAccessToken(tokenString)
}
