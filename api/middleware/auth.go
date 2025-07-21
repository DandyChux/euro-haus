package middleware

import (
	"euro-haus-api/services"
	"net/http"
	"strings"
)

// AuthValidator is a function type for validating tokens
type AuthValidator func(token string) bool

// RequireAuth creates a middleware that checks for valid authentication
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for OPTIONS requests (preflight)
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate the token using the auth service
		authService := services.GetAuthService()
		if !authService.ValidateToken(token) {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

// RequireAdminAuth is an alias for RequireAuth for clarity
func RequireAdminAuth(next http.Handler) http.Handler {
	return RequireAuth(next)
}
