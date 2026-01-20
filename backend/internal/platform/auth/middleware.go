package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// AuthMiddleware validates JWT and injects claims into context
func AuthMiddleware(tokenService *TokenService, repo *Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := tokenService.ValidateAccessToken(tokenString)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Check if user is still active
			user, err := repo.GetUserByID(r.Context(), claims.UserID)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "User not found")
				return
			}

			if !user.IsActive {
				respondError(w, http.StatusUnauthorized, "Account is inactive")
				return
			}

			// Check account lockout
			if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
				respondError(w, http.StatusUnauthorized, "Account is locked")
				return
			}

			// Inject claims into context
			ctx := context.WithValue(r.Context(), ContextKeyUserClaims, claims)

			// Continue with request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireProgramAccess verifies user has access to program in URL with minimum role
func RequireProgramAccess(minRole int, repo *Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context (set by AuthMiddleware)
			claims, ok := GetUserClaims(r.Context())
			if !ok {
				respondError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Extract program_id from URL
			programIDStr := chi.URLParam(r, "programId")
			if programIDStr == "" {
				respondError(w, http.StatusBadRequest, "Program ID required in URL")
				return
			}

			programID, err := uuid.Parse(programIDStr)
			if err != nil {
				respondError(w, http.StatusBadRequest, "Invalid program ID format")
				return
			}

			// Verify JWT program_id matches URL program_id
			if claims.ProgramID != programID {
				respondError(w, http.StatusForbidden, "Token not valid for this program. Use /auth/switch-program")
				return
			}

			// NEW: Verify program belongs to user's organization
			program, err := repo.GetProgram(r.Context(), programID)
			if err != nil {
				respondError(w, http.StatusNotFound, "Program not found")
				return
			}

			if program.OrganizationID != claims.OrganizationID {
				respondError(w, http.StatusForbidden, "Program not in your organization")
				return
			}

			// Verify user still has access to program
			programUser, err := repo.GetProgramUser(r.Context(), programID, claims.UserID)
			if err != nil {
				respondError(w, http.StatusForbidden, "No access to this program")
				return
			}

			// Check access not revoked
			if programUser.RevokedAt != nil {
				respondError(w, http.StatusForbidden, "Program access has been revoked")
				return
			}

			// Check role meets minimum requirement
			userRole, err := RoleFromString(programUser.Role)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Invalid user role")
				return
			}

			if userRole < minRole {
				minRoleStr, _ := RoleToString(minRole)
				respondError(w, http.StatusForbidden, fmt.Sprintf("Requires %s role or higher", minRoleStr))
				return
			}

			// TODO: Log access attempt to audit_logs

			// Continue with request
			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrganizationAccess verifies user has organization access with minimum role
func RequireOrganizationAccess(minRole int, repo *Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok {
				respondError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Verify org membership still valid
			orgUser, err := repo.GetOrganizationUser(r.Context(), claims.OrganizationID, claims.UserID)
			if err != nil || orgUser.RevokedAt != nil {
				respondError(w, http.StatusForbidden, "Organization access revoked")
				return
			}

			// Check role meets minimum requirement
			userRole, err := OrgRoleFromString(orgUser.OrgRole)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Invalid org role")
				return
			}

			if userRole < minRole {
				minRoleStr, _ := OrgRoleToString(minRole)
				respondError(w, http.StatusForbidden, fmt.Sprintf("Requires %s org role or higher", minRoleStr))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireGlobalAdmin verifies user is a global system admin
func RequireGlobalAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok || !claims.IsAdmin {
				respondError(w, http.StatusForbidden, "Global admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanupInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors:        make(map[string]*rate.Limiter),
		rate:            rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:           requestsPerMinute,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// getLimiter gets or creates a limiter for an IP address
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}

	return limiter
}

// cleanupVisitors periodically removes old visitors
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// Simple cleanup: remove all visitors
		// In production, you might want to track last access time
		rl.visitors = make(map[string]*rate.Limiter)
		rl.mu.Unlock()
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			respondError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		// Return the first IP (client)
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS - Force HTTPS
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS protection
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// respondError sends a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"success":false,"error":"%s"}`, message)
}
