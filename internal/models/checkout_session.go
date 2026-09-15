package models

import "time"

// CheckoutSession stores local metadata needed while processing a Stripe session.
type CheckoutSession struct {
	SessionID string    `gorm:"primaryKey"`
	Metadata  []byte    `gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (CheckoutSession) TableName() string { return "checkout_sessions" }
