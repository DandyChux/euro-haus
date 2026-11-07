package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
)

type EmailJobSummary struct {
	ID           uint64     `json:"id"`
	SubmissionID string     `json:"submission_id,omitempty"`
	EmailType    string     `json:"email_type"`
	Recipient    string     `json:"recipient"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	LastError    string     `json:"last_error,omitempty"`
	Subject      string     `json:"subject,omitempty"`
	AvailableAt  time.Time  `json:"available_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type EmailJobDetailResponse struct {
	EmailJobSummary
	Message services.EmailMessage `json:"message"`
}

// GetEmailJobs lists email jobs with filtering and search
func GetEmailJobs(w http.ResponseWriter, r *http.Request) {
	db := services.GetDB()
	if db == nil {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	query := db.WithContext(r.Context()).Model(&models.EmailJob{})

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(recipient) LIKE ? OR LOWER(submission_id) LIKE ? OR LOWER(email_type) LIKE ?",
			searchTerm, searchTerm, searchTerm,
		)
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var jobs []models.EmailJob
	if err := query.Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to list email jobs: %v", err), http.StatusInternalServerError)
		return
	}

	summaries := make([]EmailJobSummary, 0, len(jobs))
	for _, job := range jobs {
		var msg services.EmailMessage
		_ = json.Unmarshal(job.Payload, &msg)

		summaries = append(summaries, EmailJobSummary{
			ID:           job.ID,
			SubmissionID: job.SubmissionID,
			EmailType:    job.EmailType,
			Recipient:    job.Recipient,
			Status:       string(job.Status),
			Attempts:     job.Attempts,
			LastError:    job.LastError,
			Subject:      msg.Subject,
			AvailableAt:  job.AvailableAt,
			SentAt:       job.SentAt,
			CreatedAt:    job.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": summaries,
	})
}

// GetEmailJobByID retrieves full details including message payload
func GetEmailJobByID(w http.ResponseWriter, r *http.Request) {
	jobIDStr := mux.Vars(r)["id"]
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid email job ID", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	var job models.EmailJob
	if err := db.WithContext(r.Context()).First(&job, "id = ?", jobID).Error; err != nil {
		http.Error(w, "Email job not found", http.StatusNotFound)
		return
	}

	var msg services.EmailMessage
	if err := json.Unmarshal(job.Payload, &msg); err != nil {
		http.Error(w, "Failed to parse email message payload", http.StatusInternalServerError)
		return
	}

	resp := EmailJobDetailResponse{
		EmailJobSummary: EmailJobSummary{
			ID:           job.ID,
			SubmissionID: job.SubmissionID,
			EmailType:    job.EmailType,
			Recipient:    job.Recipient,
			Status:       string(job.Status),
			Attempts:     job.Attempts,
			LastError:    job.LastError,
			Subject:      msg.Subject,
			AvailableAt:  job.AvailableAt,
			SentAt:       job.SentAt,
			CreatedAt:    job.CreatedAt,
		},
		Message: msg,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ResendEmailJob duplicates the row with modifications and sets status to pending
func ResendEmailJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := mux.Vars(r)["id"]
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid email job ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Recipient        string `json:"recipient"`
		Subject          string `json:"subject"`
		BodyHTML         string `json:"body_html"`
		BodyText         string `json:"body_text"`
		UpdateSubmission bool   `json:"update_submission"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	var original models.EmailJob
	if err := db.WithContext(r.Context()).First(&original, "id = ?", jobID).Error; err != nil {
		http.Error(w, "Original email job not found", http.StatusNotFound)
		return
	}

	var msg services.EmailMessage
	if err := json.Unmarshal(original.Payload, &msg); err != nil {
		http.Error(w, "Failed to unpack email payload", http.StatusInternalServerError)
		return
	}

	// 1. Update recipient
	targetRecipient := strings.TrimSpace(req.Recipient)
	if targetRecipient == "" {
		targetRecipient = original.Recipient
	}
	msg.To = []string{targetRecipient}

	// 2. Update subject if provided
	if strings.TrimSpace(req.Subject) != "" {
		msg.Subject = strings.TrimSpace(req.Subject)
	}

	// 3. Update body HTML / Text if provided
	if req.BodyHTML != "" {
		msg.BodyHTML = req.BodyHTML
		// Crucial: Clear TemplateID so SendEmail does not overwrite custom HTML
		msg.TemplateID = ""
	}
	if req.BodyText != "" {
		msg.BodyText = req.BodyText
	}

	newPayload, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, "Failed to re-encode email payload", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	// 4. Replicate non-timestamp fields; set timestamps to NOW()
	newJob := models.EmailJob{
		SubmissionID: original.SubmissionID,
		EmailType:    original.EmailType,
		Recipient:    targetRecipient,
		Payload:      datatypes.JSON(newPayload),
		Status:       models.EmailJobPending,
		Attempts:     0,
		LastError:    "",
		AvailableAt:  now,
		LockedAt:     nil,
		SentAt:       nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newJob).Error; err != nil {
			return fmt.Errorf("create duplicated email job: %w", err)
		}

		// Optional: Keep submission and ticket email addresses in sync with the corrected recipient
		if req.UpdateSubmission && original.SubmissionID != "" {
			_ = tx.Exec(`
				UPDATE vehicle_submissions
				SET participant_email = ?,
					previous_email = CASE WHEN participant_email != ? THEN participant_email ELSE previous_email END,
					email_updated_at = NOW()
				WHERE id = ?
			`, targetRecipient, targetRecipient, original.SubmissionID).Error

			_ = tx.Exec(`
				UPDATE tickets
				SET customer_email = ?
				WHERE token = (SELECT ticket_id FROM vehicle_submissions WHERE id = ?)
			`, targetRecipient, original.SubmissionID).Error
		}

		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to queue email resend: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Email job duplicated and queued for immediate delivery",
		"job_id":  newJob.ID,
	})
}
