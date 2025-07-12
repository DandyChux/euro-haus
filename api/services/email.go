package services

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// SendEmail sends an email message
func SendEmail(msg *EmailMessage) error {
	// Get email configuration from environment
	smtpHost := os.Getenv("MAIL_HOST")
	smtpPort := os.Getenv("MAIL_PORT")
	smtpUser := os.Getenv("MAIL_USERNAME")
	smtpPass := os.Getenv("MAIL_PASSWORD")
	fromEmail := os.Getenv("MAIL_FROM_ADDRESS")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" || fromEmail == "" {
		return fmt.Errorf("missing email configuration")
	}

	// Determine the body based on the inputs
	var body string
	if msg.TemplateID != "" {
		// Load and parse templates if not done already
		emailTemplatesOnce.Do(func() {
			loadEmailTemplates()
		})

		// Get template
		tmpl, ok := emailTemplates[msg.TemplateID]
		if !ok {
			return fmt.Errorf("email template '%s' not found", msg.TemplateID)
		}

		// Execute template
		var renderedHTML bytes.Buffer
		if err := tmpl.Execute(&renderedHTML, msg.TemplateData); err != nil {
			return fmt.Errorf("failed to render email template: %w", err)
		}

		msg.BodyHTML = renderedHTML.String()
	}

	if msg.BodyHTML != "" {
		body = msg.BodyHTML
	} else if msg.BodyText != "" {
		body = msg.BodyText
	} else {
		return fmt.Errorf("no email body provided")
	}

	// Set up email headers
	from := fromEmail

	// Create the message headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(msg.To, ", ")
	if len(msg.Cc) > 0 {
		headers["Cc"] = strings.Join(msg.Cc, ", ")
	}
	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"

	// Determine if this is a multipart message
	isMultipart := len(msg.Attachments) > 0 || (msg.BodyHTML != "" && msg.BodyText != "")

	// Create the email message
	var messageBuffer bytes.Buffer
	boundary := "eurohaus-email-boundary"

	// Add headers
	for k, v := range headers {
		messageBuffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// If multipart, set content type accordingly
	if isMultipart {
		messageBuffer.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
		messageBuffer.WriteString("\r\n")
	} else if msg.BodyHTML != "" {
		// Single part HTML
		messageBuffer.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		messageBuffer.WriteString("\r\n")
	} else {
		// Single part text
		messageBuffer.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		messageBuffer.WriteString("\r\n")
	}

	// Add message body
	if isMultipart {
		// HTML part
		if msg.BodyHTML != "" {
			messageBuffer.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
			messageBuffer.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			messageBuffer.WriteString("\r\n")
			messageBuffer.WriteString(msg.BodyHTML)
		}

		// Text part
		if msg.BodyText != "" {
			messageBuffer.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
			messageBuffer.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			messageBuffer.WriteString("\r\n")
			messageBuffer.WriteString(msg.BodyText)
		}

		// Attachments
		for _, att := range msg.Attachments {
			messageBuffer.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
			messageBuffer.WriteString(fmt.Sprintf("Content-Type: %s\r\n", att.ContentType))
			messageBuffer.WriteString("Content-Transfer-Encoding: base64\r\n")
			messageBuffer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", att.Filename))
			messageBuffer.WriteString("\r\n")

			encoded := base64.StdEncoding.EncodeToString(att.Data)
			// Add line breaks to the base64 encoded attachment
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				messageBuffer.WriteString(encoded[i:end])
				messageBuffer.WriteString("\r\n")
			}
		}

		// Close the boundary
		messageBuffer.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		// Single part message
		messageBuffer.WriteString(body)
	}

	// Connect to SMTP server
	// auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// Combine recipients for SMTP
	recipients := append([]string{}, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	// Create a new TLS configuration
	tlsConfig := &tls.Config{
		ServerName: smtpHost,
	}

	// Connect to the server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS connection
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set the sender and recipients
	if err = client.Mail(fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range recipients {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send the email
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data connection: %w", err)
	}

	_, err = w.Write(messageBuffer.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data connection: %w", err)
	}

	// Close the connection
	client.Quit()

	log.Printf("Email sent to %v with subject: %s", msg.To, msg.Subject)
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
