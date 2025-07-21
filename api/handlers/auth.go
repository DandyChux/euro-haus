package handlers

import (
	"encoding/json"
	"euro-haus-api/services"
	"log"
	"net/http"
)

type LoginRequest struct {
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

// Login handles admin authentication
func Login(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse request
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get auth service
	authService := services.GetAuthService()

	// Validate password
	if !authService.ValidatePassword(req.Password) {
		response := LoginResponse{
			Success: false,
			Message: "Invalid password",
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate access token
	token, err := authService.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		response := LoginResponse{
			Success: false,
			Message: "Failed to generate access token",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return success response
	response := LoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Logout handles admin logout
func Logout(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
