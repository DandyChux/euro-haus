package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"gorm.io/gorm"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/promotioncode"
	"github.com/stripe/stripe-go/v82/refund"
)

func submissionNullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func submissionFormatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

// CreateSubmission handles vehicle submission with image uploads
func CreateSubmission(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	submission := models.VehicleSubmissionDTO{
		ID:                   generateSubmissionID(),
		EventID:              r.FormValue("event_id"),
		EventSlug:            r.FormValue("event_slug"),
		ParticipantName:      r.FormValue("participant_name"),
		ParticipantEmail:     r.FormValue("participant_email"),
		ParticipantPhone:     r.FormValue("participant_phone"),
		VehicleYear:          r.FormValue("vehicle_year"),
		VehicleMake:          r.FormValue("vehicle_make"),
		VehicleModel:         r.FormValue("vehicle_model"),
		VehicleDescription:   r.FormValue("vehicle_description"),
		VehicleModifications: r.FormValue("vehicle_modifications"),
		Status:               "pending",
		SubmittedAt:          time.Now().Format(time.RFC3339),
		Images:               []string{},
	}

	priceID := r.FormValue("price_id")
	submission.PriceID = priceID

	if submission.EventID == "" || submission.ParticipantName == "" || submission.ParticipantEmail == "" ||
		submission.VehicleYear == "" || submission.VehicleMake == "" || submission.VehicleModel == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "At least one image is required", http.StatusBadRequest)
		return
	}

	for i, fileHeader := range files {
		if i >= 5 {
			break
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Error opening file: %v", err)
			continue
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			log.Printf("Error reading file: %v", err)
			continue
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg"
		}

		folder := fmt.Sprintf("events/%s/%s/", submission.EventSlug, submission.ID)

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

	requiresApproval := priceRequiresApproval(priceID)
	submission.RequiresApproval = requiresApproval

	imagesJSON, err := json.Marshal(submission.Images)
	if err != nil {
		http.Error(w, "Failed to encode images", http.StatusInternalServerError)
		return
	}

	db := services.GetDB()

	event, err := findEventByID(r.Context(), r.FormValue("eventId"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	submission.EventID = event.ID
	submission.EventSlug = event.Slug


	err = db.WithContext(r.Context()).Exec(`
		INSERT INTO vehicle_submissions (
			id,
			event_id,
			event_slug,
			participant_name,
			participant_email,
			participant_phone,
			vehicle_year,
			vehicle_make,
			vehicle_model,
			vehicle_description,
			vehicle_modifications,
			images,
			status,
			submitted_at,
			price_id,
			requires_approval
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		submission.ID,
		submission.EventID,
		submission.EventSlug,
		submission.ParticipantName,
		submission.ParticipantEmail,
		submissionNullableString(submission.ParticipantPhone),
		submission.VehicleYear,
		submission.VehicleMake,
		submission.VehicleModel,
		submissionNullableString(submission.VehicleDescription),
		submissionNullableString(submission.VehicleModifications),
		imagesJSON,
		submission.Status,
		time.Now().UTC(),
		submissionNullableString(priceID),
		requiresApproval,
	).Error
	if err != nil {
		http.Error(w, "Failed to store submission", http.StatusInternalServerError)
		return
	}

	if !requiresApproval {
		if submission.CheckoutSessionID != "" {
			paymentCaptured, paymentProcessing, captureErr := captureSubmissionPayment(submission.ID)
			if captureErr != nil {
				log.Printf("Could not capture payment for auto-approved submission %s: %v", submission.ID, captureErr)
			}
			if paymentCaptured || paymentProcessing {
				fmt.Printf("Auto-approved submission %s with payment captured/processing. Webhook will handle the rest.\n", submission.ID)
			}
		} else if priceID != "" {
			paymentLink, err := createSubmissionPaymentLink(submission)
			if err != nil {
				log.Printf("Error creating payment link for auto-approved submission: %v", err)
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "https://eurohaus.shop"
				}
				paymentLink = fmt.Sprintf("%s/events/%s?submission=%s", baseURL, submission.EventID, submission.ID)
			}

			go sendApprovalEmail(submission, paymentLink)

			err = db.WithContext(r.Context()).Exec(`
				UPDATE vehicle_submissions
				SET approval_email_sent_at = NOW()
				WHERE id = ?
			`, submission.ID).Error
			if err != nil {
				log.Printf("Failed to mark approval email sent for submission %s: %v", submission.ID, err)
			}

			submission.ApprovalEmailSentAt = time.Now().Format(time.RFC3339)

			fmt.Printf("Auto-approved submission %s - sent approval email with payment link\n", submission.ID)
		}
	} else {
		go sendSubmissionConfirmationEmail(submission)
		fmt.Printf("Created pending submission %s and sent confirmation email\n", submission.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

// GetEventSubmissions retrieves all submissions for an event
func GetEventSubmissions(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	rows, err := db.WithContext(r.Context()).Raw(`
		SELECT id
		FROM vehicle_submissions
		WHERE event_id = ?
		ORDER BY submitted_at DESC
	`, eventID).Rows()
	if err != nil {
		http.Error(w, "Failed to retrieve submissions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	submissions := []models.VehicleSubmissionDTO{}
	for rows.Next() {
		var submissionID string
		if err := rows.Scan(&submissionID); err != nil {
			continue
		}
		submission, err := getSubmissionByID(submissionID)
		if err != nil {
			log.Printf("Error retrieving submission %s: %v", submissionID, err)
			continue
		}
		submissions = append(submissions, *submission)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": submissions,
	})
}

// GetSubmission retrieves a single submission by ID
func GetSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

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
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status == "approved" {
		log.Printf("Submission %s is already approved", submissionID)
		http.Error(w, "Submission is already approved", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	err = db.WithContext(r.Context()).Exec(`
			UPDATE vehicle_submissions
			SET status = 'approved',
				reviewed_at = NOW(),
				reviewed_by = 'admin',
				review_notes = ?
			WHERE id = ?
		`, submissionNullableString(req.Notes), submissionID).Error
	if err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Submission %s approved by admin", submissionID)

	paymentCaptured := false
	paymentProcessing := false

	if submission.CheckoutSessionID != "" {
		var captureErr error
		paymentCaptured, paymentProcessing, captureErr = captureSubmissionPayment(submissionID)
		if captureErr != nil {
			log.Printf(
				"Payment capture failed for submission %s: %v",
				submissionID,
				captureErr,
			)
		}
	}

	fmt.Printf("Submission %s approved. Payment status: captured=%v, processing=%v. Email will be sent upon payment completion.", submissionID, paymentCaptured, paymentProcessing)

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
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Review notes are required for denial", http.StatusBadRequest)
		return
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	db := services.GetDB()
	err = db.WithContext(r.Context()).Exec(`
			UPDATE vehicle_submissions
			SET status = 'denied',
				reviewed_at = NOW(),
				reviewed_by = 'admin',
				review_notes = ?
			WHERE id = ?
		`, submissionNullableString(req.Notes), submissionID).Error
	if err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	go sendDenialEmail(*submission, req.Notes)

	updatedSubmission, _ := getSubmissionByID(submissionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedSubmission)
}

// GetPendingSubmissionsCount returns the count of pending submissions
func GetPendingSubmissionsCount(w http.ResponseWriter, r *http.Request) {
	db := services.GetDB()

	var count int
	err := db.WithContext(r.Context()).Raw(`
		SELECT COUNT(*)
		FROM vehicle_submissions
		WHERE status = 'pending'
	`).Row().Scan(&count)
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

	submission, err := getSubmissionByID(req.SubmissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(w, "Submission is not approved", http.StatusBadRequest)
		return
	}

	baseURL := os.Getenv("BASE_URL")
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
		SuccessURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/success?session_id={CHECKOUT_SESSION_ID}&event_id=%s",
				baseURL,
				url.QueryEscape(submission.EventID),
			),
		),
		CancelURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/cancel?event_id=%s",
				baseURL,
				url.QueryEscape(submission.EventID),
			),
		),
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

	db := services.GetDB()
	err = db.WithContext(r.Context()).Exec(`
			UPDATE vehicle_submissions
			SET checkout_session_id = ?,
				price_id = ?
			WHERE id = ?
		`, s.ID, submissionNullableString(req.PriceID), req.SubmissionID).Error
	if err != nil {
		log.Printf("Failed to store checkout session for submission %s: %v", req.SubmissionID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionUrl": s.URL,
	})
}

// CreateParticipantCheckout creates a checkout session with manual capture for submissions
func CreateParticipantCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubmissionID  string `json:"submission_id"`
		PriceID       string `json:"price_id"`
		EventName     string `json:"event_name"`
		Quantity      int64  `json:"quantity"`
		PromotionCode string `json:"promotion_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	submission, err := getSubmissionByID(req.SubmissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	requiresApproval := true
	if req.PriceID != "" {
		requiresApproval = priceRequiresApproval(req.PriceID)
	}

	needsManualCapture := requiresApproval && submission.Status != "approved"

	baseURL := os.Getenv("BASE_URL")
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
		SuccessURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/pending?submission_id=%s&event_id=%s",
				baseURL,
				url.QueryEscape(req.SubmissionID),
				url.QueryEscape(submission.EventID),
			),
		),
		CancelURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/cancel?event_id=%s",
				baseURL,
				url.QueryEscape(submission.EventID),
			),
		),
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

	if req.PromotionCode != "" {
		pcParams := &stripe.PromotionCodeListParams{
			Code: stripe.String(req.PromotionCode),
		}
		pcParams.Filters.AddFilter("limit", "", "1")

		iter := promotioncode.List(pcParams)
		if iter.Next() {
			pc := iter.PromotionCode()
			if pc.Active {
				params.Discounts = []*stripe.CheckoutSessionDiscountParams{
					{
						PromotionCode: stripe.String(pc.ID),
					},
				}
				fmt.Printf("Applied promotion code %s to checkout for submission %s\n", req.PromotionCode, req.SubmissionID)
			} else {
				fmt.Printf("Promotion code %s is not active for submission %s\n", req.PromotionCode, req.SubmissionID)
			}
		} else {
			fmt.Printf("Promotion code %s not found for submission %s\n", req.PromotionCode, req.SubmissionID)
		}
	}

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

	db := services.GetDB()
	err = db.WithContext(r.Context()).Exec(`
			UPDATE vehicle_submissions
			SET checkout_session_id = ?,
				price_id = ?,
				requires_approval = ?,
				checkout_created_at = NOW(),
				promotion_code = ?
			WHERE id = ?
		`, s.ID, submissionNullableString(req.PriceID), requiresApproval, submissionNullableString(req.PromotionCode), req.SubmissionID).Error
	if err != nil {
		log.Printf("Error updating submission with checkout session: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id":        s.ID,
		"session_url":       s.URL,
		"requires_approval": requiresApproval,
	})
}

// GetSubmissionPaymentStatus checks the payment status for a submission
func GetSubmissionPaymentStatus(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submission_id"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	type PaymentStatus struct {
		HasPayment      bool   `json:"has_payment"`
		PaymentStatus   string `json:"payment_status"`
		PaymentAmount   int64  `json:"payment_amount"`
		PaymentCurrency string `json:"payment_currency"`
		CheckoutURL     string `json:"checkout_url,omitempty"`
		ErrorMessage    string `json:"error_message,omitempty"`
	}

	status := PaymentStatus{
		HasPayment:  false,
	}

	if submission.CheckoutSessionID != "" {
		params := &stripe.CheckoutSessionParams{}
		sess, err := session.Get(submission.CheckoutSessionID, params)
		if err != nil {
			status.ErrorMessage = fmt.Sprintf("Failed to retrieve checkout session: %v", err)
		} else {
			status.HasPayment = true
			status.PaymentStatus = string(sess.PaymentStatus)
			status.PaymentAmount = sess.AmountTotal
			status.PaymentCurrency = string(sess.Currency)

			if sess.PaymentStatus != "paid" && sess.URL != "" {
				status.CheckoutURL = sess.URL
			}
		}
	} else if submission.PaymentIntentID != "" {
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
	submissionID := mux.Vars(r)["submissionId"]

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

	if submission.Status != "approved" {
		http.Error(w, "Submission must be approved first", http.StatusBadRequest)
		return
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", os.Getenv("BASE_URL"))),
		CancelURL:  stripe.String(fmt.Sprintf("%s/event/%s", os.Getenv("BASE_URL"), submission.EventID)),
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

	db := services.GetDB()
	err = db.WithContext(r.Context()).Exec(`
			UPDATE vehicle_submissions
			SET checkout_session_id = ?,
				price_id = ?,
				approval_email_sent_at = NOW(),
			WHERE id = ?
	`, s.ID, submissionNullableString(req.PriceID), submissionID).Error
	if err != nil {
		log.Printf("Error updating submission with checkout session: %v", err)
	}

	sendApprovalEmail(*submission, s.URL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"sessionUrl": s.URL,
		"sessionId":  s.ID,
	})
}

// ResendApprovalEmail resends the approval email for a submission
func ResendApprovalEmail(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(w, "Submission must be approved to resend email", http.StatusBadRequest)
		return
	}

	paymentLink := ""

	if submission.CheckoutSessionID != "" {
		params := &stripe.CheckoutSessionParams{}
		sess, err := session.Get(submission.CheckoutSessionID, params)
		if err == nil {
			if sess.PaymentStatus != "paid" && sess.URL != "" {
				paymentLink = sess.URL
			} else if sess.PaymentStatus == "paid" {
				paymentLink = ""
			}
		}
	}

	if paymentLink == "" {
		paymentLink, _ = createSubmissionPaymentLink(*submission)
	}

	sendApprovalEmail(*submission, paymentLink)

	db := services.GetDB()
	err = db.Exec(`
		UPDATE vehicle_submissions
			SET approval_email_sent_at = NOW(),
				email_resent_count = COALESCE(email_resent_count, 0) + 1
			WHERE id = ?
		`, submissionID).Error
	if err != nil {
		log.Printf("Failed to update approval email resend state for submission %s: %v", submissionID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Approval email resent successfully",
		"paymentLink": paymentLink != "",
	})
}

// GetAllSubmissionsWithIssues retrieves submissions that have issues
func GetAllSubmissionsWithIssues(w http.ResponseWriter, r *http.Request) {
	db := services.GetDB()

	rows, err := db.WithContext(r.Context()).Raw(`
			SELECT id
			FROM vehicle_submissions
			ORDER BY submitted_at DESC
		`).Rows()
	if err != nil {
		http.Error(w, "Failed to retrieve submissions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var submissionIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			submissionIDs = append(submissionIDs, id)
		}
	}

	fmt.Printf("Found %d submissions in Postgres", len(submissionIDs))

	issueSubmissions := make([]models.VehicleSubmissionDTO, 0)

	for _, submissionID := range submissionIDs {
		submission, err := getSubmissionByID(submissionID)
		if err != nil {
			log.Printf("Error loading submission %s: %v", submissionID, err)
			continue
		}

		status := submission.Status
		fmt.Printf("Processing submission %s with status %s", submissionID, status)

		hasIssue := false
		issues := []string{}

		hasPaymentData := submission.CheckoutSessionID != "" ||
			submission.PaymentIntentID != ""

		if status == "approved" {
			if submission.CheckoutSessionID == "" && submission.PaymentIntentID == "" {
				hasIssue = true
				issues = append(issues, "no_payment")
				fmt.Printf("Submission %s has no payment", submissionID)
			}

			if submission.CheckoutSessionID != "" {
				sessionValid := false
				params := &stripe.CheckoutSessionParams{}
				sess, err := session.Get(submission.CheckoutSessionID, params)
				if err != nil {
					log.Printf("Error checking session %s: %v", submission.CheckoutSessionID, err)
					hasIssue = true
					issues = append(issues, "payment_check_failed")
				} else {
					if sess.PaymentStatus != "paid" {
						if sess.ExpiresAt < time.Now().Unix() {
							hasIssue = true
							issues = append(issues, "payment_expired")
							fmt.Printf("Submission %s payment expired", submissionID)
						} else {
							hasIssue = true
							issues = append(issues, "payment_incomplete")
							fmt.Printf("Submission %s payment incomplete", submissionID)
						}
					} else {
						sessionValid = true
					}
				}

				if submission.PaymentIntentID == "" && !sessionValid {
					hasIssue = true
					issues = append(issues, "missing_payment_intent")
					fmt.Printf("Submission %s has checkout session but no payment intent", submissionID)
				}
			}

			if submission.PaymentIntentID != "" {
				params := &stripe.PaymentIntentParams{}
				pi, err := paymentintent.Get(submission.PaymentIntentID, params)
				if err != nil {
					log.Printf("Error checking payment intent %s: %v", submission.PaymentIntentID, err)
					hasIssue = true
					issues = append(issues, "payment_intent_check_failed")
				} else if pi.Status != "succeeded" {
					hasIssue = true
					issues = append(issues, "payment_not_succeeded")
					fmt.Printf("Submission %s payment intent not succeeded: %s", submissionID, pi.Status)
				}
			}

			if submission.PaymentIntentID != "" {
				hasIssue = true
				issues = append(issues, "no_ticket_created")
				fmt.Printf("Submission %s has payment but no ticket", submissionID)
			}

			if submission.CheckoutSessionID == "" {
				hasIssue = true
				issues = append(issues, "missing_checkout_data")
				fmt.Printf("Submission %s missing checkout data", submissionID)
			}
		}

		if status != "approved" && hasPaymentData {
			hasIssue = true
			issues = append(issues, "payment_without_approval")
			fmt.Printf("Submission %s has payment data but not approved", submissionID)
		}

		if status == "pending" && hasPaymentData {
			hasIssue = true
			issues = append(issues, "pending_with_payment")
			fmt.Printf("Submission %s is pending with payment data", submissionID)
		}

		if status == "pending" {
			submittedAt, err := time.Parse(time.RFC3339, submission.SubmittedAt)
			if err == nil {
				twoWeeksAgo := time.Now().AddDate(0, 0, -14)
				if submittedAt.Before(twoWeeksAgo) {
					hasIssue = true
					issues = append(issues, "pending_too_long")
					fmt.Printf("Submission %s pending too long", submissionID)
				}
			}
		}

		if hasPaymentData {
			hasIssue = true
			if !contains(issues, "no_ticket_created") {
				issues = append(issues, "no_ticket_created")
				fmt.Printf("Submission %s has payment data but no ticket", submissionID)
			}
		}

		if submission.PaymentIntentID != "" && submission.CheckoutSessionID == "" {
			hasIssue = true
			if !contains(issues, "incomplete_payment_process") {
				issues = append(issues, "incomplete_payment_process")
				fmt.Printf("Submission %s has incomplete payment process", submissionID)
			}
		}

		forceInclude := r.URL.Query().Get("debug") == "true" ||
			r.URL.Query().Get("all") == "true" ||
			r.URL.Query().Get("include_id") == submissionID

			if hasIssue || forceInclude {
				issueSubmission := *submission

				issueSubmission.Issues = issues
				issueSubmission.HasIssue = hasIssue

				issueSubmissions = append(
					issueSubmissions,
					issueSubmission,
				)
			}
	}

	fmt.Printf("Found %d submissions with issues", len(issueSubmissions))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": issueSubmissions,
		"total":       len(issueSubmissions),
	})
}

// UpdateSubmissionEmail updates the email address for a submission
func UpdateSubmissionEmail(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		NewEmail    string `json:"newEmail"`
		ResendEmail bool   `json:"resendEmail"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NewEmail == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	oldEmail := submission.ParticipantEmail
	db := services.GetDB()

	err = db.Exec(`
		UPDATE vehicle_submissions
		SET participant_email = ?,
			email_updated_at = NOW(),
			previous_email = ?
		WHERE id = ?
	`, req.NewEmail, oldEmail, submissionID).Error
	if err != nil {
		http.Error(w, "Failed to update submission email", http.StatusInternalServerError)
		return
	}

	log.Printf("Updated email for submission %s from %s to %s", submissionID, oldEmail, req.NewEmail)

	updatedSubmission, _ := getSubmissionByID(submissionID)

	params := &stripe.CheckoutSessionParams{}
	sess, err := session.Get(submission.CheckoutSessionID, params)

	response := map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("Email updated from %s to %s", oldEmail, req.NewEmail),
		"submission": updatedSubmission,
	}

	if req.ResendEmail && submission.Status == "approved" {
		emailsSent := []string{}

		if sess.PaymentStatus != "paid" {
			paymentLink, _ := createSubmissionPaymentLink(*updatedSubmission)
			sendApprovalEmail(*updatedSubmission, paymentLink)
			emailsSent = append(emailsSent, "approval")
			log.Printf("Sent approval email with payment link to %s for submission %s", req.NewEmail, submissionID)
		} else {
			sendApprovalEmail(*updatedSubmission, "")
			emailsSent = append(emailsSent, "approval")
			log.Printf("Sent approval email to %s for submission %s", req.NewEmail, submissionID)
		}

		err = db.Exec(`
			UPDATE vehicle_submissions
				approval_email_sent_at = NOW(),
				email_resent_count = COALESCE(email_resent_count, 0) + 1
			WHERE id = ?
		`, submissionID).Error
		if err != nil {
			log.Printf("Failed to update approval email resend state for submission %s: %v", submissionID, err)
		}

		response["email_resent"] = true
		response["emails_sent"] = emailsSent

		if len(emailsSent) > 0 {
			emailType := emailsSent[0]
			if emailType == "approval" {
				response["message"] = fmt.Sprintf("Email updated from %s to %s and approval email resent", oldEmail, req.NewEmail)
			} else {
				response["message"] = fmt.Sprintf("Email updated from %s to %s and approval email resent", oldEmail, req.NewEmail)
			}
		}

		updatedSubmission, _ = getSubmissionByID(submissionID)
		response["submission"] = updatedSubmission
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevokeSubmission revokes an approved submission, refunds payment, and cancels ticket
func RevokeSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		Reason       string `json:"reason"`
		RefundAmount string `json:"refund_amount"`
		RefundReason string `json:"refund_reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "Submission revoked by administrator"
		req.RefundAmount = "full"
		req.RefundReason = "requested_by_customer"
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(w, "Can only revoke approved submissions", http.StatusBadRequest)
		return
	}

	db := services.GetDB()

	result := map[string]interface{}{
		"submission_revoked": false,
		"ticket_invalidated": false,
		"payment_refunded":   false,
		"errors":            []string{},
	}

	if submission.PaymentIntentID != "" {
		refundParams := &stripe.RefundParams{
			PaymentIntent: stripe.String(submission.PaymentIntentID),
			Reason:        stripe.String(req.RefundReason),
		}

		refundResult, err := refund.New(refundParams)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to process refund: %v", err)
			log.Printf("Error refunding payment for submission %s: %v", submissionID, err)
			result["errors"] = append(result["errors"].([]string), errMsg)
		} else {
			result["payment_refunded"] = true
			result["refund_id"] = refundResult.ID
			result["refund_amount"] = float64(refundResult.Amount) / 100.0
			result["refund_currency"] = string(refundResult.Currency)
			log.Printf("Refund issued for submission %s: %s", submissionID, refundResult.ID)

			err = db.WithContext(r.Context()).Exec(`
				UPDATE vehicle_submissions
				SET refund_id = ?,
					refund_amount = ?,
					refund_issued_at = NOW()
				WHERE id = ?
			`, refundResult.ID, float64(refundResult.Amount)/100.0, submissionID).Error
			if err != nil {
				log.Printf("Failed to update refund state for submission %s: %v", submissionID, err)
			}
		}
	} else {
		result["payment_refunded"] = true
		result["message"] = "No payment found - no refund needed"
	}

	result["ticket_invalidated"] = true

	err = db.WithContext(r.Context()).Exec(`
		UPDATE vehicle_submissions
		SET status = 'revoked',
		    revoked_at = NOW(),
		    revoked_by = 'admin',
		    revocation_reason = ?
		WHERE id = ?
	`, req.Reason, submissionID).Error
	if err != nil {
		errMsg := fmt.Sprintf("Failed to update submission status: %v", err)
		log.Printf("Error updating submission %s: %v", submissionID, err)
		result["errors"] = append(result["errors"].([]string), errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	result["submissionRevoked"] = true

	sendRevocationEmail(*submission, req.Reason)

	updatedSubmission, _ := getSubmissionByID(submissionID)
	result["submission"] = updatedSubmission

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

func getSubmissionByID(
	submissionID string,
) (*models.VehicleSubmissionDTO, error) {
	db := services.GetDB()

	var (
		imagesJSON          []byte
		submittedAt         time.Time
		reviewedAt          sql.NullTime
		approvalEmailSentAt sql.NullTime
		ticketEmailSentAt   sql.NullTime
		emailUpdatedAt      sql.NullTime
		revokedAt           sql.NullTime
	)

	submission := &models.VehicleSubmissionDTO{}

	err := db.Raw(`
		SELECT
			id,
			event_id,
			COALESCE(event_slug, ''),
			participant_name,
			participant_email,
			COALESCE(participant_phone, ''),
			COALESCE(vehicle_year, ''),
			COALESCE(vehicle_make, ''),
			COALESCE(vehicle_model, ''),
			COALESCE(vehicle_description, ''),
			COALESCE(vehicle_modifications, ''),
			COALESCE(images, '[]'::jsonb),
			status,
			submitted_at,
			reviewed_at,
			COALESCE(reviewed_by, ''),
			COALESCE(review_notes, ''),
			COALESCE(checkout_session_id, ''),
			COALESCE(payment_intent_id, ''),
			COALESCE(price_id, ''),
			approval_email_sent_at,
			COALESCE(requires_approval, true),
			COALESCE(approval_email_resent, false),
			ticket_email_sent_at,
			COALESCE(previous_email, ''),
			email_updated_at,
			COALESCE(email_resent_count, 0),
			revoked_at,
			COALESCE(revoked_by, ''),
			COALESCE(revocation_reason, '')
		FROM vehicle_submissions
		WHERE id = ?
	`, submissionID).Row().Scan(
		&submission.ID,
		&submission.EventID,
		&submission.EventSlug,
		&submission.ParticipantName,
		&submission.ParticipantEmail,
		&submission.ParticipantPhone,
		&submission.VehicleYear,
		&submission.VehicleMake,
		&submission.VehicleModel,
		&submission.VehicleDescription,
		&submission.VehicleModifications,
		&imagesJSON,
		&submission.Status,
		&submittedAt,
		&reviewedAt,
		&submission.ReviewedBy,
		&submission.ReviewNotes,
		&submission.CheckoutSessionID,
		&submission.PaymentIntentID,
		&submission.PriceID,
		&approvalEmailSentAt,
		&submission.RequiresApproval,
		&submission.ApprovalEmailResent,
		&ticketEmailSentAt,
		&submission.PreviousEmail,
		&emailUpdatedAt,
		&submission.EmailResentCount,
		&revokedAt,
		&submission.RevokedBy,
		&submission.RevocationReason,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("submission not found")
		}

		return nil, err
	}

	if err := json.Unmarshal(imagesJSON, &submission.Images); err != nil {
		submission.Images = []string{}
	}

	submission.SubmittedAt = submittedAt.Format(time.RFC3339)
	submission.ReviewedAt = submissionFormatNullTime(reviewedAt)
	submission.ApprovalEmailSentAt =
		submissionFormatNullTime(approvalEmailSentAt)
	submission.TicketEmailSentAt =
		submissionFormatNullTime(ticketEmailSentAt)
	submission.EmailUpdatedAt =
		submissionFormatNullTime(emailUpdatedAt)
	submission.RevokedAt =
		submissionFormatNullTime(revokedAt)

	return submission, nil
}

// captureSubmissionPayment captures payment for an approved submission
// The webhook will handle sending emails and creating tickets when payment succeeds
func captureSubmissionPayment(submissionID string) (bool, bool, error) {
	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		return false, false, fmt.Errorf("submission not found: %v", err)
	}

	if submission.CheckoutSessionID == "" {
		return false, false, fmt.Errorf("no checkout session found")
	}

	sess, err := session.Get(submission.CheckoutSessionID, nil)
	if err != nil {
		return false, false, fmt.Errorf("error retrieving checkout session: %v", err)
	}

	if sess.PaymentIntent == nil {
		return false, false, fmt.Errorf("no payment intent associated with checkout session")
	}

	pi, err := paymentintent.Get(sess.PaymentIntent.ID, nil)
	if err != nil {
		return false, false, fmt.Errorf("error retrieving payment intent: %v", err)
	}

	if pi.Status == "requires_capture" && pi.CaptureMethod == "manual" {
		fmt.Printf("Capturing payment for submission %s (payment intent: %s)\n", submissionID, pi.ID)

		capturedPI, err := paymentintent.Capture(pi.ID, nil)
		if err != nil {
			return false, false, fmt.Errorf("error capturing payment: %v", err)
		}

		err = services.GetDB().WithContext(context.Background()).Exec(`
			UPDATE vehicle_submissions
			SET payment_intent_id = ?,
			    payment_captured = TRUE,
			    payment_captured_at = NOW()
			WHERE id = ?
		`, capturedPI.ID, submissionID).Error
		if err != nil {
			log.Printf("Failed to update payment capture state for submission %s: %v", submissionID, err)
		}

		fmt.Printf("Successfully captured payment %s for submission %s\n", capturedPI.ID, submissionID)
		return true, true, nil
	} else if pi.Status == "succeeded" {
		log.Printf("Payment already succeeded for submission %s\n", submissionID)
		return true, false, nil
	} else if pi.Status == "processing" {
		log.Printf("Payment is processing for submission %s\n", submissionID)
		return false, true, nil
	}

	return false, false, fmt.Errorf("payment intent status: %s (capture method: %s)", pi.Status, pi.CaptureMethod)
}

func createSubmissionPaymentLink(submission models.VehicleSubmissionDTO) (string, error) {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://eurohaus.shop"
	}

	// Important: use price_id, not ticket_id
	if submission.PriceID != "" {
		stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(submission.PriceID),
					Quantity: stripe.Int64(1),
				},
			},
			SuccessURL: stripe.String(
				fmt.Sprintf(
					"%s/checkout/success?session_id={CHECKOUT_SESSION_ID}&event_id=%s",
					baseURL,
					url.QueryEscape(submission.EventID),
				),
			),
			CancelURL: stripe.String(
				fmt.Sprintf(
					"%s/checkout/cancel?event_id=%s",
					baseURL,
					url.QueryEscape(submission.EventID),
				),
			),
			Metadata: map[string]string{
				"submission_id": submission.ID,
				"event_id":      submission.EventID,
				"event_slug":    submission.EventSlug,
				"auto_approved": "true",
				"participant":   "true",
			},
		}

		if submission.ParticipantEmail != "" {
			params.CustomerEmail = stripe.String(submission.ParticipantEmail)
		}

		s, err := session.New(params)
		if err != nil {
			log.Printf("Failed to create checkout session for auto-approved submission: %v", err)
			return fmt.Sprintf("%s/events/%s?submission=%s", baseURL, submission.EventID, submission.ID), nil
		}

		err = services.GetDB().WithContext(context.Background()).Exec(`
			UPDATE vehicle_submissions
			SET checkout_session_id = ?
			WHERE id = ?
		`, s.ID, submission.ID).Error
		if err != nil {
			log.Printf("Failed to persist checkout session for submission %s: %v", submission.ID, err)
		}

		return s.URL, nil
	}

	return fmt.Sprintf("%s/events/%s?submission=%s", baseURL, submission.EventSlug, submission.ID), nil
}

// Email functions

// sendSubmissionTicketEmail sends a ticket email for an approved submission
func sendSubmissionTicketEmail(submissionData map[string]string, ticketToken string, eventName string) {
	// Generate QR code
	qrCodeURL, err := generateQRCode(ticketToken)
	if err != nil {
		log.Printf("Error generating QR code: %v", err)
		return
	}

	vehicleDetails := fmt.Sprintf("%s %s %s",
		submissionData["vehicle_year"],
		submissionData["vehicle_make"],
		submissionData["vehicle_model"])

	customerEmail := submissionData["participant_email"]
	customerName := submissionData["participant_name"]

	emailData := map[string]interface{}{
		"CustomerName":   customerName,
		"EventName":      eventName,
		"TicketCode":     ticketToken,
		"QRCodeURL":      qrCodeURL,
		"VehicleDetails": vehicleDetails,
		"TicketType":     "Event Participant",
		"CheckInURL":     fmt.Sprintf("%s/events/checkin?ticket=%s", os.Getenv("BASE_URL"), ticketToken),
	}

	// Generate ticket HTML
	ticketHTML := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
				<h1 style="color: #007bff;">Your Event Participant Ticket</h1>
				<p>Dear %s,</p>
				<p>Congratulations! Your registration as an event participant is complete. Your vehicle has been approved:</p>
				<p style="font-size: 18px; font-weight: bold;">%s</p>

				<div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px; margin: 20px 0;">
					<h2>Event Details</h2>
					<p><strong>Event:</strong> %s</p>
					<p><strong>Ticket Type:</strong> Event Participant</p>
					<p><strong>Ticket Code:</strong> <span style="font-family: monospace; font-size: 18px;">%s</span></p>
				</div>

				<div style="text-align: center; margin: 30px 0;">
					<img src="%s" alt="QR Code" style="width: 200px; height: 200px;">
					<p style="font-size: 12px; color: #666;">Show this QR code at check-in</p>
				</div>

				<h3>Important Information for Participants:</h3>
				<ul>
					<li>Please arrive at least 30 minutes before the event start time</li>
					<li>Have your vehicle clean and ready for display</li>
					<li>Bring this ticket (printed or on your phone) for check-in</li>
					<li>Follow all event guidelines and instructions from staff</li>
				</ul>

				<p>We're excited to have you showcase your vehicle at our event!</p>
				<p>Best regards,<br>The Euro Haus Events Team</p>
			</div>
		</body>
		</html>
	`, customerName, vehicleDetails, eventName, ticketToken, qrCodeURL)

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      fmt.Sprintf("Event Participant Ticket - %s", eventName),
		TemplateID:   "participant-ticket",
		TemplateData: emailData,
		BodyHTML:     ticketHTML,
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending participant ticket email: %v", err)
	} else {
		log.Printf("Successfully sent ticket email to %s for ticket %s", customerEmail, ticketToken)
	}
}

func sendSubmissionConfirmationEmail(submission models.VehicleSubmissionDTO) {
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

func sendApprovalEmail(submission models.VehicleSubmissionDTO, paymentLink string) {
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

func sendDenialEmail(submission models.VehicleSubmissionDTO, reason string) {
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

func sendRevocationEmail(submission models.VehicleSubmissionDTO, reason string) {
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
		log.Printf("Error sending revocation email: %v", err)
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

func priceRequiresApproval(priceID string) bool {
	if priceID == "" {
		return true
	}

	var priceInfo models.PriceInfo

	err := services.GetDB().
		Where("id = ?", priceID).
		First(&priceInfo).
		Error

	if err != nil {
		log.Printf(
			"Unable to load price %s: %v",
			priceID,
			err,
		)

		return true
	}

	return priceInfo.RequiresApproval
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
