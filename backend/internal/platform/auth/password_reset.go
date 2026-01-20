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

// InitiatePasswordReset initiates a password reset flow
func (s *AuthService) InitiatePasswordReset(ctx context.Context, email string) error {
	// Fetch user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists - always return success
		return nil
	}

	// Check if user uses password auth (not OAuth only)
	if user.PasswordHash == nil && user.AuthProvider != nil {
		// OAuth-only user, can't reset password
		// TODO: Send email informing them to use OAuth login
		return nil
	}

	// Generate secure reset token (32 bytes random)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}
	_ = base64.URLEncoding.EncodeToString(tokenBytes) // TODO: Send via email

	// Hash token for storage
	hashedToken := sha256.Sum256(tokenBytes)
	tokenHash := hex.EncodeToString(hashedToken[:])

	// Store token in database (expires in 1 hour)
	err = s.repo.StorePasswordResetToken(ctx, PasswordResetToken{
		TokenID:   uuid.New(),
		UserID:    user.UserID,
		TokenHash: tokenHash,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// TODO: Send email with reset link
	// Format: https://app.cerberus.com/reset-password?token=tokenStr
	// s.emailService.SendPasswordResetEmail(user.Email, user.FullName, resetURL)

	return nil
}

// ResetPassword completes a password reset
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// 1. Validate password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 2. Hash the provided token to lookup in database
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("invalid reset token format")
	}

	hashedToken := sha256.Sum256(tokenBytes)
	tokenHash := hex.EncodeToString(hashedToken[:])

	// 3. Get reset token from database
	resetToken, err := s.repo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// 4. Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		// Clean up expired token
		_ = s.repo.DeletePasswordResetToken(ctx, resetToken.TokenID)
		return fmt.Errorf("reset token has expired")
	}

	// 5. Hash new password
	passwordHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 6. Update password
	err = s.repo.UpdatePassword(ctx, resetToken.UserID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 7. Revoke all refresh tokens for security
	// This forces the user to login again on all devices
	err = s.repo.RevokeAllRefreshTokens(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}

	// 8. Reset failed login attempts
	_ = s.repo.ResetFailedAttempts(ctx, resetToken.UserID)

	// 9. Delete the reset token (single use)
	err = s.repo.DeletePasswordResetToken(ctx, resetToken.TokenID)
	if err != nil {
		return fmt.Errorf("failed to delete reset token: %w", err)
	}

	// TODO: Send email confirming password change
	// s.emailService.SendPasswordChangedEmail(user.Email, user.FullName)

	return nil
}

// ChangePassword allows an authenticated user to change their password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// 1. Get user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// 2. Check if user has a password (not OAuth-only)
	if user.PasswordHash == nil {
		return fmt.Errorf("cannot change password for OAuth-only accounts")
	}

	// 3. Verify current password
	valid, err := s.passwordHasher.Verify(currentPassword, *user.PasswordHash)
	if err != nil || !valid {
		return fmt.Errorf("current password is incorrect")
	}

	// 4. Validate new password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 5. Check new password is different from current
	samePassword, _ := s.passwordHasher.Verify(newPassword, *user.PasswordHash)
	if samePassword {
		return fmt.Errorf("new password must be different from current password")
	}

	// 6. Hash new password
	passwordHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 7. Update password
	err = s.repo.UpdatePassword(ctx, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 8. Revoke all refresh tokens except current one (force re-login on other devices)
	err = s.repo.RevokeAllRefreshTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}

	// TODO: Send email confirming password change
	// s.emailService.SendPasswordChangedEmail(user.Email, user.FullName)

	return nil
}
