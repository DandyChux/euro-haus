package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// TokenData represents the data stored for a token
type TokenData struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthService handles authentication operations
type AuthService struct {
	adminPassword string
	tokenTTL      time.Duration
	memoryStore   map[string]TokenData
	mutex         sync.RWMutex
}

// NewAuthService creates a new auth service instance
func NewAuthService() *AuthService {
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		// Use default for backward compatibility
		adminPassword = "eurohaus2024"
		log.Println("Warning: ADMIN_PASSWORD not set, using default value")
	}

	return &AuthService{
		adminPassword: adminPassword,
		tokenTTL:      24 * time.Hour, // Tokens expire after 24 hours
		memoryStore:   make(map[string]TokenData),
	}
}

// ValidatePassword checks if the provided password matches the admin password
func (s *AuthService) ValidatePassword(password string) bool {
	return password == s.adminPassword
}

// GenerateToken creates a new access token
func (s *AuthService) GenerateToken() (string, error) {
	// Generate a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store token with expiration
	tokenData := TokenData{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.tokenTTL),
	}

	// Store in Redis if available
	redisClient := GetRedisClient()
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)

		// Serialize token data
		data, err := json.Marshal(tokenData)
		if err != nil {
			log.Printf("Failed to marshal token data: %v", err)
		} else {
			// Store in Redis with TTL
			err = redisClient.Set(ctx, key, data, s.tokenTTL).Err()
			if err != nil {
				log.Printf("Failed to store token in Redis: %v", err)
			}
		}
	}

	// Also store in memory as fallback
	s.mutex.Lock()
	s.memoryStore[token] = tokenData
	s.mutex.Unlock()

	return token, nil
}

// ValidateToken checks if a token is valid and not expired
func (s *AuthService) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	// Remove any whitespace
	token = strings.TrimSpace(token)

	// For backward compatibility, also accept the raw admin password
	if token == s.adminPassword {
		return true
	}

	// Try Redis first
	redisClient := GetRedisClient()
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)

		// Get from Redis
		data, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			// Parse token data
			var tokenData TokenData
			if err := json.Unmarshal([]byte(data), &tokenData); err == nil {
				// Check if token is expired
				if time.Now().After(tokenData.ExpiresAt) {
					// Token expired, remove it
					redisClient.Del(ctx, key)
					s.removeFromMemory(token)
					return false
				}
				return true
			}
		}
	}

	// Fallback to in-memory storage
	s.mutex.RLock()
	tokenData, exists := s.memoryStore[token]
	s.mutex.RUnlock()

	if !exists {
		return false
	}

	// Check if token is expired
	if time.Now().After(tokenData.ExpiresAt) {
		// Token expired, remove it
		s.removeFromMemory(token)
		return false
	}

	return true
}

// RevokeToken removes a token from storage
func (s *AuthService) RevokeToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Remove from Redis if available
	redisClient := GetRedisClient()
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)
		err := redisClient.Del(ctx, key).Err()
		if err != nil {
			log.Printf("Failed to delete token from Redis: %v", err)
		}
	}

	// Also remove from in-memory storage
	s.removeFromMemory(token)

	return nil
}

// ExtendToken extends the expiration time of a valid token
func (s *AuthService) ExtendToken(token string) error {
	// Don't extend if it's the raw password
	if token == s.adminPassword {
		return nil
	}

	if !s.ValidateToken(token) {
		return fmt.Errorf("invalid or expired token")
	}

	// Get existing token data
	var tokenData TokenData
	found := false

	// Try Redis first
	redisClient := GetRedisClient()
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)
		data, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			if err := json.Unmarshal([]byte(data), &tokenData); err == nil {
				found = true
			}
		}
	}

	// Fallback to memory if not found in Redis
	if !found {
		s.mutex.RLock()
		td, exists := s.memoryStore[token]
		s.mutex.RUnlock()
		if exists {
			tokenData = td
			found = true
		}
	}

	if !found {
		return fmt.Errorf("token not found")
	}

	// Extend expiration
	tokenData.ExpiresAt = time.Now().Add(s.tokenTTL)

	// Update in Redis if available
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)
		data, err := json.Marshal(tokenData)
		if err == nil {
			err = redisClient.Set(ctx, key, data, s.tokenTTL).Err()
			if err != nil {
				log.Printf("Failed to update token in Redis: %v", err)
			}
		}
	}

	// Update in memory
	s.mutex.Lock()
	s.memoryStore[token] = tokenData
	s.mutex.Unlock()

	return nil
}

// GetTokenInfo returns information about a token
func (s *AuthService) GetTokenInfo(token string) (*TokenData, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Try Redis first
	redisClient := GetRedisClient()
	if redisClient != nil {
		key := fmt.Sprintf("auth:token:%s", token)
		data, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			var tokenData TokenData
			if err := json.Unmarshal([]byte(data), &tokenData); err == nil {
				return &tokenData, nil
			}
		}
	}

	// Fallback to in-memory storage
	s.mutex.RLock()
	tokenData, exists := s.memoryStore[token]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	return &tokenData, nil
}

// removeFromMemory removes a token from memory store
func (s *AuthService) removeFromMemory(token string) {
	s.mutex.Lock()
	delete(s.memoryStore, token)
	s.mutex.Unlock()
}

// CleanupExpiredTokens removes expired tokens from memory
// This should be called periodically to prevent memory bloat
func (s *AuthService) CleanupExpiredTokens() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for token, data := range s.memoryStore {
		if now.After(data.ExpiresAt) {
			delete(s.memoryStore, token)
		}
	}
}

// Singleton instance
var (
	authServiceInstance *AuthService
	authServiceOnce     sync.Once
)

// GetAuthService returns the singleton auth service instance
func GetAuthService() *AuthService {
	authServiceOnce.Do(func() {
		authServiceInstance = NewAuthService()

		// Start periodic cleanup of expired tokens
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()

			for range ticker.C {
				authServiceInstance.CleanupExpiredTokens()
			}
		}()
	})

	return authServiceInstance
}
