package models

import (
	"time"

	"gorm.io/datatypes"
)

type EmailJobStatus string

const (
	EmailJobPending    EmailJobStatus = "pending"
	EmailJobProcessing EmailJobStatus = "processing"
	EmailJobSent       EmailJobStatus = "sent"
	EmailJobFailed     EmailJobStatus = "failed"
)

type EmailJob struct {
	ID uint64 `gorm:"primaryKey"`

	SubmissionID string `gorm:"type:varchar(64);index:idx_email_jobs_submission"`

	EmailType string `gorm:"type:varchar(64);not null;index:idx_email_jobs_type"`

	Recipient string `gorm:"type:varchar(320);not null"`

	// Serialized services.EmailMessage. Storing the complete message means
	// the worker does not need to reconstruct submission state later.
	Payload datatypes.JSON `gorm:"type:jsonb;not null"`

	Status EmailJobStatus `gorm:"type:varchar(32);not null;default:pending;index:idx_email_jobs_status"`

	Attempts  int `gorm:"not null;default:0"`
	LastError string

	AvailableAt time.Time `gorm:"not null;index:idx_email_jobs_available"`
	LockedAt    *time.Time
	SentAt      *time.Time

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (EmailJob) TableName() string {
	return "email_jobs"
}
