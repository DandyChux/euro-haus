package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"euro-haus-api/services"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/paymentintent"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// VehicleSubmission represents a participant's vehicle submission
type VehicleSubmission struct {
	ID                   string   `json:"id"`
	EventID              string   `json:"eventId"`
	EventSlug            string   `json:"eventSlug"`
	ParticipantName      string   `json:"participantName"`
	ParticipantEmail     string   `json:"participantEmail"`
	ParticipantPhone     string   `json:"participantPhone,omitempty"`
	VehicleYear          string   `json:"vehicleYear"`
	VehicleMake          string   `json:"vehicleMake"`
	VehicleModel         string   `json:"vehicleModel"`
	VehicleDescription   string   `json:"vehicleDescription,omitempty"`
	VehicleModifications string   `json:"vehicleModifications,omitempty"`
	Images               []string `json:"images"`
	Status               string   `json:"status"`
	SubmittedAt          string   `json:"submittedAt"`
	ReviewedAt           string   `json:"reviewedAt,omitempty"`
	ReviewedBy           string   `json:"reviewedBy,omitempty"`
	ReviewNotes          string   `json:"reviewNotes,omitempty"`
	CheckoutSessionID    string   `json:"checkoutSessionId,omitempty"`
	PaymentIntentID      string   `json:"paymentIntentId,omitempty"`
	TicketID             string   `json:"ticketId,omitempty"`
	TicketTier           string   `json:"ticketTier,omitempty"`
	TicketPrice          float64  `json:"ticketPrice,omitempty"`
	TicketQuantity       int      `json:"ticketQuantity,omitempty"`
}

// CreateSubmission handles vehicle submission with image uploads
func CreateSubmission(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 50MB)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form fields
	submission := VehicleSubmission{
		ID:                   generateSubmissionID(),
		EventID:              r.FormValue("eventId"),
		EventSlug:            r.FormValue("eventSlug"),
		ParticipantName:      r.FormValue("participantName"),
		ParticipantEmail:     r.FormValue("participantEmail"),
		ParticipantPhone:     r.FormValue("participantPhone"),
		VehicleYear:          r.FormValue("vehicleYear"),
		VehicleMake:          r.FormValue("vehicleMake"),
		VehicleModel:         r.FormValue("vehicleModel"),
		VehicleDescription:   r.FormValue("vehicleDescription"),
		VehicleModifications: r.FormValue("vehicleModifications"),
		TicketTier:           r.FormValue("ticketTier"),
		Status:               "pending",
		SubmittedAt:          time.Now().Format(time.RFC3339),
		Images:               []string{},
	}

	// Parse ticket price if provided
	if priceStr := r.FormValue("ticketPrice"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			submission.TicketPrice = price
		}
	}

	// Parse ticket quantity if provided
	if qtyStr := r.FormValue("ticketQuantity"); qtyStr != "" {
		if qty, err := strconv.Atoi(qtyStr); err == nil {
			submission.TicketQuantity = qty
		}
	}

	// Validate required fields
	if submission.EventID == "" || submission.ParticipantName == "" || submission.ParticipantEmail == "" ||
		submission.VehicleYear == "" || submission.VehicleMake == "" || submission.VehicleModel == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Handle image uploads
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "At least one image is required", http.StatusBadRequest)
		return
	}

	// Upload images to Spaces
	for i, fileHeader := range files {
		if i >= 5 { // Limit to 5 images
			break
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Error opening file: %v", err)
			continue
		}
		defer file.Close()

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			log.Printf("Error reading file: %v", err)
			continue
		}

		// Determine content type
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg" // Default
		}

		// Generate folder path: events/<event_slug>/<submission_id>/
		folder := fmt.Sprintf("events/%s/%s/", submission.EventSlug, submission.ID)

		// Upload to Spaces
		imageURL, err := services.UploadFile(
			strings.NewReader(string(fileBytes)),
			fileHeader.Filename,
			contentType,
			folder,
		)
		if err != nil {
			log.Printf("Error uploading image: %v", err)
			continue
		}

		submission.Images = append(submission.Images, imageURL)
	}

	if len(submission.Images) == 0 {
		http.Error(w, "Failed to upload any images", http.StatusInternalServerError)
		return
	}

	// Store submission in Redis
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Store submission data
	submissionKey := fmt.Sprintf("submission:%s", submission.ID)
	submissionData := map[string]interface{}{
		"id":                    submission.ID,
		"event_id":              submission.EventID,
		"event_slug":            submission.EventSlug,
		"participant_name":      submission.ParticipantName,
		"participant_email":     submission.ParticipantEmail,
		"participant_phone":     submission.ParticipantPhone,
		"vehicle_year":          submission.VehicleYear,
		"vehicle_make":          submission.VehicleMake,
		"vehicle_model":         submission.VehicleModel,
		"vehicle_description":   submission.VehicleDescription,
		"vehicle_modifications": submission.VehicleModifications,
		"images":                strings.Join(submission.Images, ","),
		"status":                submission.Status,
		"submitted_at":          submission.SubmittedAt,
		"ticket_tier":           submission.TicketTier,
		"ticket_price":          fmt.Sprintf("%.2f", submission.TicketPrice),
		"ticket_quantity":       strconv.Itoa(submission.TicketQuantity),
	}

	if err := rdb.HSet(ctx, submissionKey, submissionData).Err(); err != nil {
		http.Error(w, "Failed to store submission", http.StatusInternalServerError)
		return
	}

	// Add submission ID to event's submission set
	eventSubmissionsKey := fmt.Sprintf("event:%s:submissions", submission.EventID)
	if err := rdb.SAdd(ctx, eventSubmissionsKey, submission.ID).Err(); err != nil {
		log.Printf("Failed to add submission to event set: %v", err)
	}

	// Send confirmation email
	go sendSubmissionConfirmationEmail(submission)

	// Return the created submission
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

// GetEventSubmissions retrieves all submissions for an event
func GetEventSubmissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["eventId"]

	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get all submission IDs for this event
	eventSubmissionsKey := fmt.Sprintf("event:%s:submissions", eventID)
	submissionIDs, err := rdb.SMembers(ctx, eventSubmissionsKey).Result()
	if err != nil {
		http.Error(w, "Failed to retrieve submissions", http.StatusInternalServerError)
		return
	}

	// Retrieve each submission
	submissions := []VehicleSubmission{}
	for _, submissionID := range submissionIDs {
		submission, err := getSubmissionByID(submissionID)
		if err != nil {
			log.Printf("Error retrieving submission %s: %v", submissionID, err)
			continue
		}
		submissions = append(submissions, *submission)
	}

	// Return submissions
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": submissions,
	})
}

// GetSubmission retrieves a single submission by ID
func GetSubmission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

// ApproveSubmission approves a vehicle submission and captures payment
func ApproveSubmission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Notes are optional, so we can continue
	}

	// Get submission
	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Get the checkout session to find the payment intent
	if submission.CheckoutSessionID != "" {
		// Retrieve the checkout session
		sess, err := session.Get(submission.CheckoutSessionID, nil)
		if err != nil {
			log.Printf("Error retrieving checkout session: %v", err)
		} else if sess.PaymentIntent != nil {
			// Capture the payment intent
			pi, err := paymentintent.Capture(sess.PaymentIntent.ID, nil)
			if err != nil {
				log.Printf("Error capturing payment: %v", err)
				http.Error(w, "Failed to capture payment", http.StatusInternalServerError)
				return
			}

			// Update submission with payment intent ID
			submission.PaymentIntentID = pi.ID
		}
	}

	// Update submission status
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	updates := map[string]interface{}{
		"status":            "approved",
		"reviewed_at":       time.Now().Format(time.RFC3339),
		"reviewed_by":       "admin", // TODO: Get from auth context
		"review_notes":      req.Notes,
		"payment_intent_id": submission.PaymentIntentID,
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	// Remove from pending set
	rdb.SRem(ctx, "submissions:pending", submissionID)

	// Add to approved set
	rdb.SAdd(ctx, "submissions:approved", submissionID)

	// The webhook will handle the rest (ticket creation, email sending)
	// when it receives the payment_intent.succeeded event

	// Get updated submission
	updatedSubmission, _ := getSubmissionByID(submissionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedSubmission)
}

// DenySubmission denies a vehicle submission
func DenySubmission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Review notes are required for denial", http.StatusBadRequest)
		return
	}

	// Get submission
	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Update submission status
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	updates := map[string]interface{}{
		"status":       "denied",
		"reviewed_at":  time.Now().Format(time.RFC3339),
		"reviewed_by":  "admin", // TODO: Get from auth context
		"review_notes": req.Notes,
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	// Remove from pending set
	rdb.SRem(ctx, "submissions:pending", submissionID)

	// Add to denied set
	rdb.SAdd(ctx, "submissions:denied", submissionID)

	// Send denial email
	go sendDenialEmail(*submission, req.Notes)

	// Get updated submission
	updatedSubmission, _ := getSubmissionByID(submissionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedSubmission)
}

// GetPendingSubmissionsCount returns the count of pending submissions
func GetPendingSubmissionsCount(w http.ResponseWriter, r *http.Request) {
	rdb := services.GetRedisClient()
	ctx := context.Background()

	count, err := rdb.SCard(ctx, "submissions:pending").Result()
	if err != nil {
		count = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": count,
	})
}

// CreateSubmissionCheckout creates a checkout session for an approved submission
func CreateSubmissionCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubmissionID string `json:"submissionId"`
		PriceID      string `json:"priceId"`
		EventName    string `json:"eventName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get submission
	submission, err := getSubmissionByID(req.SubmissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Verify submission is approved
	if submission.Status != "approved" {
		http.Error(w, "Submission is not approved", http.StatusBadRequest)
		return
	}

	// Create checkout session with manual capture
	baseUrl := os.Getenv("BASE_URL")
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", baseUrl)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/checkout/cancel", baseUrl)),
		Metadata: map[string]string{
			"submission_id": req.SubmissionID,
			"event_id":      submission.EventID,
			"event_name":    req.EventName,
			"participant":   "true",
		},
		CustomerEmail: stripe.String(submission.ParticipantEmail),
	}

	s, err := session.New(params)
	if err != nil {
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	// Store checkout session ID with submission
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", req.SubmissionID)
	rdb.HSet(ctx, submissionKey, "checkout_session_id", s.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionUrl": s.URL,
	})
}

// CreateParticipantCheckout creates a checkout session with manual capture for submissions
func CreateParticipantCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubmissionID string `json:"submissionId"`
		PriceID      string `json:"priceId"`
		EventName    string `json:"eventName"`
		Quantity     int64  `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get submission
	submission, err := getSubmissionByID(req.SubmissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Create checkout session with manual capture
	baseUrl := os.Getenv("BASE_URL")
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(req.Quantity),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s/checkout/pending?submission_id=%s", baseUrl, req.SubmissionID)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/checkout/cancel", baseUrl)),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			CaptureMethod: stripe.String("manual"),
			Metadata: map[string]string{
				"submission_id": req.SubmissionID,
				"event_id":      submission.EventID,
				"event_name":    req.EventName,
				"participant":   "true",
			},
		},
		Metadata: map[string]string{
			"submission_id": req.SubmissionID,
			"event_id":      submission.EventID,
			"event_name":    req.EventName,
			"participant":   "true",
		},
		CustomerEmail: stripe.String(submission.ParticipantEmail),
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Failed to create checkout session: %v", err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	// Store checkout session ID with submission
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", req.SubmissionID)
	rdb.HSet(ctx, submissionKey, "checkout_session_id", s.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":  s.ID,
		"sessionUrl": s.URL,
	})
}

// Helper functions

func generateSubmissionID() string {
	timestamp := time.Now().Unix()
	randomStr := generateRandomString(6)
	return fmt.Sprintf("SUB-%d-%s", timestamp, randomStr)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func getSubmissionByID(submissionID string) (*VehicleSubmission, error) {
	rdb := services.GetRedisClient()
	ctx := context.Background()

	submissionKey := fmt.Sprintf("submission:%s", submissionID)
	data, err := rdb.HGetAll(ctx, submissionKey).Result()
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("submission not found")
	}

	// Parse images
	images := []string{}
	if data["images"] != "" {
		images = strings.Split(data["images"], ",")
	}

	// Parse ticket price
	var ticketPrice float64
	if data["ticket_price"] != "" {
		ticketPrice, _ = strconv.ParseFloat(data["ticket_price"], 64)
	}

	// Parse ticket quantity
	var ticketQuantity int
	if data["ticket_quantity"] != "" {
		ticketQuantity, _ = strconv.Atoi(data["ticket_quantity"])
	}

	submission := &VehicleSubmission{
		ID:                   data["id"],
		EventID:              data["event_id"],
		EventSlug:            data["event_slug"],
		ParticipantName:      data["participant_name"],
		ParticipantEmail:     data["participant_email"],
		ParticipantPhone:     data["participant_phone"],
		VehicleYear:          data["vehicle_year"],
		VehicleMake:          data["vehicle_make"],
		VehicleModel:         data["vehicle_model"],
		VehicleDescription:   data["vehicle_description"],
		VehicleModifications: data["vehicle_modifications"],
		Images:               images,
		Status:               data["status"],
		SubmittedAt:          data["submitted_at"],
		ReviewedAt:           data["reviewed_at"],
		ReviewedBy:           data["reviewed_by"],
		ReviewNotes:          data["review_notes"],
		CheckoutSessionID:    data["checkout_session_id"],
		PaymentIntentID:      data["payment_intent_id"],
		TicketID:             data["ticket_id"],
		TicketTier:           data["ticket_tier"],
		TicketPrice:          ticketPrice,
		TicketQuantity:       ticketQuantity,
	}

	return submission, nil
}

func createSubmissionPaymentLink(submission VehicleSubmission) (string, error) {
	// This creates a payment link that the participant can use to complete their purchase
	// You might want to create a Stripe Payment Link or generate a custom checkout URL
	baseUrl := os.Getenv("BASE_URL")
	if baseUrl == "" {
		baseUrl = os.Getenv("WEBSITE_URL")
	}

	// For now, return a link to the event page with submission ID
	paymentLink := fmt.Sprintf("%s/events/%s?submission=%s", baseUrl, submission.EventSlug, submission.ID)
	return paymentLink, nil
}

// Email functions

func sendSubmissionConfirmationEmail(submission VehicleSubmission) {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails":  fmt.Sprintf("%s %s %s", submission.VehicleYear, submission.VehicleMake, submission.VehicleModel),
		"EventID":         submission.EventID,
		"SubmissionID":    submission.ID,
		"SubmittedAt":     submission.SubmittedAt,
	}

	msg := &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Received - Euro Haus",
		TemplateID:   "submission-received",
		TemplateData: emailData,
		BodyHTML:     generateSubmissionConfirmationHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending submission confirmation email: %v", err)
	}
}

func sendApprovalEmail(submission VehicleSubmission, paymentLink string) {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails":  fmt.Sprintf("%s %s %s", submission.VehicleYear, submission.VehicleMake, submission.VehicleModel),
		"EventID":         submission.EventID,
		"PaymentLink":     paymentLink,
		"ReviewNotes":     submission.ReviewNotes,
	}

	msg := &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Approved - Complete Your Registration",
		TemplateID:   "submission-approved",
		TemplateData: emailData,
		BodyHTML:     generateApprovalEmailHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending approval email: %v", err)
	}
}

func sendDenialEmail(submission VehicleSubmission, reason string) {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails":  fmt.Sprintf("%s %s %s", submission.VehicleYear, submission.VehicleMake, submission.VehicleModel),
		"EventID":         submission.EventID,
		"DenialReason":    reason,
	}

	msg := &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Update - Euro Haus",
		TemplateID:   "submission-denied",
		TemplateData: emailData,
		BodyHTML:     generateDenialEmailHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending denial email: %v", err)
	}
}

// HTML email templates
func generateSubmissionConfirmationHTML(data map[string]interface{}) string {
	return fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; color: #333;">
			<h2>Vehicle Submission Received</h2>
			<p>Dear %s,</p>
			<p>Thank you for submitting your vehicle for our event. We have received your submission for:</p>
			<p><strong>%s</strong></p>
			<p>Your submission ID is: <strong>%s</strong></p>
			<p>We will review your submission and notify you within 48 hours. Once approved, you'll receive a link to complete your registration and payment.</p>
			<p>If you have any questions, please don't hesitate to contact us.</p>
			<p>Best regards,<br>The Euro Haus Events Team</p>
		</body>
		</html>
	`, data["ParticipantName"], data["VehicleDetails"], data["SubmissionID"])
}

func generateApprovalEmailHTML(data map[string]interface{}) string {
	reviewNotes := ""
	if notes, ok := data["ReviewNotes"].(string); ok && notes != "" {
		reviewNotes = fmt.Sprintf("<p><strong>Admin Notes:</strong> %s</p>", notes)
	}

	return fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; color: #333;">
			<h2>Vehicle Submission Approved!</h2>
			<p>Dear %s,</p>
			<p>Great news! Your vehicle submission has been approved:</p>
			<p><strong>%s</strong></p>
			%s
			<p>To complete your registration and secure your spot at the event, please click the link below:</p>
			<p><a href="%s" style="display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">Complete Registration</a></p>
			<p>This link will expire in 72 hours. If you need assistance, please contact us.</p>
			<p>We look forward to seeing you at the event!</p>
			<p>Best regards,<br>The Euro Haus Events Team</p>
		</body>
		</html>
	`, data["ParticipantName"], data["VehicleDetails"], reviewNotes, data["PaymentLink"])
}

func generateDenialEmailHTML(data map[string]interface{}) string {
	return fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; color: #333;">
			<h2>Vehicle Submission Update</h2>
			<p>Dear %s,</p>
			<p>Thank you for your interest in participating in our event with your:</p>
			<p><strong>%s</strong></p>
			<p>After careful review, we regret to inform you that we are unable to approve your submission at this time.</p>
			<p><strong>Reason:</strong> %s</p>
			<p>We encourage you to consider attending as a general participant. You can still purchase regular event tickets on our website.</p>
			<p>If you have questions or would like to discuss this decision, please don't hesitate to contact us.</p>
			<p>Best regards,<br>The Euro Haus Events Team</p>
		</body>
		</html>
	`, data["ParticipantName"], data["VehicleDetails"], data["DenialReason"])
}
