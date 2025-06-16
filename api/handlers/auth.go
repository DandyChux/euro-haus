package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

// Simple in-memory session store
var (
	sessions     = make(map[string]sessionData)
	sessionMutex sync.RWMutex
)

type sessionData struct {
	Token     string
	ExpiresAt time.Time
}

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

type ValidateResponse struct {
	Valid bool `json:"valid"`
}

// generateToken creates a random session token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// cleanupExpiredSessions removes expired sessions from memory
func cleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	now := time.Now()
	for token, session := range sessions {
		if now.After(session.ExpiresAt) {
			delete(sessions, token)
		}
	}
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

	// Get admin password from environment
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "eurohaus2024" // Default for development
	}

	// Validate password
	if req.Password != adminPassword {
		response := LoginResponse{
			Success: false,
			Message: "Invalid password",
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate session token
	token, err := generateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Store session (expires in 24 hours)
	sessionMutex.Lock()
	sessions[token] = sessionData{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	sessionMutex.Unlock()

	// Clean up old sessions periodically
	go cleanupExpiredSessions()

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

	// Remove session
	sessionMutex.Lock()
	delete(sessions, token)
	sessionMutex.Unlock()

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
		json.NewEncoder(w).Encode(ValidateResponse{Valid: false})
		return
	}

	token := authHeader[7:]

	// Check if token exists and is not expired
	sessionMutex.RLock()
	session, exists := sessions[token]
	sessionMutex.RUnlock()

	valid := exists && time.Now().Before(session.ExpiresAt)

	json.NewEncoder(w).Encode(ValidateResponse{Valid: valid})
}

// VerifyAuth is a middleware function to verify authentication
func VerifyAuth(token string) bool {
	// For backward compatibility, also accept the raw admin password
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "eurohaus2024"
	}

	if token == adminPassword {
		return true
	}

	// Check session token
	sessionMutex.RLock()
	session, exists := sessions[token]
	sessionMutex.RUnlock()

	return exists && time.Now().Before(session.ExpiresAt)
}
