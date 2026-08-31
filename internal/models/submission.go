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
	ID string `gorm:"type:varchar(64);primaryKey"`

	EventID   string `gorm:"type:uuid;not null;index:idx_submissions_event"`
	EventSlug string

	ParticipantName  string `gorm:"not null"`
	ParticipantEmail string `gorm:"not null;index:idx_submissions_email"`
	ParticipantPhone string

	VehicleYear          string
	VehicleMake          string
	VehicleModel         string
	VehicleDescription   string
	VehicleModifications string

	Images datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`

	Status      string    `gorm:"not null;default:pending;index:idx_submissions_status"`
	SubmittedAt time.Time `gorm:"not null"`

	ReviewedAt   *time.Time
	ReviewedBy   string
	ReviewNotes  string

	CheckoutSessionID  string     `gorm:"index:idx_submissions_checkout"`
	CheckoutCreatedAt  *time.Time
	CheckoutCompleted  bool
	CheckoutCompletedAt *time.Time

	PaymentIntentID             string
	PaymentSucceededBeforeApproval bool
	PaymentSucceededAt          *time.Time
	PaymentCaptured             bool
	PaymentCapturedAt           *time.Time

	PriceID       string `gorm:"index:idx_submissions_price"`
	PromotionCode string

	RequiresApproval bool
	AwaitingApproval bool

	ApprovalEmailSent   bool
	ApprovalEmailSentAt *time.Time
	ApprovalEmailResent bool

	TicketID          string     `gorm:"index:idx_submissions_ticket"`
	TicketCreatedAt   *time.Time
	TicketEmailSent   bool
	TicketEmailSentAt *time.Time

	EmailUpdatedAt   *time.Time
	PreviousEmail    string
	EmailResentCount int

	RecoveryAttempts  int        `gorm:"not null;default:0"`
	RecoveryLastSentAt *time.Time

	RefundID        string
	RefundAmount    float64
	RefundIssuedAt  *time.Time

	RevokedAt        *time.Time
	RevokedBy        string
	RevocationReason string

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (VehicleSubmission) TableName() string {
	return "vehicle_submissions"
}

func (v *VehicleSubmission) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		v.ID = id.String()
	}

	if v.SubmittedAt.IsZero() {
		v.SubmittedAt = time.Now().UTC()
	}

	return nil
}

type PriceRequirement struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`

	PriceID string `gorm:"type:varchar(255);not null;index" json:"price_id"`

	Key      string `gorm:"type:varchar(100);not null" json:"key"`
	Label    string `gorm:"not null" json:"label"`
	Type     string `gorm:"type:varchar(30);not null" json:"type"`
	Required bool   `gorm:"not null;default:false" json:"required"`

	Options datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb" json:"options"`

	SortOrder int  `gorm:"not null;default:0" json:"sort_order"`
	Active    bool `gorm:"not null;default:true" json:"active"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PriceRequirement) TableName() string {
	return "price_requirements"
}

func (r *PriceRequirement) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		r.ID = id.String()
	}

	return nil
}

type SubmissionRequirementAnswer struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`

	SubmissionID  string `gorm:"type:varchar(64);not null;index:idx_answers_submission" json:"submission_id"`
	RequirementID string `gorm:"type:uuid;not null;index:idx_answers_requirement" json:"requirement_id"`

	Value datatypes.JSON `gorm:"type:jsonb;not null;index:idx_answers_value" json:"value"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Requirement *PriceRequirement `gorm:"foreignKey:RequirementID;references:ID" json:"requirement,omitempty"`
}

func (SubmissionRequirementAnswer) TableName() string {
	return "submission_requirement_answers"
}

func (a *SubmissionRequirementAnswer) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		a.ID = id.String()
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

	Images []string `json:"images"`

	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at"`

	ReviewedAt  string `json:"reviewed_at,omitempty"`
	ReviewedBy  string `json:"reviewed_by,omitempty"`
	ReviewNotes string `json:"review_notes,omitempty"`

	CheckoutSessionID   string `json:"checkout_session_id,omitempty"`
	CheckoutCreatedAt   string `json:"checkout_created_at,omitempty"`
	CheckoutCompleted   bool   `json:"checkout_completed"`
	CheckoutCompletedAt string `json:"checkout_completed_at,omitempty"`

	PaymentIntentID                string `json:"payment_intent_id,omitempty"`
	PaymentSucceededBeforeApproval bool   `json:"payment_succeeded_before_approval"`
	PaymentSucceededAt             string `json:"payment_succeeded_at,omitempty"`
	PaymentCaptured                bool   `json:"payment_captured"`
	PaymentCapturedAt              string `json:"payment_captured_at,omitempty"`

	PriceID       string `json:"price_id,omitempty"`
	PriceNickname string `json:"price_nickname,omitempty"`
	PromotionCode string `json:"promotion_code,omitempty"`

	RequiresApproval bool `json:"requires_approval"`
	AwaitingApproval bool `json:"awaiting_approval"`

	ApprovalEmailSent   bool   `json:"approval_email_sent"`
	ApprovalEmailSentAt string `json:"approval_email_sent_at,omitempty"`
	ApprovalEmailResent bool   `json:"approval_email_resent"`

	TicketID          string `json:"ticket_id,omitempty"`
	TicketType        string `json:"ticket_type,omitempty"`
	TicketCreatedAt   string `json:"ticket_created_at,omitempty"`
	TicketEmailSent   bool   `json:"ticket_email_sent"`
	TicketEmailSentAt string `json:"ticket_email_sent_at,omitempty"`

	CreatedAt string `json:"created_at"`

	EmailUpdatedAt   string `json:"email_updated_at,omitempty"`
	PreviousEmail    string `json:"previous_email,omitempty"`
	EmailResentCount int    `json:"email_resent_count"`

	RecoveryAttempts  int    `json:"recovery_attempts"`
	RecoveryLastSentAt string `json:"recovery_last_sent_at,omitempty"`

	RefundID       string  `json:"refund_id,omitempty"`
	RefundAmount   float64 `json:"refund_amount,omitempty"`
	RefundIssuedAt string  `json:"refund_issued_at,omitempty"`

	RevokedAt        string `json:"revoked_at,omitempty"`
	RevokedBy        string `json:"revoked_by,omitempty"`
	RevocationReason string `json:"revocation_reason,omitempty"`

	Issues  []string `json:"issues,omitempty"`
	HasIssue bool     `json:"has_issue,omitempty"`

	RequirementAnswers []SubmissionRequirementAnswerDTO `json:"requirement_answers,omitempty"`
}

type SubmissionRequirementAnswerDTO struct {
	ID            string `json:"id"`
	RequirementID string `json:"requirement_id"`
	Key           string `json:"key"`
	Label         string `json:"label"`
	Type          string `json:"type"`
	Value         any    `json:"value"`
}
