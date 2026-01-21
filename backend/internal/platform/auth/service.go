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

	// 6. Reset failed attempts on successful login
	_ = s.repo.ResetFailedAttempts(ctx, user.UserID)

	// 7. Get user's organization
	org, err := s.repo.GetUserOrganization(ctx, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not linked to organization")
	}

	// 8. Get organization membership
	orgUser, err := s.repo.GetOrganizationUser(ctx, org.OrganizationID, user.UserID)
	if err != nil || orgUser.RevokedAt != nil {
		return nil, fmt.Errorf("organization access revoked")
	}

	// 9. Fetch all programs user has access to in their org
	programs, err := s.repo.GetUserPrograms(ctx, user.UserID, org.OrganizationID)
	if err != nil {
		programs = []ProgramAccess{}
	}

	// 10. If no programs, return response without tokens
	if len(programs) == 0 {
		return &LoginResponse{
			User: UserInfo{
				UserID:   user.UserID,
				Email:    user.Email,
				FullName: user.FullName,
				IsAdmin:  user.IsAdmin,
			},
			Organization: OrganizationInfo{
				OrganizationID:   org.OrganizationID,
				OrganizationName: org.OrganizationName,
				OrganizationCode: org.OrganizationCode,
				OrgRole:          orgUser.OrgRole,
			},
			CurrentProgram: nil,
			Tokens:         nil,
			Programs:       []ProgramAccess{},
		}, nil
	}

	// 11. Select program (last accessed or first)
	selectedProgram := programs[0]
	if user.LastProgramAccessed != nil {
		for _, p := range programs {
			if p.ProgramID == *user.LastProgramAccessed {
				selectedProgram = p
				break
			}
		}
	}

	// 12. Get program access details
	programUser, err := s.repo.GetProgramUser(ctx, selectedProgram.ProgramID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get program access: %w", err)
	}

	// 13. Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		org.OrganizationID,
		selectedProgram.ProgramID,
		user.Email,
		orgUser.OrgRole,
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

	// 14. Store refresh token in database
	err = s.repo.StoreRefreshToken(ctx, RefreshToken{
		TokenID:        tokenID,
		UserID:         user.UserID,
		OrganizationID: org.OrganizationID,
		ProgramID:      selectedProgram.ProgramID,
		IssuedAt:       time.Now(),
		ExpiresAt:      time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// 15. Update last login and last program accessed
	_ = s.repo.UpdateLastLogin(ctx, user.UserID)
	_ = s.repo.UpdateLastProgramAccessed(ctx, user.UserID, selectedProgram.ProgramID)

	return &LoginResponse{
		User: UserInfo{
			UserID:   user.UserID,
			Email:    user.Email,
			FullName: user.FullName,
			IsAdmin:  user.IsAdmin,
		},
		Organization: OrganizationInfo{
			OrganizationID:   org.OrganizationID,
			OrganizationName: org.OrganizationName,
			OrganizationCode: org.OrganizationCode,
			OrgRole:          orgUser.OrgRole,
		},
		CurrentProgram: &selectedProgram,
		Tokens: &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int64(s.tokenService.GetAccessTokenExpiry().Seconds()),
		},
		Programs: programs,
	}, nil
}

// RefreshToken generates new tokens using a refresh token and returns full session data
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*RefreshResponse, error) {
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

	// 4. Get organization and organization membership
	org, err := s.repo.GetOrganizationByID(ctx, storedToken.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	orgUser, err := s.repo.GetOrganizationUser(ctx, storedToken.OrganizationID, user.UserID)
	if err != nil || orgUser.RevokedAt != nil {
		return nil, fmt.Errorf("organization access revoked")
	}

	// 5. Get all user's programs in this organization
	programs, err := s.repo.GetUserPrograms(ctx, user.UserID, storedToken.OrganizationID)
	if err != nil || len(programs) == 0 {
		return nil, fmt.Errorf("no program access available")
	}

	// 6. Find the current program (or use first if not found)
	var currentProgram *ProgramAccess
	for i := range programs {
		if programs[i].ProgramID == storedToken.ProgramID {
			currentProgram = &programs[i]
			break
		}
	}
	if currentProgram == nil {
		// Program access was revoked, use first available
		currentProgram = &programs[0]
		storedToken.ProgramID = programs[0].ProgramID
	}

	// 7. Get program user for role
	programUser, err := s.repo.GetProgramUser(ctx, currentProgram.ProgramID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("program access not found")
	}

	// 8. Revoke old refresh token (rotation)
	err = s.repo.RevokeRefreshToken(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// 9. Generate new tokens
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		storedToken.OrganizationID,
		currentProgram.ProgramID,
		user.Email,
		orgUser.OrgRole,
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

	// 10. Store new refresh token
	err = s.repo.StoreRefreshToken(ctx, RefreshToken{
		TokenID:        newTokenID,
		UserID:         user.UserID,
		OrganizationID: storedToken.OrganizationID,
		ProgramID:      currentProgram.ProgramID,
		IssuedAt:       time.Now(),
		ExpiresAt:      time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	// 11. Return full session data (newRefreshToken will be set as cookie by handler)
	resp := &RefreshResponse{
		User: UserInfo{
			UserID:   user.UserID,
			Email:    user.Email,
			FullName: user.FullName,
			IsAdmin:  user.IsAdmin,
		},
		Organization: OrganizationInfo{
			OrganizationID:   org.OrganizationID,
			OrganizationName: org.OrganizationName,
			OrganizationCode: org.OrganizationCode,
			OrgRole:          orgUser.OrgRole,
		},
		CurrentProgram: currentProgram,
		Programs:       programs,
		AccessToken:    accessToken,
		ExpiresIn:      int64(s.tokenService.GetAccessTokenExpiry().Seconds()),
	}

	// Store the new refresh token string in the response (handler will extract it for cookie)
	resp.RefreshToken = newRefreshToken

	return resp, nil
}

// SwitchProgram generates a new access token for a different program
func (s *AuthService) SwitchProgram(ctx context.Context, userID, organizationID, programID uuid.UUID) (*TokenPair, error) {
	// 1. Verify program belongs to user's organization
	program, err := s.repo.GetProgram(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("program not found")
	}

	if program.OrganizationID != organizationID {
		return nil, fmt.Errorf("program not in your organization")
	}

	// 2. Verify user has access to the program
	programUser, err := s.repo.GetProgramUser(ctx, programID, userID)
	if err != nil {
		return nil, fmt.Errorf("no access to this program")
	}

	if programUser.RevokedAt != nil {
		return nil, fmt.Errorf("program access has been revoked")
	}

	// 3. Get user details
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// 4. Get organization membership
	orgUser, err := s.repo.GetOrganizationUser(ctx, organizationID, userID)
	if err != nil || orgUser.RevokedAt != nil {
		return nil, fmt.Errorf("organization access revoked")
	}

	// 5. Generate new access token for this program
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		organizationID,
		programID,
		user.Email,
		orgUser.OrgRole,
		programUser.Role,
		user.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 6. Update last program accessed
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
	// Get user's organization first
	org, err := s.repo.GetUserOrganization(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not linked to organization: %w", err)
	}

	return s.repo.GetUserPrograms(ctx, userID, org.OrganizationID)
}

// ValidateAccessToken validates an access token and returns claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	return s.tokenService.ValidateAccessToken(tokenString)
}
