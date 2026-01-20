package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenService handles JWT token generation and validation
type TokenService struct {
	jwtSecret     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewTokenService creates a new token service
func NewTokenService() *TokenService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable not set")
	}

	if len(secret) < 32 {
		panic("JWT_SECRET must be at least 32 characters long")
	}

	return &TokenService{
		jwtSecret:     []byte(secret),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

// GenerateAccessToken generates a new access token
func (ts *TokenService) GenerateAccessToken(
	userID uuid.UUID,
	organizationID uuid.UUID,
	programID uuid.UUID,
	email string,
	orgRole string,
	programRole string,
	isAdmin bool,
) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		UserID:         userID,
		Email:          email,
		OrganizationID: organizationID,
		ProgramID:      programID,
		OrgRole:        orgRole,
		ProgramRole:    programRole,
		IsAdmin:        isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cerberus-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(ts.jwtSecret)
}

// GenerateRefreshToken generates a new refresh token
func (ts *TokenService) GenerateRefreshToken(userID uuid.UUID) (string, uuid.UUID, error) {
	now := time.Now()
	tokenID := uuid.New()

	claims := RefreshTokenClaims{
		UserID:  userID,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cerberus-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(ts.jwtSecret)
	return signedToken, tokenID, err
}

// ValidateAccessToken validates an access token and returns the claims
func (ts *TokenService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ts.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (ts *TokenService) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ts.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid refresh token claims")
}

// GetAccessTokenExpiry returns the access token expiry duration
func (ts *TokenService) GetAccessTokenExpiry() time.Duration {
	return ts.accessExpiry
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func (ts *TokenService) GetRefreshTokenExpiry() time.Duration {
	return ts.refreshExpiry
}
