package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateInvitation creates a new user invitation
func (s *AuthService) CreateInvitation(ctx context.Context, inviterID uuid.UUID, req CreateInvitationRequest) (*Invitation, error) {
	// 1. Verify role is valid
	if err := ValidateRole(req.Role); err != nil {
		return nil, fmt.Errorf("invalid role: %w", err)
	}

	// 2. Verify inviter has permission
	inviter, err := s.repo.GetUserByID(ctx, inviterID)
	if err != nil {
		return nil, fmt.Errorf("inviter not found")
	}

	// Check if inviter is global admin or admin of the program
	if !inviter.IsAdmin {
		programUser, err := s.repo.GetProgramUser(ctx, req.ProgramID, inviterID)
		if err != nil || programUser.Role != RoleNameAdmin {
			return nil, fmt.Errorf("insufficient permissions to invite users")
		}
	}

	// 3. Check if user already exists with access
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		// User exists - check if they already have access
		_, err := s.repo.GetProgramUser(ctx, req.ProgramID, existingUser.UserID)
		if err == nil {
			return nil, fmt.Errorf("user already has access to this program")
		}
	}

	// 4. Check if pending invitation exists
	existing, err := s.repo.GetPendingInvitation(ctx, req.Email)
	if err == nil && existing.ProgramID == req.ProgramID {
		return nil, fmt.Errorf("invitation already sent to this email for this program")
	}

	// 5. Generate secure invitation token (32 bytes)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// 6. Hash token for storage
	hashedToken := sha256.Sum256(tokenBytes)
	tokenHash := hex.EncodeToString(hashedToken[:])

	// 7. Create invitation
	invitation := Invitation{
		InvitationID: uuid.New(),
		Email:        req.Email,
		ProgramID:    req.ProgramID,
		Role:         req.Role,
		IsAdmin:      req.IsAdmin,
		InvitedBy:    inviterID,
		InvitedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
		Token:        token,                                // Only set for response
	}

	err = s.repo.CreateInvitation(ctx, &invitation, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// TODO: Send invitation email
	// s.emailService.SendInvitationEmail(req.Email, inviter.FullName, inviteURL)

	return &invitation, nil
}

// GetInvitation retrieves an invitation by token
func (s *AuthService) GetInvitation(ctx context.Context, token string) (*Invitation, error) {
	// Hash the token to lookup
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token format")
	}

	hashedToken := sha256.Sum256(tokenBytes)
	tokenHash := hex.EncodeToString(hashedToken[:])

	invitation, err := s.repo.GetInvitationByToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invitation not found")
	}

	if invitation.AcceptedAt != nil {
		return nil, fmt.Errorf("invitation already accepted")
	}

	if time.Now().After(invitation.ExpiresAt) {
		return nil, fmt.Errorf("invitation expired")
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation and creates/updates user account
func (s *AuthService) AcceptInvitation(ctx context.Context, req AcceptInvitationRequest) (*LoginResponse, error) {
	// 1. Validate invitation token
	invitation, err := s.GetInvitation(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	// 2. Validate password strength
	if err := ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	// 3. Hash password
	passwordHash, err := s.passwordHasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, invitation.Email)
	var user *User

	if err == nil {
		// User exists - update password if they don't have one (OAuth user)
		user = existingUser
		if user.PasswordHash == nil {
			err = s.repo.UpdatePassword(ctx, user.UserID, passwordHash)
			if err != nil {
				return nil, fmt.Errorf("failed to update password: %w", err)
			}
		}
	} else {
		// Create new user
		user = &User{
			UserID:       uuid.New(),
			Email:        invitation.Email,
			FullName:     req.FullName,
			PasswordHash: &passwordHash,
			IsActive:     true,
			IsAdmin:      invitation.IsAdmin,
			CreatedAt:    time.Now(),
		}

		err = s.repo.CreateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// 5. Grant program access (if not already exists)
	_, err = s.repo.GetProgramUser(ctx, invitation.ProgramID, user.UserID)
	if err != nil {
		// Access doesn't exist, create it
		err = s.repo.CreateProgramUser(ctx, ProgramUser{
			ProgramUserID: uuid.New(),
			ProgramID:     invitation.ProgramID,
			UserID:        user.UserID,
			Role:          invitation.Role,
			GrantedBy:     &invitation.InvitedBy,
			GrantedAt:     time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to grant program access: %w", err)
		}
	}

	// 6. Mark invitation as accepted
	err = s.repo.AcceptInvitation(ctx, invitation.InvitationID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark invitation as accepted: %w", err)
	}

	// 7. Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(
		user.UserID,
		invitation.ProgramID,
		user.Email,
		invitation.Role,
		user.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, tokenID, err := s.tokenService.GenerateRefreshToken(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 8. Store refresh token
	err = s.repo.StoreRefreshToken(ctx, RefreshToken{
		TokenID:   tokenID,
		UserID:    user.UserID,
		ProgramID: invitation.ProgramID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// 9. Update last program accessed
	_ = s.repo.UpdateLastProgramAccessed(ctx, user.UserID, invitation.ProgramID)

	// 10. Fetch all programs user has access to
	programs, err := s.repo.GetUserPrograms(ctx, user.UserID)
	if err != nil {
		programs = []ProgramAccess{}
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

// GetOrCreateOAuthUser gets or creates a user from OAuth provider info
func (s *AuthService) GetOrCreateOAuthUser(ctx context.Context, provider string, userInfo *OAuthUserInfo) (*User, error) {
	// Try to find user by provider ID
	user, err := s.repo.GetUserByOAuthProvider(ctx, provider, userInfo.ProviderID)
	if err == nil {
		// User exists, update last login
		_ = s.repo.UpdateLastLogin(ctx, user.UserID)
		return user, nil
	}

	// User doesn't exist with this provider - check by email
	existingUser, err := s.repo.GetUserByEmail(ctx, userInfo.Email)
	if err == nil {
		// Email exists - link OAuth account to existing user
		err = s.repo.LinkOAuthProvider(ctx, existingUser.UserID, provider, userInfo.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to link OAuth provider: %w", err)
		}
		return existingUser, nil
	}

	// New user - check if they have an invitation
	invitation, err := s.repo.GetPendingInvitation(ctx, userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("no invitation found for this email. Please contact an administrator")
	}

	// Create new user
	newUser := User{
		UserID:         uuid.New(),
		Email:          userInfo.Email,
		FullName:       userInfo.FullName,
		AuthProvider:   &provider,
		AuthProviderID: &userInfo.ProviderID,
		IsActive:       true,
		IsAdmin:        invitation.IsAdmin,
		CreatedAt:      time.Now(),
	}

	err = s.repo.CreateUser(ctx, &newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Grant program access based on invitation
	err = s.repo.CreateProgramUser(ctx, ProgramUser{
		ProgramUserID: uuid.New(),
		ProgramID:     invitation.ProgramID,
		UserID:        newUser.UserID,
		Role:          invitation.Role,
		GrantedBy:     &invitation.InvitedBy,
		GrantedAt:     time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to grant program access: %w", err)
	}

	// Mark invitation as accepted
	_ = s.repo.AcceptInvitation(ctx, invitation.InvitationID, newUser.UserID)

	return &newUser, nil
}
