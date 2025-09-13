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
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
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
	ApprovalEmailSent    bool     `json:"approvalEmailSent,omitempty"`
	ApprovalEmailSentAt  string   `json:"approvalEmailSentAt,omitempty"`
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

	// Get the price ID from the request to check metadata
	priceID := r.FormValue("priceId")

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

	// Check if this tier requires approval
	requiresApproval := checkIfTierRequiresApproval(priceID)

	if !requiresApproval {
		// Auto-approve the submission
		submission.Status = "approved"
		submission.ReviewedAt = time.Now().Format(time.RFC3339)
		submission.ReviewedBy = "system-auto-approved"
		submission.ReviewNotes = "Automatically approved - tier does not require manual approval"

		fmt.Printf("Auto-approving submission %s for tier that doesn't require approval", submission.ID)
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

	// Add approval fields if auto-approved
	if !requiresApproval {
		submissionData["reviewed_at"] = submission.ReviewedAt
		submissionData["reviewed_by"] = submission.ReviewedBy
		submissionData["review_notes"] = submission.ReviewNotes
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

	// Send appropriate email
	if !requiresApproval {
		// Generate payment link for auto-approved submission
		paymentLink, err := createSubmissionPaymentLink(submission)
		if err != nil {
			log.Printf("Error creating payment link for auto-approved submission: %v", err)
			// Use fallback link if payment link generation fails
			baseUrl := os.Getenv("BASE_URL")
			if baseUrl == "" {
				baseUrl = "https://eurohaus.shop"
			}
			paymentLink = fmt.Sprintf("%s/events/%s?submission=%s", baseUrl, submission.EventSlug, submission.ID)
		}

		// Send approval email immediately for auto-approved submissions
		go sendApprovalEmail(submission, paymentLink)

		fmt.Printf("Auto-approved submission %s and sent approval email with payment link: %s", submission.ID, paymentLink)
	} else {
		// Send confirmation email for pending submissions
		go sendSubmissionConfirmationEmail(submission)

		fmt.Printf("Created pending submission %s and sent confirmation email", submission.ID)
	}

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

	// Check if already approved
	if submission.Status == "approved" {
		log.Printf("Submission %s is already approved", submissionID)
		http.Error(w, "Submission is already approved", http.StatusBadRequest)
		return
	}

	// Update submission status first
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	updates := map[string]interface{}{
		"status":       "approved",
		"reviewed_at":  time.Now().Format(time.RFC3339),
		"reviewed_by":  "admin", // TODO: Get from auth context
		"review_notes": req.Notes,
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	// Remove from pending set and add to approved set
	rdb.SRem(ctx, "submissions:pending", submissionID)
	rdb.SAdd(ctx, "submissions:approved", submissionID)

	fmt.Printf("Submission %s approved by admin", submissionID)

	// Now handle payment capture if there's a payment intent
	paymentCaptured := false
	paymentProcessing := false

	if submission.CheckoutSessionID != "" {
		// Retrieve the checkout session
		sess, err := session.Get(submission.CheckoutSessionID, nil)
		if err != nil {
			log.Printf("Error retrieving checkout session for submission %s: %v", submissionID, err)
		} else if sess.PaymentIntent != nil {
			// Check if payment intent needs to be captured
			pi, err := paymentintent.Get(sess.PaymentIntent.ID, nil)
			if err != nil {
				log.Printf("Error retrieving payment intent %s: %v", sess.PaymentIntent.ID, err)
			} else {
				// Check the status and capture method
				if pi.Status == "requires_capture" && pi.CaptureMethod == "manual" {
					// Capture the payment
					fmt.Printf("Capturing payment for submission %s (payment intent: %s)", submissionID, pi.ID)
					paymentProcessing = true

					capturedPI, err := paymentintent.Capture(pi.ID, nil)
					if err != nil {
						log.Printf("Error capturing payment for submission %s: %v", submissionID, err)
						// Don't fail the approval, but note the error
						rdb.HSet(ctx, submissionKey, map[string]interface{}{
							"payment_capture_error":        err.Error(),
							"payment_capture_attempted_at": time.Now().Format(time.RFC3339),
						})
					} else {
						// Payment captured successfully
						paymentCaptured = true
						submission.PaymentIntentID = capturedPI.ID

						rdb.HSet(ctx, submissionKey, map[string]interface{}{
							"payment_intent_id":   capturedPI.ID,
							"payment_captured":    "true",
							"payment_captured_at": time.Now().Format(time.RFC3339),
						})

						fmt.Printf("Successfully captured payment %s for submission %s", capturedPI.ID, submissionID)

						// The ticket will be created when the payment.intent.succeeded webhook fires
						// This ensures proper flow and prevents duplicate tickets
						// The webhook will also send the approval + ticket email
					}
				} else if pi.Status == "succeeded" {
					// Payment already succeeded (might be auto-capture)
					log.Printf("Payment already succeeded for submission %s", submissionID)
					paymentCaptured = true
					submission.PaymentIntentID = pi.ID

					// Check if ticket was already created
					existingTicket, _ := rdb.HGet(ctx, submissionKey, "ticket_id").Result()
					if existingTicket == "" {
						log.Printf("No ticket found for approved submission %s with succeeded payment. Webhook might handle it.", submissionID)
					}
				} else {
					fmt.Printf("Payment intent %s for submission %s has status: %s (capture method: %s)",
						pi.ID, submissionID, pi.Status, pi.CaptureMethod)
				}
			}
		}
	}

	// Log the decision
	fmt.Printf("Submission %s approved. Payment status: captured=%v, processing=%v. Email will be sent upon payment completion.", submissionID, paymentCaptured, paymentProcessing)

	// Get updated submission
	updatedSubmission, _ := getSubmissionByID(submissionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submission":      updatedSubmission,
		"paymentCaptured": paymentCaptured,
		"message":         "Submission approved successfully. Payment is being processed and confirmation email will be sent shortly.",
	})
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

	// Check if this tier requires approval
	requiresApproval := true // Default to requiring approval
	if req.PriceID != "" {
		requiresApproval = checkIfTierRequiresApproval(req.PriceID)
	}

	// Determine if we need manual capture based on submission status and tier requirements
	needsManualCapture := requiresApproval && submission.Status != "approved"

	// Create checkout session params
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
		Metadata: map[string]string{
			"submission_id":     req.SubmissionID,
			"event_id":          submission.EventID,
			"event_name":        req.EventName,
			"participant":       "true",
			"requires_approval": strconv.FormatBool(requiresApproval),
			"submission_status": submission.Status,
		},
		CustomerEmail: stripe.String(submission.ParticipantEmail),
	}

	// Only use manual capture if submission needs approval
	if needsManualCapture {
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			CaptureMethod: stripe.String("manual"),
			Metadata: map[string]string{
				"submission_id":     req.SubmissionID,
				"event_id":          submission.EventID,
				"event_name":        req.EventName,
				"participant":       "true",
				"requires_approval": "true",
			},
		}

		fmt.Printf("Creating participant checkout with manual capture for submission %s (requires approval)", req.SubmissionID)
	} else {
		// Auto-approved or already approved - use automatic capture
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"submission_id":     req.SubmissionID,
				"event_id":          submission.EventID,
				"event_name":        req.EventName,
				"participant":       "true",
				"requires_approval": "false",
				"auto_approved":     strconv.FormatBool(submission.Status == "approved"),
			},
		}

		fmt.Printf("Creating participant checkout with automatic capture for submission %s (approved/auto-approved)", req.SubmissionID)
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

	updates := map[string]interface{}{
		"checkout_session_id": s.ID,
		"price_id":            req.PriceID,
		"requires_approval":   strconv.FormatBool(requiresApproval),
		"checkout_created_at": time.Now().Format(time.RFC3339),
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		log.Printf("Error updating submission with checkout session: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":        s.ID,
		"sessionUrl":       s.URL,
		"requiresApproval": requiresApproval,
	})
}

// GetSubmissionPaymentStatus checks the payment status for a submission
func GetSubmissionPaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	type PaymentStatus struct {
		HasPayment      bool   `json:"hasPayment"`
		PaymentStatus   string `json:"paymentStatus"`
		PaymentAmount   int64  `json:"paymentAmount"`
		PaymentCurrency string `json:"paymentCurrency"`
		CheckoutURL     string `json:"checkoutUrl,omitempty"`
		ErrorMessage    string `json:"errorMessage,omitempty"`
		EmailSent       bool   `json:"emailSent"`
		EmailSentAt     string `json:"emailSentAt,omitempty"`
	}

	status := PaymentStatus{
		HasPayment: false,
		EmailSent:  false,
	}

	// Check email status from Redis
	rdb := services.GetRedisClient()
	ctx := context.Background()
	emailStatus, _ := rdb.HGet(ctx, fmt.Sprintf("submission:%s", submissionID), "approval_email_sent").Result()
	emailSentAt, _ := rdb.HGet(ctx, fmt.Sprintf("submission:%s", submissionID), "approval_email_sent_at").Result()

	status.EmailSent = emailStatus == "true"
	status.EmailSentAt = emailSentAt

	// Check payment status from Stripe
	if submission.CheckoutSessionID != "" {
		// Check checkout session status
		params := &stripe.CheckoutSessionParams{}
		sess, err := session.Get(submission.CheckoutSessionID, params)
		if err != nil {
			status.ErrorMessage = fmt.Sprintf("Failed to retrieve checkout session: %v", err)
		} else {
			status.HasPayment = true
			status.PaymentStatus = string(sess.PaymentStatus)
			status.PaymentAmount = sess.AmountTotal
			status.PaymentCurrency = string(sess.Currency)

			// If payment is incomplete, provide the URL
			if sess.PaymentStatus != "paid" && sess.URL != "" {
				status.CheckoutURL = sess.URL
			}
		}
	} else if submission.PaymentIntentID != "" {
		// Check payment intent status
		params := &stripe.PaymentIntentParams{}
		pi, err := paymentintent.Get(submission.PaymentIntentID, params)
		if err != nil {
			status.ErrorMessage = fmt.Sprintf("Failed to retrieve payment intent: %v", err)
		} else {
			status.HasPayment = true
			status.PaymentStatus = string(pi.Status)
			status.PaymentAmount = pi.Amount
			status.PaymentCurrency = string(pi.Currency)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// CreateSubmissionPayment manually creates a payment for an approved submission
func CreateSubmissionPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	var req struct {
		PriceID   string `json:"priceId"`
		EventName string `json:"eventName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Ensure submission is approved
	if submission.Status != "approved" {
		http.Error(w, "Submission must be approved first", http.StatusBadRequest)
		return
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", os.Getenv("BASE_URL"))),
		CancelURL:  stripe.String(fmt.Sprintf("%s/events/%s", os.Getenv("BASE_URL"), submission.EventSlug)),
		Metadata: map[string]string{
			"submission_id": submission.ID,
			"event_id":      submission.EventID,
			"event_name":    req.EventName,
			"type":          "participant_registration",
		},
		CustomerEmail: stripe.String(submission.ParticipantEmail),
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, "Failed to create payment session", http.StatusInternalServerError)
		return
	}

	// Update submission with checkout session ID
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	if err := rdb.HSet(ctx, submissionKey, "checkout_session_id", s.ID).Err(); err != nil {
		log.Printf("Error updating submission with checkout session: %v", err)
	}

	// Send or resend approval email with payment link
	sendApprovalEmail(*submission, s.URL)

	// Update email sent status
	rdb.HSet(ctx, submissionKey, map[string]interface{}{
		"approval_email_sent":    "true",
		"approval_email_sent_at": time.Now().Format(time.RFC3339),
		"manual_payment_created": "true",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"sessionUrl": s.URL,
		"sessionId":  s.ID,
	})
}

// ResendApprovalEmail resends the approval email for a submission
func ResendApprovalEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	submissionID := vars["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Ensure submission is approved
	if submission.Status != "approved" {
		http.Error(w, "Submission must be approved to resend email", http.StatusBadRequest)
		return
	}

	// Generate payment link
	paymentLink := ""

	// If there's a checkout session, get the URL
	if submission.CheckoutSessionID != "" {
		params := &stripe.CheckoutSessionParams{}
		sess, err := session.Get(submission.CheckoutSessionID, params)
		if err == nil && sess.URL != "" {
			paymentLink = sess.URL
		}
	}

	// If no checkout session URL, create a new payment link
	if paymentLink == "" {
		paymentLink, _ = createSubmissionPaymentLink(*submission)
	}

	// Send approval email
	sendApprovalEmail(*submission, paymentLink)

	// Update email sent status
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	rdb.HSet(ctx, submissionKey, map[string]interface{}{
		"approval_email_sent":    "true",
		"approval_email_sent_at": time.Now().Format(time.RFC3339),
		"approval_email_resent":  "true",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Approval email resent successfully",
	})
}

// GetAllSubmissionsWithIssues retrieves submissions that have issues
func GetAllSubmissionsWithIssues(w http.ResponseWriter, r *http.Request) {
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get all submission keys
	submissionKeys, err := rdb.Keys(ctx, "submission:*").Result()
	if err != nil {
		http.Error(w, "Failed to retrieve submissions", http.StatusInternalServerError)
		return
	}

	// For debugging, let's log how many submissions we found
	fmt.Printf("Found %d submission keys in Redis", len(submissionKeys))

	var issueSubmissions []map[string]interface{}

	for _, key := range submissionKeys {
		data, err := rdb.HGetAll(ctx, key).Result()
		if err != nil || len(data) == 0 {
			log.Printf("Error or empty data for key %s: %v", key, err)
			continue
		}

		submissionID := extractSubmissionID(key)
		status := data["status"]

		// Log the submission we're processing for debugging
		fmt.Printf("Processing submission %s with status %s", submissionID, status)

		// Check for issues
		hasIssue := false
		issues := []string{}

		// Check ALL submissions that have any payment-related fields
		// This will catch any submissions that were started in the payment process
		hasPaymentData := data["checkout_session_id"] != "" ||
			data["payment_intent_id"] != "" ||
			data["ticket_id"] != ""

		// Issues for approved submissions
		if status == "approved" {
			// Issue 1: Approved but no payment initialized
			if data["checkout_session_id"] == "" && data["payment_intent_id"] == "" {
				hasIssue = true
				issues = append(issues, "no_payment")
				log.Printf("Submission %s has no payment", submissionID)
			}

			// Issue 2: Approved but email not sent
			if data["approval_email_sent"] != "true" {
				hasIssue = true
				issues = append(issues, "email_not_sent")
				log.Printf("Submission %s email not sent", submissionID)
			}

			// Issue 3: Has checkout session but need to check with Stripe
			if data["checkout_session_id"] != "" {
				sessionValid := false
				params := &stripe.CheckoutSessionParams{}
				sess, err := session.Get(data["checkout_session_id"], params)
				if err != nil {
					// Can't check session status - mark as potential issue
					log.Printf("Error checking session %s: %v", data["checkout_session_id"], err)
					hasIssue = true
					issues = append(issues, "payment_check_failed")
				} else {
					if sess.PaymentStatus != "paid" {
						if sess.ExpiresAt < time.Now().Unix() {
							hasIssue = true
							issues = append(issues, "payment_expired")
							log.Printf("Submission %s payment expired", submissionID)
						} else {
							hasIssue = true
							issues = append(issues, "payment_incomplete")
							log.Printf("Submission %s payment incomplete", submissionID)
						}
					} else {
						// Payment is paid
						sessionValid = true
					}
				}

				// Issue 3b: Has checkout_session_id but not payment_intent_id (incomplete process)
				if data["payment_intent_id"] == "" && !sessionValid {
					hasIssue = true
					issues = append(issues, "missing_payment_intent")
					log.Printf("Submission %s has checkout session but no payment intent", submissionID)
				}
			}

			// Issue 4: Has payment_intent_id but we should verify it with Stripe
			if data["payment_intent_id"] != "" && data["ticket_id"] == "" {
				params := &stripe.PaymentIntentParams{}
				pi, err := paymentintent.Get(data["payment_intent_id"], params)
				if err != nil {
					log.Printf("Error checking payment intent %s: %v", data["payment_intent_id"], err)
					hasIssue = true
					issues = append(issues, "payment_intent_check_failed")
				} else if pi.Status != "succeeded" {
					hasIssue = true
					issues = append(issues, "payment_not_succeeded")
					log.Printf("Submission %s payment intent not succeeded: %s", submissionID, pi.Status)
				}
			}

			// Issue 5: Approved but no ticket created despite payment
			if data["payment_intent_id"] != "" && data["ticket_id"] == "" {
				hasIssue = true
				issues = append(issues, "no_ticket_created")
				log.Printf("Submission %s has payment but no ticket", submissionID)
			}

			// Issue 6: Missing checkout data (neither checkout session nor ticket)
			if data["checkout_session_id"] == "" && data["ticket_id"] == "" {
				hasIssue = true
				issues = append(issues, "missing_checkout_data")
				log.Printf("Submission %s missing checkout data", submissionID)
			}
		}

		// Check for submissions that were paid but not approved
		if status != "approved" && hasPaymentData {
			hasIssue = true
			issues = append(issues, "payment_without_approval")
			log.Printf("Submission %s has payment data but not approved", submissionID)
		}

		// Check for pending submissions that have checkout/payment info but no approval
		if status == "pending" && hasPaymentData {
			hasIssue = true
			issues = append(issues, "pending_with_payment")
			log.Printf("Submission %s is pending with payment data", submissionID)
		}

		// Check if this submission has been stuck in pending for too long
		if status == "pending" {
			submittedAt, err := time.Parse(time.RFC3339, data["submitted_at"])
			if err == nil {
				twoWeeksAgo := time.Now().AddDate(0, 0, -14) // 2 weeks ago
				if submittedAt.Before(twoWeeksAgo) {
					hasIssue = true
					issues = append(issues, "pending_too_long")
					log.Printf("Submission %s pending too long", submissionID)
				}
			}
		}

		// Include ALL submissions with checkout data that have no ticket
		if hasPaymentData && data["ticket_id"] == "" {
			hasIssue = true
			if !contains(issues, "no_ticket_created") {
				issues = append(issues, "no_ticket_created")
				log.Printf("Submission %s has payment data but no ticket", submissionID)
			}
		}

		// Include any submission with payment_intent_id that doesn't have either a checkout_session_id or ticket_id
		// This could indicate a partially completed process
		if data["payment_intent_id"] != "" && (data["checkout_session_id"] == "" || data["ticket_id"] == "") {
			hasIssue = true
			if !contains(issues, "incomplete_payment_process") {
				issues = append(issues, "incomplete_payment_process")
				log.Printf("Submission %s has incomplete payment process", submissionID)
			}
		}

		// Allow forcing inclusion via URL parameter for debugging
		forceInclude := r.URL.Query().Get("debug") == "true" ||
			r.URL.Query().Get("all") == "true" ||
			r.URL.Query().Get("include_id") == submissionID

		if hasIssue || forceInclude {
			// Parse images
			var images []string
			if data["images"] != "" {
				json.Unmarshal([]byte(data["images"]), &images)
			}

			// Convert to submission format
			submission := map[string]interface{}{
				"id":                   submissionID,
				"eventId":              data["event_id"],
				"eventSlug":            data["event_slug"],
				"participantName":      data["participant_name"],
				"participantEmail":     data["participant_email"],
				"participantPhone":     data["participant_phone"],
				"vehicleYear":          data["vehicle_year"],
				"vehicleMake":          data["vehicle_make"],
				"vehicleModel":         data["vehicle_model"],
				"vehicleDescription":   data["vehicle_description"],
				"vehicleModifications": data["vehicle_modifications"],
				"status":               status,
				"submittedAt":          data["submitted_at"],
				"reviewedAt":           data["reviewed_at"],
				"reviewedBy":           data["reviewed_by"],
				"reviewNotes":          data["review_notes"],
				"images":               images,
				"issues":               issues,
				"checkoutSessionId":    data["checkout_session_id"],
				"paymentIntentId":      data["payment_intent_id"],
				"ticketId":             data["ticket_id"],
				"emailSent":            data["approval_email_sent"] == "true",
				"emailSentAt":          data["approval_email_sent_at"],
				"hasIssue":             hasIssue,
			}

			issueSubmissions = append(issueSubmissions, submission)
		}
	}

	log.Printf("Found %d submissions with issues", len(issueSubmissions))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": issueSubmissions,
		"total":       len(issueSubmissions),
	})
}

// Helper function to extract submission ID from Redis key
func extractSubmissionID(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
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
	baseUrl := os.Getenv("BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://eurohaus.shop"
	}

	// Check if we have a price ID to create a proper checkout session
	if submission.TicketID != "" {
		// Initialize Stripe
		stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

		// Create checkout session for auto-approved submission
		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(submission.TicketID),
					Quantity: stripe.Int64(int64(submission.TicketQuantity)),
				},
			},
			SuccessURL: stripe.String(fmt.Sprintf("%s/events/%s/success?submission=%s", baseUrl, submission.EventSlug, submission.ID)),
			CancelURL:  stripe.String(fmt.Sprintf("%s/events/%s?submission=%s", baseUrl, submission.EventSlug, submission.ID)),
			Metadata: map[string]string{
				"submission_id": submission.ID,
				"event_id":      submission.EventID,
				"event_slug":    submission.EventSlug,
				"auto_approved": "true",
			},
		}

		// Add customer email if available
		if submission.ParticipantEmail != "" {
			params.CustomerEmail = stripe.String(submission.ParticipantEmail)
		}

		session, err := session.New(params)
		if err != nil {
			log.Printf("Failed to create checkout session for auto-approved submission: %v", err)
			// Fallback to basic link
			return fmt.Sprintf("%s/events/%s?submission=%s", baseUrl, submission.EventSlug, submission.ID), nil
		}

		// Store the checkout session ID with the submission
		rdb := services.GetRedisClient()
		ctx := context.Background()
		submissionKey := fmt.Sprintf("submission:%s", submission.ID)
		rdb.HSet(ctx, submissionKey, "checkout_session_id", session.ID)

		return session.URL, nil
	}

	// Fallback: return a link to the event page with submission ID
	return fmt.Sprintf("%s/events/%s?submission=%s", baseUrl, submission.EventSlug, submission.ID), nil
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
			<p>We will review your submission and notify you within a week. Once approved, you'll receive a link to complete your registration and payment.</p>
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

// checkIfTierRequiresApproval checks if a price tier requires manual approval
func checkIfTierRequiresApproval(priceID string) bool {
	if priceID == "" {
		// If no price ID provided, default to requiring approval
		return true
	}

	// Initialize Stripe client
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Fetch the price from Stripe
	priceObj, err := price.Get(priceID, nil)
	if err != nil {
		log.Printf("Error fetching price %s: %v", priceID, err)
		// Default to requiring approval if we can't fetch the price
		return true
	}

	// Check the metadata for requires_approval flag
	if requiresApproval, exists := priceObj.Metadata["requires_approval"]; exists {
		// If explicitly set to "false", don't require approval
		return requiresApproval != "false"
	}

	// Default to requiring approval if not specified
	return true
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
