package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/resend/resend-go/v2"
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
	apiKey := os.Getenv("RESEND_API_KEY")
	fromEmail := os.Getenv("MAIL_FROM_ADDRESS")
	fromName := os.Getenv("MAIL_FROM_NAME")

	if apiKey == "" || fromEmail == "" {
		return fmt.Errorf("missing email configuration: RESEND_API_KEY and MAIL_FROM_ADDRESS are required")
	}

	// Initialize Resend client
	client := resend.NewClient(apiKey)

	// Determine the body based on the inputs
	var bodyHTML string
	var bodyText string

	if msg.TemplateID != "" {
		// Load and parse templates if not done already
		emailTemplatesOnce.Do(func() {
			loadEmailTemplates()
		})

		// Get template
		tmpl, ok := emailTemplates[msg.TemplateID]
		if ok {
			// Execute template
			var renderedHTML bytes.Buffer
			if err := tmpl.Execute(&renderedHTML, msg.TemplateData); err != nil {
				return fmt.Errorf("failed to render email template: %w", err)
			}
			bodyHTML = renderedHTML.String()
		} else {
			return fmt.Errorf("template %s not found", msg.TemplateID)
		}
	} else {
		bodyHTML = msg.BodyHTML
		bodyText = msg.BodyText
	}

	// Ensure we have at least one body
	if bodyHTML == "" && bodyText == "" {
		return fmt.Errorf("no email body provided")
	}

	// Format the "From" field
	var from string
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	} else {
		from = fromEmail
	}

	// Prepare Resend email parameters
	params := &resend.SendEmailRequest{
		From:    from,
		To:      msg.To,
		Subject: msg.Subject,
	}

	// Add optional fields
	if len(msg.Cc) > 0 {
		params.Cc = msg.Cc
	}

	if len(msg.Bcc) > 0 {
		params.Bcc = msg.Bcc
	}

	// Set body content
	if bodyHTML != "" {
		params.Html = bodyHTML
	}
	if bodyText != "" {
		params.Text = bodyText
	}

	// Handle attachments if present
	if len(msg.Attachments) > 0 {
		attachments := make([]*resend.Attachment, len(msg.Attachments))
		for i, att := range msg.Attachments {
			// Encode attachment data to base64
			encodedContent := base64.StdEncoding.EncodeToString(att.Data)

			attachments[i] = &resend.Attachment{
				Filename: att.Filename,
				Content:  []byte(encodedContent),
			}
		}
		params.Attachments = attachments
	}

	// Send the email
	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}

	// Log success
	fmt.Printf("Email sent successfully via Resend. ID: %s, To: %v, Subject: %s", sent.Id, msg.To, msg.Subject)

	return nil
}

// loadEmailTemplates loads all email templates from the templates directory
func loadEmailTemplates() {
	emailTemplates = make(map[string]*template.Template)

	// Get template directory path (adjust as needed)
	templatesDir := "templates/email"

	// Create default templates for tickets
	ticketTemplate := `
	<!DOCTYPE html>
	<html>
	<head>
	    <meta charset="UTF-8">
	    <title>Your Ticket for {{.EventName}}</title>
	    <style>
	        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
	        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
	        .header { text-align: center; margin-bottom: 30px; }
	        .ticket { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin-bottom: 20px; }
	        .qr-code { text-align: center; margin: 20px 0; }
	        .event-details { margin: 20px 0; }
	        .footer { text-align: center; font-size: 12px; color: #777; margin-top: 30px; }
	    </style>
	</head>
	<body>
	    <div class="container">
	        <div class="header">
	            <h1>Your Event Ticket</h1>
	        </div>

	        <div class="ticket">
	            <h2>{{.EventName}}</h2>

	            <div class="event-details">
	                <p><strong>Date:</strong> {{.EventDate}}</p>
	                <p><strong>Location:</strong> {{.Location}}</p>
	                <p><strong>Ticket Quantity:</strong> {{.Quantity}}</p>
	                <p><strong>Ticket ID:</strong> {{.TicketToken}}</p>
	            </div>

	            <div class="qr-code">
	                <p>Scan this QR code at the event for check-in:</p>
	                <img src="data:image/png;base64,{{.QRCode}}" alt="QR Code" style="max-width: 200px;">
	            </div>

	            <p>Please bring this ticket (printed or on your phone) to the event for check-in.</p>
	        </div>

	        <div class="footer">
	            <p>Thank you for your purchase!</p>
	            <p>Euro Haus Events</p>
	            <p>For questions or assistance, please contact us at info@theeurohaus.com</p>
	        </div>
	    </div>
	</body>
	</html>
	`
	// Add the default ticket template
	emailTemplates["event-ticket"] = template.Must(template.New("event-ticket").Parse(ticketTemplate))

	// Check if the template directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Printf("Email templates directory not found, using default templates only")
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
