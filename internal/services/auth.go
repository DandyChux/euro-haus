package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidUser        = errors.New("invalid user")
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	return nil
}

// AuthService handles authentication operations
type AuthService struct {
	tokenTTL      time.Duration
	memoryStore   map[string]models.TokenData
	mutex         sync.RWMutex
}

func NewAuthService() *AuthService {
	return &AuthService{
		tokenTTL:    24 * time.Hour,
		memoryStore: make(map[string]models.TokenData),
	}
}

type RegisterRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
}

func (s *AuthService) RegisterUser(
	ctx context.Context,
	req *RegisterRequest,
) (*models.User, error) {
	return s.createUser(ctx, req, "customer")
}

func (s *AuthService) CreateAdminUser(
	ctx context.Context,
	req *RegisterRequest,
) (*models.User, error) {
	return s.createUser(ctx, req, "admin")
}

func (s *AuthService) AuthenticateUser(
	ctx context.Context,
	email string,
	password string,
) (*models.User, error) {
	email = normalizeEmail(email)

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var user models.User
	if err := db.WithContext(ctx).
		Where("email = ? AND active = TRUE", email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

// GenerateToken creates a new access token
func (s *AuthService) GenerateToken(ctx context.Context, user *models.User) (string, error) {
	if user == nil || user.ID == uuid.Nil {
		return "", ErrInvalidUser
	}

	return s.generateToken(ctx, &user.ID)
}

func (s *AuthService) ValidateToken(token string) bool {
	token = strings.TrimSpace(token)

	if token == "" {
		return false
	}

	tokenData, err := s.getTokenData(context.Background(), token)
	if err != nil {
		return false
	}

	if time.Now().UTC().After(tokenData.ExpiresAt) {
		_ = s.RevokeToken(token)
		return false
	}

	return true
}

func (s *AuthService) GetTokenUser(
	ctx context.Context,
	token string,
) (*models.User, error) {
	token = strings.TrimSpace(token)

	if token == "" {
		return nil, ErrInvalidCredentials
	}

	tokenInfo, err := s.getTokenData(ctx, token)
	if err != nil {
		return nil, err
	}

	if time.Now().UTC().After(tokenInfo.ExpiresAt) {
		_ = s.RevokeToken(token)
		return nil, ErrInvalidCredentials
	}

	if tokenInfo.UserID == uuid.Nil {
		return nil, ErrInvalidUser
	}

	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var user models.User
	if err := db.WithContext(ctx).
		Where("id = ? AND active = TRUE", tokenInfo.UserID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find token user: %w", err)
	}

	return &user, nil
}

// RevokeToken removes a token from storage
func (s *AuthService) RevokeToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	db := GetDB()
	if db != nil {
		ctx := context.Background()
		err := db.WithContext(ctx).Exec(`DELETE FROM auth_tokens WHERE token = ?`, token).Error
		if err != nil {
			log.Printf("Failed to delete token from database: %v", err)
		}
	}

	s.removeFromMemory(token)
	return nil
}

// ExtendToken extends the expiration time of a valid token
func (s *AuthService) ExtendToken(token string) error {
	if !s.ValidateToken(token) {
		return fmt.Errorf("invalid or expired token")
	}

	newExpiresAt := time.Now().Add(s.tokenTTL)

	// Update database
	db := GetDB()
	if db != nil {
		ctx := context.Background()
		err := db.WithContext(ctx).Exec(
			`UPDATE auth_tokens SET expires_at = ? WHERE token = ?`,
			newExpiresAt, token,
		).Error
		if err != nil {
			log.Printf("Failed to extend token in database: %v", err)
		}
	}

	// Update memory
	s.mutex.Lock()
	if td, exists := s.memoryStore[token]; exists {
		td.ExpiresAt = newExpiresAt
		s.memoryStore[token] = td
	}
	s.mutex.Unlock()

	return nil
}

func (s *AuthService) GetTokenInfo(token string) (*models.TokenData, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	return s.getTokenData(context.Background(), strings.TrimSpace(token))
}

// CleanupExpiredTokens removes expired tokens
func (s *AuthService) CleanupExpiredTokens() {
	db := GetDB()
	if db != nil {
		ctx := context.Background()
		err := db.WithContext(ctx).Exec(`DELETE FROM auth_tokens WHERE expires_at < NOW()`).Error
		if err != nil {
			log.Printf("Failed to clean up expired tokens from database: %v", err)
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	for token, data := range s.memoryStore {
		if now.After(data.ExpiresAt) {
			delete(s.memoryStore, token)
		}
	}
}

func (s *AuthService) createUser(
	ctx context.Context,
	req *RegisterRequest,
	role string,
) (*models.User, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	country := strings.ToUpper(strings.TrimSpace(req.Country))
	if country == "" {
		country = "US"
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		Name:         name,
		Phone:        strings.TrimSpace(req.Phone),
		AddressLine1: strings.TrimSpace(req.AddressLine1),
		AddressLine2: strings.TrimSpace(req.AddressLine2),
		City:         strings.TrimSpace(req.City),
		State:        strings.TrimSpace(req.State),
		PostalCode:   strings.TrimSpace(req.PostalCode),
		Country:      country,
		Role:         role,
		Active:       true,
	}

	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
			strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
			return nil, ErrEmailAlreadyExists
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) removeFromMemory(token string) {
	s.mutex.Lock()
	delete(s.memoryStore, token)
	s.mutex.Unlock()
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

func (s *AuthService) generateToken(
	ctx context.Context,
	userID *uuid.UUID,
) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenTTL)

	var storedUserID uuid.UUID
	if userID != nil {
		storedUserID = *userID
	}

	tokenData := models.TokenData{
		Token:     token,
		UserID:    storedUserID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	db := GetDB()
		if db != nil {
			err := db.WithContext(ctx).Exec(`
				INSERT INTO auth_tokens
					(token, user_id, created_at, expires_at)
				VALUES (
					?,
					NULLIF(?::uuid, '00000000-0000-0000-0000-000000000000'::uuid),
					?,
					?
				)
				ON CONFLICT (token) DO NOTHING
			`,
				token,
				storedUserID.String(),
				now,
				expiresAt,
			).Error
			if err != nil {
				return "", fmt.Errorf("store auth token: %w", err)
			}
		}

	s.mutex.Lock()
	s.memoryStore[token] = tokenData
	s.mutex.Unlock()

	return token, nil
}

func (s *AuthService) getTokenData(
	ctx context.Context,
	token string,
) (*models.TokenData, error) {
	db := GetDB()
	if db != nil {
		var tokenData models.TokenData

		err := db.WithContext(ctx).
			Where("token = ?", token).
			First(&tokenData).
			Error

		if err == nil {
			return &tokenData, nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	s.mutex.RLock()
	tokenData, exists := s.memoryStore[token]
	s.mutex.RUnlock()

	if !exists {
		return nil, ErrInvalidCredentials
	}

	return &tokenData, nil
}
