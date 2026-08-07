package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Email        string `gorm:"type:varchar(320);not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"type:text;not null" json:"-"`

	Name        string `gorm:"type:varchar(200);not null" json:"name"`
	Phone       string `gorm:"type:varchar(50)" json:"phone"`
	AddressLine1 string `gorm:"type:varchar(255)" json:"address_line1"`
	AddressLine2 string `gorm:"type:varchar(255)" json:"address_line2"`
	City        string `gorm:"type:varchar(100)" json:"city"`
	State       string `gorm:"type:varchar(100)" json:"state"`
	PostalCode  string `gorm:"type:varchar(30)" json:"postal_code"`
	Country     string `gorm:"type:varchar(2);default:US" json:"country"`

	Role   string `gorm:"type:varchar(30);not null;default:customer;index" json:"role"`
	Active bool   `gorm:"not null;default:true" json:"active"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) Normalize() {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
}

func (User) TableName() string {
	return "users"
}

// TokenData represents the data stored for a token
type TokenData struct {
	Token     string    `gorm:"primaryKey" json:"token"`
	UserID    uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TableName defines the table name for Gorm
func (TokenData) TableName() string {
	return "auth_tokens"
}
