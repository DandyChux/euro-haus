package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ------------------------------------------------------------
// VehicleSubmission – replaces submission:<id> and submissions:* sets
// ------------------------------------------------------------
type VehicleSubmission struct {
	ID                             string `gorm:"type:uuid;primaryKey"`
	EventID                        string `gorm:"not null;index:idx_submissions_event"`
	EventSlug                      string
	ParticipantName                string `gorm:"not null"`
	ParticipantEmail               string `gorm:"not null;index:idx_submissions_email"`
	ParticipantPhone               string
	VehicleYear                    string
	VehicleMake                    string
	VehicleModel                   string
	VehicleDescription             string
	VehicleModifications           string
	Images                         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	Status                         string         `gorm:"not null;default:pending;index:idx_submissions_status"`
	SubmittedAt                    time.Time      `gorm:"not null"`
	ReviewedAt                     *time.Time
	ReviewedBy                     string
	ReviewNotes                    string
	CheckoutSessionID              string `gorm:"index:idx_submissions_checkout"`
	PaymentIntentID                string
	PriceID                        string `gorm:"index:idx_submissions_price"`
	ApprovalEmailSentAt            *time.Time
	RequiresApproval               bool `gorm:"not null;default:true"`
	ApprovalEmailResent            bool `gorm:"not null;default:false"`
	TicketID                       string `gorm:"index:idx_submissions_ticket"`
	TicketEmailSentAt              *time.Time
	EmailUpdatedAt                 *time.Time
	PreviousEmail                  string
	EmailResentCount               int `gorm:"not null;default:0"`
	RevokedAt                      *time.Time
	RevokedBy                      string
	RevocationReason               string
	CreatedAt                      time.Time `gorm:"autoCreateTime"`
	UpdatedAt                      time.Time `gorm:"autoUpdateTime"`
}

func (VehicleSubmission) TableName() string { return "vehicle_submissions" }

func (v *VehicleSubmission) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		v.ID = id.String()
	}
	if v.SubmittedAt.IsZero() {
		v.SubmittedAt = time.Now()
	}
	return nil
}

// Data Transfer Objects

type VehicleSubmissionDTO struct {
	ID                   string   `json:"id"`
	EventID              string   `json:"event_id"`
	EventSlug            string   `json:"event_slug"`
	ParticipantName      string   `json:"participant_name"`
	ParticipantEmail     string   `json:"participant_email"`
	ParticipantPhone     string   `json:"participant_phone,omitempty"`
	VehicleYear          string   `json:"vehicle_year"`
	VehicleMake          string   `json:"vehicle_make"`
	VehicleModel         string   `json:"vehicle_model"`
	VehicleDescription   string   `json:"vehicle_description,omitempty"`
	VehicleModifications string   `json:"vehicle_modifications,omitempty"`
	Images               []string `json:"images"`
	Status               string   `json:"status"`
	SubmittedAt          string   `json:"submitted_at"`
	ReviewedAt           string   `json:"reviewed_at,omitempty"`
	ReviewedBy           string   `json:"reviewed_by,omitempty"`
	ReviewNotes          string   `json:"review_notes,omitempty"`
	CheckoutSessionID    string   `json:"checkout_session_id,omitempty"`
	PaymentIntentID      string   `json:"payment_intent_id,omitempty"`
	PriceID              string   `json:"price_id,omitempty"`
	TicketID           string `json:"ticket_id,omitempty"`
	CreatedAt            string   `json:"created_at"`
	ApprovalEmailSentAt  string   `json:"approval_email_sent_at,omitempty"`

	// Issue endpoint fields.
	Issues  []string `json:"issues,omitempty"`
	HasIssue bool     `json:"has_issue,omitempty"`

	RequiresApproval  bool   `json:"-"`
	ApprovalEmailResent bool `json:"-"`
	TicketEmailSentAt  string `json:"-"`
	PreviousEmail      string `json:"-"`
	EmailUpdatedAt     string `json:"-"`
	EmailResentCount   int    `json:"-"`

	RevokedAt          string `json:"-"`
	RevokedBy          string `json:"-"`
	RevocationReason   string `json:"-"`

}
