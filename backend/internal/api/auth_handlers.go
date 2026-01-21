package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/cerberus/backend/internal/platform/auth"
	"github.com/cerberus/backend/internal/platform/db"
)

// handleRegister handles user registration
// Currently returns 501 - registration is by invitation only
func handleRegister(db *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusNotImplemented, ErrorResponse{
			Error: "Registration is by invitation only. Contact your administrator.",
		})
	}
}

// handleLogin handles user login with email/password
func handleLogin(db *db.DB, authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req auth.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		if req.Email == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		// Attempt login
		loginResp, err := authService.Login(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Set httpOnly cookie for refresh token (if tokens exist)
		if loginResp.Tokens != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    loginResp.Tokens.RefreshToken,
				Path:     "/",
				HttpOnly: true,
				Secure:   isProduction(),
				SameSite: http.SameSiteStrictMode,
				MaxAge:   7 * 24 * 60 * 60, // 7 days
			})
		}

		// Return access token in response body
		respondJSON(w, http.StatusOK, SuccessResponse{
			Success: true,
			Data:    loginResp,
		})
	}
}

// handleRefreshToken handles refresh token rotation
func handleRefreshToken(db *db.DB, authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get refresh token from httpOnly cookie
		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Refresh token required")
			return
		}

		// Refresh tokens and get full session data
		sessionData, err := authService.RefreshToken(r.Context(), cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   -1, // Delete cookie
			})
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Set new httpOnly cookie for new refresh token
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    sessionData.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   isProduction(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   7 * 24 * 60 * 60,
		})

		// Return full session data (refresh token excluded from JSON)
		respondJSON(w, http.StatusOK, SuccessResponse{
			Success: true,
			Data:    sessionData,
		})
	}
}

// handleSwitchProgram handles switching between programs
func handleSwitchProgram(authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req auth.SwitchProgramRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Get user ID and organization ID from claims (injected by AuthMiddleware)
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		organizationID, err := auth.GetOrganizationID(r.Context())
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Organization context required")
			return
		}

		// Switch program
		tokens, err := authService.SwitchProgram(r.Context(), userID, organizationID, req.ProgramID)
		if err != nil {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, SuccessResponse{
			Success: true,
			Data:    tokens,
		})
	}
}

// handleLogout handles user logout
func handleLogout(authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get refresh token from httpOnly cookie
		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			// No refresh token, but that's okay - consider logout successful
			respondNoContent(w)
			return
		}

		// Revoke refresh token
		err = authService.Logout(r.Context(), cookie.Value)
		if err != nil {
			// Log error but still return success (idempotent)
		}

		// Clear refresh token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1, // Delete cookie
		})

		respondNoContent(w)
	}
}

// isProduction checks if running in production environment
func isProduction() bool {
	env := os.Getenv("APP_ENV")
	return env == "production"
}
