package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordHasher handles password hashing and verification using Argon2id
type PasswordHasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// NewPasswordHasher creates a new password hasher with secure defaults
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		time:    3,          // Number of iterations
		memory:  64 * 1024,  // 64 MB
		threads: 4,          // Parallelism
		keyLen:  32,         // Output key length
	}
}

// Hash generates an Argon2id hash for the given password
func (ph *PasswordHasher) Hash(password string) (string, error) {
	// Generate a cryptographically secure random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the hash
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		ph.time,
		ph.memory,
		ph.threads,
		ph.keyLen,
	)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		ph.memory,
		ph.time,
		ph.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// Verify checks if the provided password matches the encoded hash
func (ph *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Verify it's argon2id
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported hash algorithm: %s", parts[1])
	}

	// Parse parameters
	var version int
	var memory, time uint32
	var threads uint8

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("failed to parse version: %w", err)
	}

	if version != argon2.Version {
		return false, fmt.Errorf("incompatible argon2 version")
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decode hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute hash with the provided password and stored parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(hash)),
	)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return true, nil
	}

	return false, nil
}

// ValidatePasswordStrength checks if password meets minimum requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters long")
	}

	// Could add additional requirements here:
	// - At least one uppercase letter
	// - At least one lowercase letter
	// - At least one digit
	// - At least one special character
	// For now, we just enforce minimum length

	return nil
}
