package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

var (
	emailTemplates     map[string]*template.Template
	emailTemplatesOnce sync.Once
)

// EmailAttachment represents a file attachment in an email
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailMessage represents an email message
type EmailMessage struct {
	To           []string
	Cc           []string
	Bcc          []string
	Subject      string
	BodyHTML     string
	BodyText     string
	Attachments  []EmailAttachment
	TemplateID   string
	TemplateData interface{}
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

	// Handle attachments if present
	if len(msg.Attachments) > 0 {
		for _, att := range msg.Attachments {
			mgMessage.AddAttachment(att.Filename)
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

// SendTicketEmail sends an email with the ticket information
func SendTicketEmail(email string, name string, ticketToken string, eventDetails map[string]interface{}, qrCode string) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"CustomerName": name,
		"EventName":    eventDetails["name"],
		"EventDate":    eventDetails["metadata"].(map[string]string)["event_date"],
		"Location":     eventDetails["metadata"].(map[string]string)["location"],
		"Quantity":     eventDetails["quantity"],
		"TicketToken":  ticketToken,
		"QRCode":       qrCode,
	}

	// Create email message
	msg := &EmailMessage{
		To:           []string{email},
		Subject:      fmt.Sprintf("Your Ticket for %s", eventDetails["name"]),
		TemplateID:   "event-ticket",
		TemplateData: templateData,
	}

	// Send the email
	return SendEmail(msg)
}
