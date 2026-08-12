package models

import "time"

// ------------------------------------------------------------
// Ticket – replaces ticket:<token> hashes
// ------------------------------------------------------------
type Ticket struct {
	Token string `gorm:"primaryKey"`

	EventID string `gorm:"type:uuid;not null;index"`

	StripeProductID string
	StripeSessionID string
	StripePaymentIntentID string

	Status        string `gorm:"not null;default:active"`
	CustomerName  string `gorm:"not null"`
	CustomerEmail string `gorm:"not null;index:idx_tickets_email"`
	TicketType    string `gorm:"not null;default:General"`
	Quantity      int    `gorm:"not null;default:1"`

	CheckedIn   bool       `gorm:"not null;default:false"`
	CheckedInAt *time.Time

	Invalidated       bool `gorm:"not null;default:false"`
	InvalidatedReason string
	InvalidatedAt     *time.Time

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}


func (Ticket) TableName() string { return "tickets" }

// TicketInfo contains ticket details returned to the client
type TicketInfo struct {
	Valid         bool   `json:"valid"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	EventID       string `json:"event_id"`
	Quantity      int    `json:"quantity"`
	CheckedIn     bool   `json:"checked_in"`
	CheckedInAt   string `json:"checked_in_at,omitempty"`
	TicketType    string `json:"ticket_type"`
	TicketCode    string `json:"ticket_code"`
}
