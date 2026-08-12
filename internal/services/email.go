package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/mailgun/mailgun-go/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	emailTemplates     map[string]*template.Template
	emailTemplatesOnce sync.Once
)

const (
	emailJobPollInterval = 2 * time.Second
	emailJobLockTimeout  = 10 * time.Minute
	emailJobMaxAttempts  = 8
)

type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

type EmailMessage struct {
	To           []string          `json:"to"`
	Cc           []string          `json:"cc"`
	Bcc          []string          `json:"bcc"`
	Subject      string            `json:"subject"`
	BodyHTML     string            `json:"body_html"`
	BodyText     string            `json:"body_text"`
	Attachments  []EmailAttachment `json:"attachments"`
	TemplateID   string            `json:"template_id"`
	TemplateData interface{}       `json:"template_data"`
}

// SendEmail sends an email message using Resend API
func SendEmail(msg *EmailMessage) error {
	// Get email configuration from environment
	apiKey := os.Getenv("MAILGUN_API_KEY")
	fromEmail := os.Getenv("MAIL_FROM_ADDRESS")

	if apiKey == "" || fromEmail == "" {
		return fmt.Errorf("missing email configuration: MAILGUN_API_KEY and MAIL_FROM_ADDRESS are required")
	}

	domain := os.Getenv("MAILGUN_DOMAIN")
	if domain == "" {
		return fmt.Errorf("missing email configuration: MAILGUN_DOMAIN is required")
	}

	if msg.TemplateID != "" {
		// Load and parse templates if not done already
		emailTemplatesOnce.Do(func() {
			loadEmailTemplates()
		})

		// Get template
		if tmpl, ok := emailTemplates[msg.TemplateID]; ok {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, msg.TemplateData); err != nil {
				return fmt.Errorf("failed to render email template: %w", err)
			}
			msg.BodyHTML = buf.String()
		}
	}

	if msg.BodyHTML == "" && msg.BodyText == "" {
		return fmt.Errorf("no email body provided")
	}

	// Initialize Mailgun client
	mg := mailgun.NewMailgun(domain, apiKey)

	mgMessage := mailgun.NewMessage(fromEmail, msg.Subject, msg.BodyText)

	if msg.BodyHTML != "" {
		mgMessage.SetHTML(msg.BodyHTML)
	}

	// Add recipients
	for _, to := range msg.To {
		if err := mgMessage.AddRecipient(to); err != nil {
			return fmt.Errorf("failed to add recipient: %s: %w", to, err)
		}
	}

	for _, cc := range msg.Cc {
		mgMessage.AddCC(cc)
	}

	for _, bcc := range msg.Bcc {
		mgMessage.AddBCC(bcc)
	}

	if len(msg.Attachments) > 0 {
		for _, att := range msg.Attachments {
			tempFile, err := os.CreateTemp("", "euro-haus-email-*")
			if err != nil {
				return fmt.Errorf(
					"create temporary attachment file: %w",
					err,
				)
			}

			tempPath := tempFile.Name()

			if _, err := tempFile.Write(att.Data); err != nil {
				tempFile.Close()
				os.Remove(tempPath)

				return fmt.Errorf(
					"write temporary attachment file: %w",
					err,
				)
			}

			if err := tempFile.Close(); err != nil {
				os.Remove(tempPath)

				return fmt.Errorf(
					"close temporary attachment file: %w",
					err,
				)
			}

			mgMessage.AddBufferAttachment(tempPath, att.Data)

			defer os.Remove(tempPath)
		}
	}

	// Create contexxt with a timeout for the API call
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Send the message
	_, id, err := mg.Send(ctx, mgMessage)
	if err != nil {
		return fmt.Errorf("failed to send email via Mailgun: %w", err)
	}

	log.Printf("Email sent via Mailgun w/ ID: %s", id)
	return nil
}

// loadEmailTemplates loads all email templates from the templates directory
func loadEmailTemplates() {
	emailTemplates = make(map[string]*template.Template)

	// Get template directory path (adjust as needed)
	templatesDir := "templates/email"

	// Check if the template directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Printf("Email templates directory not found")
		return
	}

	// Load all template files from the directory
	files, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		log.Printf("Error loading email templates: %v", err)
		return
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		tmpl, err := template.ParseFiles(file)
		if err != nil {
			log.Printf("Error parsing template %s: %v", name, err)
			continue
		}
		emailTemplates[name] = tmpl
		log.Printf("Loaded email template: %s", name)
	}
}

// BuildTicketEmail creates the ticket email message.
//
// It intentionally does not send the email. The caller must enqueue the
// returned message with EnqueueEmail or EnqueueEmailTx.
func BuildTicketEmail(
	email string,
	name string,
	ticketToken string,
	eventDetails map[string]interface{},
	qrCode string,
) (*EmailMessage, error) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	ticketToken = strings.TrimSpace(ticketToken)

	if email == "" {
		return nil, errors.New("ticket email recipient is empty")
	}

	if ticketToken == "" {
		return nil, errors.New("ticket token is empty")
	}

	eventName, _ := eventDetails["name"].(string)
	if strings.TrimSpace(eventName) == "" {
		eventName = "Euro Haus Event"
	}

	quantity := eventDetails["quantity"]

	metadata := make(map[string]string)

	switch value := eventDetails["metadata"].(type) {
	case map[string]string:
		metadata = value

	case map[string]interface{}:
		for key, rawValue := range value {
			if stringValue, ok := rawValue.(string); ok {
				metadata[key] = stringValue
			}
		}
	}

	templateData := map[string]interface{}{
		"CustomerName": name,
		"EventName":    eventName,
		"EventDate":    metadata["event_date"],
		"Location":     metadata["location"],
		"Quantity":     quantity,
		"TicketToken":  ticketToken,
		"QRCode":       qrCode,
	}

	return &EmailMessage{
		To: []string{email},
		Subject: fmt.Sprintf(
			"Your Ticket for %s",
			eventName,
		),
		TemplateID:   "event-ticket",
		TemplateData: templateData,
	}, nil
}

func QueueEmail(
	ctx context.Context,
	submissionID string,
	msg *EmailMessage,
) error {
	db := GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return QueueEmailTx(ctx, tx, submissionID, msg)
	})
}

func QueueEmailTx(
	ctx context.Context,
	tx *gorm.DB,
	submissionID string,
	msg *EmailMessage,
) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}

	if msg == nil {
		return errors.New("email message is nil")
	}

	if len(msg.To) == 0 {
		return errors.New("email message has no recipients")
	}

	if strings.TrimSpace(msg.Subject) == "" {
		return errors.New("email message has no subject")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	job := &models.EmailJob{
		SubmissionID: submissionID,
		EmailType:    emailTypeForMessage(msg),
		Recipient:    strings.Join(msg.To, ","),
		Payload:      payload,
		Status:       models.EmailJobPending,
		AvailableAt:  time.Now().UTC(),
	}

	if err := tx.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("create email job: %w", err)
	}

	return nil
}

func emailTypeForMessage(msg *EmailMessage) string {
	if msg == nil {
		return "unknown"
	}

	switch msg.TemplateID {
	case "submission-received":
		return "submission_received"
	case "submission-approved":
		return "submission_approved"
	case "submission-denied":
		return "submission_denied"
	case "participant-ticket":
		return "participant_ticket"
	case "submission-approved-with-ticket":
		return "submission_approved_with_ticket"
	default:
		return "generic"
	}
}

func StartEmailJobWorker(ctx context.Context) {
	go runEmailJobWorker(ctx)
}

func runEmailJobWorker(ctx context.Context) {
	ticker := time.NewTicker(emailJobPollInterval)
	defer ticker.Stop()

	log.Println("Email job worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Email job worker stopped")
			return

		case <-ticker.C:
			if err := processNextEmailJob(ctx); err != nil {
				log.Printf("Email job worker error: %v", err)
			}
		}
	}
}

func processNextEmailJob(ctx context.Context) error {
	db := GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}

	if err := recoverStaleEmailJobs(ctx, db); err != nil {
		return err
	}

	job, claimed, err := claimNextEmailJob(ctx, db)
	if err != nil {
		return err
	}

	if !claimed {
		return nil
	}

	var msg EmailMessage
	if err := json.Unmarshal(job.Payload, &msg); err != nil {
		return markEmailJobFailed(
			ctx,
			db,
			job.ID,
			fmt.Errorf("decode email payload: %w", err),
		)
	}

	if err := SendEmail(&msg); err != nil {
		return markEmailJobRetry(ctx, db, job, err)
	}

	now := time.Now().UTC()

	if err := db.WithContext(ctx).
		Model(&models.EmailJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"status":     models.EmailJobSent,
			"sent_at":    now,
			"locked_at":  nil,
			"last_error": "",
		}).
		Error; err != nil {
		return fmt.Errorf("mark email job %d as sent: %w", job.ID, err)
	}

	if err := markRelatedEmailSent(ctx, db, job); err != nil {
		log.Printf(
			"Email job %d sent, but related submission state failed: %v",
			job.ID,
			err,
		)
	}

	return nil
}

func claimNextEmailJob(
	ctx context.Context,
	db *gorm.DB,
) (*models.EmailJob, bool, error) {
	var job models.EmailJob

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
				Options:  "SKIP LOCKED",
			}).
			Where(
				"status = ? AND available_at <= ?",
				models.EmailJobPending,
				time.Now().UTC(),
			).
			Order("available_at ASC, id ASC").
			First(&job)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}

		if result.Error != nil {
			return result.Error
		}

		now := time.Now().UTC()

		if err := tx.Model(&models.EmailJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]interface{}{
				"status":    models.EmailJobProcessing,
				"locked_at": now,
				"attempts":  gorm.Expr("attempts + 1"),
			}).
			Error; err != nil {
			return err
		}

		job.Status = models.EmailJobProcessing
		job.LockedAt = &now
		job.Attempts++

		return nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("claim email job: %w", err)
	}

	if job.ID == 0 {
		return nil, false, nil
	}

	return &job, true, nil
}

func recoverStaleEmailJobs(ctx context.Context, db *gorm.DB) error {
	staleBefore := time.Now().UTC().Add(-emailJobLockTimeout)

	return db.WithContext(ctx).
		Model(&models.EmailJob{}).
		Where(
			"status = ? AND locked_at < ?",
			models.EmailJobProcessing,
			staleBefore,
		).
		Updates(map[string]interface{}{
			"status":       models.EmailJobPending,
			"available_at": time.Now().UTC(),
			"locked_at":    nil,
			"last_error":   "recovered stale processing job",
		}).
		Error
}

func markEmailJobRetry(
	ctx context.Context,
	db *gorm.DB,
	job *models.EmailJob,
	sendErr error,
) error {
	if job.Attempts >= emailJobMaxAttempts {
		return markEmailJobFailed(ctx, db, job.ID, sendErr)
	}

	delay := retryDelay(job.Attempts)
	availableAt := time.Now().UTC().Add(delay)

	return db.WithContext(ctx).
		Model(&models.EmailJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]interface{}{
			"status":       models.EmailJobPending,
			"available_at": availableAt,
			"locked_at":    nil,
			"last_error":   sendErr.Error(),
		}).
		Error
}

func markEmailJobFailed(
	ctx context.Context,
	db *gorm.DB,
	jobID uint64,
	sendErr error,
) error {
	return db.WithContext(ctx).
		Model(&models.EmailJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":     models.EmailJobFailed,
			"locked_at":  nil,
			"last_error": sendErr.Error(),
		}).
		Error
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	seconds := math.Pow(2, float64(attempt))
	if seconds > 3600 {
		seconds = 3600
	}

	return time.Duration(seconds) * time.Second
}

func markRelatedEmailSent(
	ctx context.Context,
	db *gorm.DB,
	job *models.EmailJob,
) error {
	if job.SubmissionID == "" {
		return nil
	}

	now := time.Now().UTC()

	switch job.EmailType {
	case "submission_approved", "submission_approved_with_ticket":
		return db.WithContext(ctx).
			Model(&models.VehicleSubmission{}).
			Where("id = ?", job.SubmissionID).
			Updates(map[string]interface{}{
				"approval_email_sent":    true,
				"approval_email_sent_at": now,
			}).
			Error

	case "participant_ticket":
		return db.WithContext(ctx).
			Model(&models.VehicleSubmission{}).
			Where("id = ?", job.SubmissionID).
			Updates(map[string]interface{}{
				"ticket_email_sent":    true,
				"ticket_email_sent_at": now,
			}).
			Error

	default:
		return nil
	}
}
