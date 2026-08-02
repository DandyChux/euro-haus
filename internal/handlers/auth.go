package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dandychux/euro-haus/internal/services"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

type ValidateResponse struct {
	Valid     bool   `json:"valid"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authService := services.GetAuthService()

	user, err := authService.AuthenticateUser(
		r.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, LoginResponse{
				Success: false,
				Message: "Invalid email or password",
			})
			return
		}

		log.Printf("Failed to authenticate user: %v", err)
		writeJSON(w, http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "Unable to sign in",
		})
		return
	}

	if user.Role != "admin" {
		writeJSON(w, http.StatusForbidden, LoginResponse{
			Success: false,
			Message: "Administrator access required",
		})
		return
	}

	token, err := authService.GenerateToken(r.Context(), user)
	if err != nil {
		log.Printf("Failed to generate access token: %v", err)
		writeJSON(w, http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "Failed to generate access token",
		})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// Logout handles admin logout
func Logout(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		http.Error(w, "Invalid authorization header", http.StatusBadRequest)
		return
	}

	token := authHeader[7:]

	// Get auth service and revoke token
	authService := services.GetAuthService()
	if err := authService.RevokeToken(token); err != nil {
		log.Printf("Failed to revoke token: %v", err)
		// Continue anyway - the token might already be gone
	}

	// Return success
	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ValidateToken checks if a token is valid
func ValidateToken(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		response := ValidateResponse{
			Valid: false,
			Error: "Missing or invalid authorization header",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	token := authHeader[7:]

	// Get auth service and validate token
	authService := services.GetAuthService()
	isValid := authService.ValidateToken(token)

	if isValid {
		// Try to extend token expiration
		if err := authService.ExtendToken(token); err != nil {
			log.Printf("Failed to extend token: %v", err)
		}

		// Try to get token info for expiration time
		tokenInfo, err := authService.GetTokenInfo(token)
		response := ValidateResponse{
			Valid: true,
		}

		if err == nil && tokenInfo != nil {
			response.ExpiresAt = tokenInfo.ExpiresAt.Unix() * 1000 // Convert to milliseconds for JavaScript
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		response := ValidateResponse{
			Valid: false,
			Error: "Invalid or expired token",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
