package handlers

import (
	"bytes"
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
	"github.com/google/uuid"
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

func validateRequirementValue(
	requirement models.PriceRequirement,
	value any,
) error {
	switch requirement.Type {
	case "text", "textarea":
		if _, ok := value.(string); !ok {
			return fmt.Errorf(
				"requirement %q must be text",
				requirement.Label,
			)
		}

	case "number":
		switch value.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf(
				"requirement %q must be a number",
				requirement.Label,
			)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf(
				"requirement %q must be boolean",
				requirement.Label,
			)
		}

	case "select", "radio":
		selected, ok := value.(string)
		if !ok {
			return fmt.Errorf(
				"requirement %q must be an option",
				requirement.Label,
			)
		}

		var options []string
		if err := json.Unmarshal(requirement.Options, &options); err != nil {
			return fmt.Errorf(
				"decode options for %s: %w",
				requirement.Key,
				err,
			)
		}

		found := false
		for _, option := range options {
			if option == selected {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf(
				"invalid option for requirement %q",
				requirement.Label,
			)
		}
	}

	return nil
}

func validateSubmittedAnswers(
	tx *gorm.DB,
	priceID string,
	answers map[string]any,
) ([]models.SubmissionRequirementAnswer, error) {
	if priceID == "" {
		if len(answers) > 0 {
			return nil, fmt.Errorf(
				"requirement answers cannot be submitted without a price",
			)
		}

		return nil, nil
	}

	requirements, err := loadPriceRequirements(tx, priceID)
	if err != nil {
		return nil, fmt.Errorf("load requirements: %w", err)
	}

	byID := make(map[string]models.PriceRequirement, len(requirements))
	for _, requirement := range requirements {
		byID[requirement.ID] = requirement
	}

	for requirementID := range answers {
		if _, ok := byID[requirementID]; !ok {
			return nil, fmt.Errorf(
				"unknown requirement %s",
				requirementID,
			)
		}
	}

	result := make([]models.SubmissionRequirementAnswer, 0, len(requirements))

	for _, requirement := range requirements {
		rawValue, exists := answers[requirement.ID]

		if !exists || rawValue == nil || rawValue == "" {
			if requirement.Required {
				return nil, fmt.Errorf(
					"requirement %q is required",
					requirement.Label,
				)
			}

			continue
		}

		if err := validateRequirementValue(requirement, rawValue); err != nil {
			return nil, err
		}

		value, err := json.Marshal(rawValue)
		if err != nil {
			return nil, fmt.Errorf(
				"encode answer for %s: %w",
				requirement.Key,
				err,
			)
		}

		result = append(result, models.SubmissionRequirementAnswer{
			RequirementID: requirement.ID,
			Value:        value,
		})
	}

	return result, nil
}

type SubmittedRequirementAnswer struct {
	RequirementID string `json:"requirement_id"`
	Value any `json:"value"`
}

func CreateSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	eventID := strings.TrimSpace(r.FormValue("event_id"))
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	// Resolve the event before uploading anything. The event slug used for
	// storage must come from the database, not from the client request.
	event, err := findEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Failed to resolve event %s: %v", eventID, err)
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	var submittedAnswers map[string]any

	rawAnswers := strings.TrimSpace(r.FormValue("requirement_answers"))
	if rawAnswers != "" {
		if err := json.Unmarshal(
			[]byte(rawAnswers),
			&submittedAnswers,
		); err != nil {
			http.Error(
				w,
				"Invalid requirement answers",
				http.StatusBadRequest,
			)
			return
		}
	}

	if submittedAnswers == nil {
		submittedAnswers = map[string]any{}
	}

	priceID := strings.TrimSpace(r.FormValue("price_id"))

	uuid, err := uuid.NewV7()
	if err != nil {
		log.Printf("Failed to generate UUID: %v", err)
		http.Error(w, "Unable to generate submission ID", http.StatusInternalServerError)
		return
	}

	submission := models.VehicleSubmissionDTO{
		ID:                   uuid.String(),
		EventID:              event.ID,
		EventSlug:            event.Slug,
		ParticipantName:      strings.TrimSpace(r.FormValue("participant_name")),
		ParticipantEmail:     strings.TrimSpace(r.FormValue("participant_email")),
		ParticipantPhone:     strings.TrimSpace(r.FormValue("participant_phone")),
		VehicleYear:          strings.TrimSpace(r.FormValue("vehicle_year")),
		VehicleMake:          strings.TrimSpace(r.FormValue("vehicle_make")),
		VehicleModel:         strings.TrimSpace(r.FormValue("vehicle_model")),
		VehicleDescription:   strings.TrimSpace(r.FormValue("vehicle_description")),
		VehicleModifications: strings.TrimSpace(r.FormValue("vehicle_modifications")),
		Status:               "pending",
		SubmittedAt:          time.Now().UTC().Format(time.RFC3339),
		Images:               []string{},
		PriceID:              priceID,
	}

	if submission.ParticipantName == "" ||
		submission.ParticipantEmail == "" ||
		submission.VehicleYear == "" ||
		submission.VehicleMake == "" ||
		submission.VehicleModel == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "At least one image is required", http.StatusBadRequest)
		return
	}

	// DigitalOcean Spaces uses object keys, not real folders. This creates
	// keys like:
	//
	// events/summer-show/SUB-123-ABC123/car-1234abcd.jpg
	folder := fmt.Sprintf("events/%s/%s", event.Slug, submission.ID)

	for i, fileHeader := range files {
		if i >= 5 {
			break
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf(
				"Failed to open uploaded image %q: %v",
				fileHeader.Filename,
				err,
			)
			continue
		}

		fileBytes, readErr := io.ReadAll(file)
		closeErr := file.Close()

		if readErr != nil {
			log.Printf(
				"Failed to read uploaded image %q: %v",
				fileHeader.Filename,
				readErr,
			)
			continue
		}

		if closeErr != nil {
			log.Printf(
				"Failed to close uploaded image %q: %v",
				fileHeader.Filename,
				closeErr,
			)
		}

		if len(fileBytes) == 0 {
			log.Printf(
				"Skipping empty uploaded image %q",
				fileHeader.Filename,
			)
			continue
		}

		contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = http.DetectContentType(fileBytes)
		}

		imageURL, err := services.UploadFile(
			bytes.NewReader(fileBytes),
			fileHeader.Filename,
			contentType,
			folder,
		)
		if err != nil {
			log.Printf(
				"Failed to upload image %q for submission %s: %v",
				fileHeader.Filename,
				submission.ID,
				err,
			)
			continue
		}

		submission.Images = append(submission.Images, imageURL)
	}

	if len(submission.Images) == 0 {
		http.Error(
			w,
			"Failed to upload any images",
			http.StatusInternalServerError,
		)
		return
	}

	requiresApproval := priceRequiresApproval(priceID)
	submission.RequiresApproval = requiresApproval

	imagesJSON, err := json.Marshal(submission.Images)
	if err != nil {
		http.Error(
			w,
			"Failed to encode uploaded images",
			http.StatusInternalServerError,
		)
		return
	}

	db := services.GetDB()
	if db == nil {
		log.Printf(
			"Database is unavailable while storing submission %s",
			submission.ID,
		)
		http.Error(
			w,
			"Unable to store submission",
			http.StatusInternalServerError,
		)
		return
	}

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		err := tx.Exec(`
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
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			return fmt.Errorf("insert vehicle submission: %w", err)
		}

		answers, err := validateSubmittedAnswers(
			tx,
			priceID,
			submittedAnswers,
		)
		if err != nil {
			return err
		}

		for index := range answers {
			answers[index].SubmissionID = submission.ID

			if err := tx.Create(&answers[index]).Error; err != nil {
				return fmt.Errorf(
					"insert requirement answer: %w",
					err,
				)
			}
		}

		if requiresApproval {
			message := buildSubmissionConfirmationEmail(submission)

			return services.QueueEmailTx(
				r.Context(),
				tx,
				submission.ID,
				message,
			)
		}

		if priceID == "" {
			return nil
		}

		paymentLink, err := createSubmissionPaymentLink(submission)
		if err != nil {
			log.Printf(
				"Failed to create payment link for auto-approved submission %s: %v",
				submission.ID,
				err,
			)

			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "https://eurohaus.shop"
			}

			paymentLink = fmt.Sprintf(
				"%s/events/%s?submission=%s",
				strings.TrimRight(baseURL, "/"),
				url.PathEscape(submission.EventSlug),
				url.QueryEscape(submission.ID),
			)
		}

		message := buildApprovalEmail(submission, paymentLink)

		return services.QueueEmailTx(
			r.Context(),
			tx,
			submission.ID,
			message,
		)
	})

	if err != nil {
		log.Printf(
			"Failed to store submission %s and enqueue email: %v",
			submission.ID,
			err,
		)

		http.Error(
			w,
			"Failed to store submission",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(submission); err != nil {
		log.Printf(
			"Failed to encode submission response %s: %v",
			submission.ID,
			err,
		)
	}
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

func ApproveSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}

	// An empty request body is allowed because notes are optional.
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil &&
			!errors.Is(err, io.EOF) {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.CheckoutSessionID == "" &&
		submission.PaymentIntentID == "" {
		recoveredSessionID, recoveredPaymentIntentID, recoveryErr :=
			recoverSubmissionPaymentReferences(
				r.Context(),
				submissionID,
			)

		if recoveryErr != nil {
			log.Printf(
				"No Stripe payment references recovered for submission %s: %v",
				submissionID,
				recoveryErr,
			)
		} else {
			submission.CheckoutSessionID = recoveredSessionID
			submission.PaymentIntentID = recoveredPaymentIntentID

			log.Printf(
				"Recovered Stripe payment references for submission %s: session=%s payment_intent=%s",
				submissionID,
				recoveredSessionID,
				recoveredPaymentIntentID,
			)
		}
	}

	if submission.Status == "approved" {
		http.Error(
			w,
			"Submission is already approved",
			http.StatusBadRequest,
		)
		return
	}

	if submission.Status == "denied" || submission.Status == "revoked" {
		http.Error(
			w,
			"Submission cannot be approved from its current status",
			http.StatusBadRequest,
		)
		return
	}

	submission.Status = "approved"
	submission.ReviewNotes = strings.TrimSpace(req.Notes)

	paymentLink := ""

	// If an unpaid checkout already exists, reuse it.
	if submission.CheckoutSessionID != "" || submission.PaymentIntentID != "" {
		params := &stripe.CheckoutSessionParams{}

		checkoutSession, getErr := session.Get(
			submission.CheckoutSessionID,
			params,
		)

		if getErr != nil {
			log.Printf(
				"Failed to retrieve checkout session %s: %v",
				submission.CheckoutSessionID,
				getErr,
			)
		} else if checkoutSession.PaymentStatus != "paid" {
			paymentLink = checkoutSession.URL
		}
	}

	// If no usable payment link exists, create one.
	if paymentLink == "" &&
		submission.TicketID == "" &&
		submission.PaymentIntentID == "" {
		paymentLink, err = createSubmissionPaymentLink(*submission)
		if err != nil {
			log.Printf(
				"Failed to create payment link for approved submission %s: %v",
				submissionID,
				err,
			)

			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "https://eurohaus.shop"
			}

			paymentLink = fmt.Sprintf(
				"%s/events/%s?submission=%s",
				strings.TrimRight(baseURL, "/"),
				url.PathEscape(submission.EventSlug),
				url.QueryEscape(submission.ID),
			)
		}
	}

	db := services.GetDB()

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			UPDATE vehicle_submissions
			SET status = 'approved',
				reviewed_at = NOW(),
				reviewed_by = 'admin',
				review_notes = ?
			WHERE id = ?
				AND status NOT IN ('approved', 'revoked')
		`,
			submissionNullableString(submission.ReviewNotes),
			submissionID,
		)

		if result.Error != nil {
			return fmt.Errorf(
				"update approved submission: %w",
				result.Error,
			)
		}

		if result.RowsAffected != 1 {
			return fmt.Errorf(
				"submission %s was changed by another request",
				submissionID,
			)
		}

		// If a ticket already exists, the payment webhook has already
		// handled ticket delivery. Do not enqueue a duplicate approval email.
		if submission.TicketID != "" {
			return nil
		}

		message := buildApprovalEmail(
			*submission,
			paymentLink,
		)

		if err := services.QueueEmailTx(
			r.Context(),
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf(
				"enqueue approval email: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to approve submission %s: %v",
			submissionID,
			err,
		)
		http.Error(
			w,
			"Failed to approve submission",
			http.StatusInternalServerError,
		)
		return
	}

	paymentCaptured := false
	paymentProcessing := false
	paymentCaptureError := ""

	hasPaymentReference :=
		strings.TrimSpace(submission.CheckoutSessionID) != "" ||
			strings.TrimSpace(submission.PaymentIntentID) != ""

	if hasPaymentReference {
		var captureErr error

		paymentCaptured, paymentProcessing, captureErr =
			captureSubmissionPayment(submissionID)

		if captureErr != nil {
			paymentCaptureError = captureErr.Error()

			log.Printf(
				"Payment capture failed for submission %s: %v",
				submissionID,
				captureErr,
			)
		}
	}

	updatedSubmission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(
			w,
			"Submission approved but could not be retrieved",
			http.StatusInternalServerError,
		)
		return
	}

	responseStatus := http.StatusOK

	response := map[string]interface{}{
		"submission":        updatedSubmission,
		"payment_captured":   paymentCaptured,
		"payment_processing": paymentProcessing,
		"message":            "Submission approved successfully. Email delivery has been queued.",
	}

	if paymentCaptureError != "" {
		response["payment_capture_error"] = paymentCaptureError
		response["message"] =
			"Submission was approved, but payment capture requires attention."
	}

	if paymentProcessing {
		response["message"] =
			"Submission was approved. Payment is still processing."
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseStatus)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(
			"Failed to encode approval response for %s: %v",
			submissionID,
			err,
		)
	}
}

func DenySubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		Notes string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Review notes are required for denial",
			http.StatusBadRequest,
		)
		return
	}

	req.Notes = strings.TrimSpace(req.Notes)
	if req.Notes == "" {
		http.Error(
			w,
			"Review notes are required for denial",
			http.StatusBadRequest,
		)
		return
	}

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	submission.ReviewNotes = req.Notes
	submission.Status = "denied"

	db := services.GetDB()

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE vehicle_submissions
			SET status = 'denied',
				reviewed_at = NOW(),
				reviewed_by = 'admin',
				review_notes = ?
			WHERE id = ?
		`,
			req.Notes,
			submissionID,
		).Error; err != nil {
			return fmt.Errorf("update denied submission: %w", err)
		}

		message := buildDenialEmail(*submission, req.Notes)

		if err := services.QueueEmailTx(
			r.Context(),
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf("enqueue denial email: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to deny submission %s: %v",
			submissionID,
			err,
		)
		http.Error(
			w,
			"Failed to update submission",
			http.StatusInternalServerError,
		)
		return
	}

	updatedSubmission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(
			w,
			"Submission was updated but could not be retrieved",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedSubmission); err != nil {
		log.Printf(
			"Failed to encode denied submission %s: %v",
			submissionID,
			err,
		)
	}
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
		SubmissionID string `json:"submission_id"`
		PriceID      string `json:"price_id"`
		EventName    string `json:"event_name"`
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
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
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
	if db == nil {
		http.Error(
			w,
			"Database is unavailable",
			http.StatusInternalServerError,
		)
		return
	}

	result := db.WithContext(r.Context()).Exec(`
		UPDATE vehicle_submissions
		SET
			checkout_session_id = ?,
			price_id = NULLIF(?, ''),
			checkout_created_at = NOW()
		WHERE id = ?
	`,
		s.ID,
		strings.TrimSpace(req.PriceID),
		req.SubmissionID,
	)

	if result.Error != nil {
		log.Printf(
			"Failed to persist checkout session %s for submission %s: %v",
			s.ID,
			req.SubmissionID,
			result.Error,
		)

		http.Error(
			w,
			"Checkout session was created but could not be saved. Contact support before retrying.",
			http.StatusInternalServerError,
		)
		return
	}

	if result.RowsAffected != 1 {
		http.Error(
			w,
			"Submission not found while saving checkout session",
			http.StatusNotFound,
		)
		return
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

	req.SubmissionID = strings.TrimSpace(req.SubmissionID)
	req.PriceID = strings.TrimSpace(req.PriceID)

	if req.SubmissionID == "" {
		http.Error(
			w,
			"Submission ID is required",
			http.StatusBadRequest,
		)
		return
	}

	if req.PriceID == "" {
		http.Error(
			w,
			"Price ID is required",
			http.StatusBadRequest,
		)
		return
	}

	// Submission-backed checkouts currently support only one submission/ticket
	req.Quantity = 1

	submission, err := getSubmissionByID(req.SubmissionID)
	if err != nil {
		log.Printf(
			"Failed to load submission %q for participant checkout: %v",
			req.SubmissionID,
			err,
		)

		if errors.Is(err, sql.ErrNoRows) ||
			err.Error() == "submission not found" {
			http.Error(w, "Submission not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to load submission", http.StatusInternalServerError)
		return
	}

	event, err := findEventByID(
		r.Context(),
		submission.EventID,
	)
	if err != nil {
		http.Error(
			w,
			"Event not found",
			http.StatusInternalServerError,
		)
		return
	}

	var eventPrice models.PriceInfo

	err = services.GetDB().
		WithContext(r.Context()).
		Where(
			"id = ? AND stripe_product_id = ? AND active = TRUE",
			req.PriceID,
			event.StripeProductID,
		).
		First(&eventPrice).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(
			w,
			"Price does not belong to this event",
			http.StatusBadRequest,
		)
		return
	}

	if err != nil {
		http.Error(
			w,
			"Unable to validate submission price",
			http.StatusInternalServerError,
		)
		return
	}

	// requiresApproval := eventPrice.RequiresApproval
	requiresApproval := true

	if !requiresApproval &&
		submission.Status != "approved" {
		http.Error(
			w,
			"This price does not require a submission approval flow",
			http.StatusBadRequest,
		)
		return
	}

	needsManualCapture := submission.Status != "approved"

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
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
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

	if err := persistCheckoutSessionAfterCreation(
		r.Context(),
		req.SubmissionID,
		s.ID,
		req.PriceID,
		req.PromotionCode,
		requiresApproval,
	); err != nil {
		log.Printf(
			"Checkout session %s was created in Stripe but could not be persisted for submission %s: %v",
			s.ID,
			req.SubmissionID,
			err,
		)

		http.Error(
			w,
			"Checkout session was created but could not be saved. Contact support before retrying.",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id":        s.ID,
		"session_url":       s.URL,
		"requires_approval": requiresApproval,
	})
}

func GetSubmissionPaymentStatus(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	type PaymentStatus struct {
		HasPayment      bool   `json:"has_payment"`
		PaymentStatus   string `json:"payment_status,omitempty"`
		PaymentAmount   int64  `json:"payment_amount,omitempty"`
		PaymentCurrency string `json:"payment_currency,omitempty"`
		CheckoutURL     string `json:"checkout_url,omitempty"`
		CheckoutExists  bool   `json:"checkout_exists"`
		PaymentIntentID string `json:"payment_intent_id,omitempty"`
		ErrorMessage    string `json:"error_message,omitempty"`
	}

	status := PaymentStatus{
		PaymentIntentID: submission.PaymentIntentID,
	}

	if submission.CheckoutSessionID != "" {
		params := &stripe.CheckoutSessionParams{}
		params.AddExpand("payment_intent")

		sess, sessionErr := session.Get(
			submission.CheckoutSessionID,
			params,
		)

		if sessionErr == nil {
			status.CheckoutExists = true
			status.PaymentStatus = string(sess.PaymentStatus)
			status.PaymentAmount = sess.AmountTotal
			status.PaymentCurrency = string(sess.Currency)
			status.CheckoutURL = sess.URL

			if sess.PaymentIntent != nil &&
				sess.PaymentIntent.ID != "" {
				status.PaymentIntentID = sess.PaymentIntent.ID
				status.HasPayment = true
			}
		} else {
			status.ErrorMessage = fmt.Sprintf(
				"Checkout session not found: %v",
				sessionErr,
			)
		}
	}

	if status.PaymentIntentID != "" {
		pi, paymentErr := paymentintent.Get(
			status.PaymentIntentID,
			nil,
		)

		if paymentErr == nil {
			status.HasPayment = true
			status.PaymentStatus = string(pi.Status)
			status.PaymentAmount = pi.Amount
			status.PaymentCurrency = string(pi.Currency)
		} else if status.ErrorMessage == "" {
			status.ErrorMessage = fmt.Sprintf(
				"PaymentIntent not found: %v",
				paymentErr,
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func CreateSubmissionPayment(
	w http.ResponseWriter,
	r *http.Request,
) {
	submissionID := mux.Vars(r)["submissionId"]

	var req struct {
		PriceID   string `json:"priceId"`
		EventName string `json:"eventName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.PriceID = strings.TrimSpace(req.PriceID)

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(
			w,
			"Submission must be approved first",
			http.StatusBadRequest,
		)
		return
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(
			string(stripe.CheckoutSessionModePayment),
		),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/success?session_id={CHECKOUT_SESSION_ID}",
				strings.TrimRight(os.Getenv("BASE_URL"), "/"),
			),
		),
		CancelURL: stripe.String(
			fmt.Sprintf(
				"%s/event/%s",
				strings.TrimRight(os.Getenv("BASE_URL"), "/"),
				url.PathEscape(submission.EventID),
			),
		),
		Metadata: map[string]string{
			"submission_id": submission.ID,
			"event_id":      submission.EventID,
			"event_name":    req.EventName,
			"type":          "participant_registration",
		},
		CustomerEmail: stripe.String(submission.ParticipantEmail),
	}

	checkoutSession, err := session.New(params)
	if err != nil {
		log.Printf(
			"Failed to create payment session for submission %s: %v",
			submissionID,
			err,
		)
		http.Error(
			w,
			"Failed to create payment session",
			http.StatusInternalServerError,
		)
		return
	}

	db := services.GetDB()

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			UPDATE vehicle_submissions
			SET
				checkout_session_id = ?,
				price_id = NULLIF(?, ''),
				checkout_created_at = NOW()
			WHERE id = ?
		`,
			checkoutSession.ID,
			req.PriceID,
			submissionID)

		if result.Error != nil {
			return fmt.Errorf(
				"update submission checkout session: %w",
				result.Error,
			)
		}

		if result.RowsAffected != 1 {
			return fmt.Errorf(
				"submission %s was not found while saving checkout session",
				submissionID,
			)
		}

		message := buildApprovalEmail(
			*submission,
			checkoutSession.URL,
		)

		if err := services.QueueEmailTx(
			r.Context(),
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf(
				"enqueue approval email: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to update submission %s and enqueue approval email: %v",
			submissionID,
			err,
		)
		http.Error(
			w,
			"Failed to create submission payment",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"sessionUrl": checkoutSession.URL,
		"sessionId":  checkoutSession.ID,
	}); err != nil {
		log.Printf(
			"Failed to encode payment response for submission %s: %v",
			submissionID,
			err,
		)
	}
}

func ResendApprovalEmail(
	w http.ResponseWriter,
	r *http.Request,
) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(
			w,
			"Submission must be approved to resend email",
			http.StatusBadRequest,
		)
		return
	}

	paymentLink := ""

	if submission.CheckoutSessionID != "" || submission.PaymentIntentID != "" {
		params := &stripe.CheckoutSessionParams{}

		checkoutSession, getErr := session.Get(
			submission.CheckoutSessionID,
			params,
		)

		if getErr != nil {
			log.Printf(
				"Failed to retrieve checkout session %s: %v",
				submission.CheckoutSessionID,
				getErr,
			)
		} else if checkoutSession.PaymentStatus != "paid" {
			paymentLink = checkoutSession.URL
		}
	}

	if paymentLink == "" {
		paymentLink, err = createSubmissionPaymentLink(*submission)
		if err != nil {
			log.Printf(
				"Failed to create replacement payment link for %s: %v",
				submissionID,
				err,
			)
		}
	}

	message := buildApprovalEmail(*submission, paymentLink)

	db := services.GetDB()

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := services.QueueEmailTx(
			r.Context(),
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf("enqueue approval resend: %w", err)
		}

		if err := tx.Exec(`
			UPDATE vehicle_submissions
			SET approval_email_resent = TRUE,
				email_resent_count = COALESCE(email_resent_count, 0) + 1
			WHERE id = ?
		`, submissionID).Error; err != nil {
			return fmt.Errorf(
				"update approval resend state: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to enqueue approval resend for %s: %v",
			submissionID,
			err,
		)
		http.Error(
			w,
			"Failed to queue approval email",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Approval email queued successfully",
		"paymentLink": paymentLink != "",
	}); err != nil {
		log.Printf(
			"Failed to encode resend response for %s: %v",
			submissionID,
			err,
		)
	}
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

		hasPaymentData :=
			submission.CheckoutSessionID != "" ||
				submission.PaymentIntentID != ""

		if status == "approved" {
			if !hasPaymentData {
				hasIssue = true
				issues = appendUniqueIssue(issues, "no_payment")
			}

			sessionPaid := false

			if submission.CheckoutSessionID != "" {
				params := &stripe.CheckoutSessionParams{}
				params.AddExpand("payment_intent")

				sess, sessionErr := session.Get(
					submission.CheckoutSessionID,
					params,
				)

				if sessionErr != nil {
					hasIssue = true
					issues = appendUniqueIssue(issues, "orphaned_checkout_session")

					log.Printf(
						"Checkout session %s for submission %s cannot be retrieved: %v",
						submission.CheckoutSessionID,
						submissionID,
						sessionErr,
					)
				} else {
					sessionPaid = sess.PaymentStatus == "paid"

					if submission.PaymentIntentID == "" &&
						sess.PaymentIntent != nil &&
						sess.PaymentIntent.ID != "" {
						hasIssue = true
						issues = appendUniqueIssue(issues, "missing_payment_intent")
					}

					if !sessionPaid {
						hasIssue = true

						if sess.ExpiresAt > 0 &&
							sess.ExpiresAt < time.Now().Unix() {
							issues = appendUniqueIssue(issues, "payment_expired")
						} else {
							issues = appendUniqueIssue(issues, "payment_incomplete")
						}
					}
				}
			}

			if submission.PaymentIntentID != "" {
				pi, paymentErr := paymentintent.Get(
					submission.PaymentIntentID,
					nil,
				)

				if paymentErr != nil {
					hasIssue = true
					issues = appendUniqueIssue(issues, "payment_intent_check_failed")

					log.Printf(
						"PaymentIntent %s for submission %s cannot be retrieved: %v",
						submission.PaymentIntentID,
						submissionID,
						paymentErr,
					)
				} else {
					switch pi.Status {
					case stripe.PaymentIntentStatusSucceeded:
						sessionPaid = true

					case stripe.PaymentIntentStatusRequiresCapture:
						// This is valid for an approved submission that is waiting
						// for administrative capture.
						if pi.CaptureMethod != stripe.PaymentIntentCaptureMethodManual {
							hasIssue = true
							issues = appendUniqueIssue(issues, "capture_failed")
						}

					case stripe.PaymentIntentStatusProcessing:
						hasIssue = true
						issues = appendUniqueIssue(issues, "payment_processing")

					default:
						hasIssue = true
						issues = appendUniqueIssue(issues, "payment_not_succeeded")
					}
				}
			}

			if submission.CheckoutSessionID == "" &&
				submission.PaymentIntentID != "" {
				hasIssue = true
				issues = appendUniqueIssue(issues, "incomplete_payment_process")
			}

			if sessionPaid &&
				submission.TicketID == "" &&
				submission.PaymentIntentID != "" {
				hasIssue = true
				issues = appendUniqueIssue(issues, "no_ticket_created")
			}

			if submission.CheckoutSessionID == "" {
				hasIssue = true
				issues = appendUniqueIssue(issues, "missing_checkout_data")
			}
		}

		if status != "approved" && hasPaymentData {
			hasIssue = true
			issues = appendUniqueIssue(issues, "payment_without_approval")
			fmt.Printf("Submission %s has payment data but not approved", submissionID)
		}

		if status == "pending" && hasPaymentData {
			hasIssue = true
			issues = appendUniqueIssue(issues, "pending_with_payment")
			fmt.Printf("Submission %s is pending with payment data", submissionID)
		}

		if status == "pending" {
			submittedAt, err := time.Parse(time.RFC3339, submission.SubmittedAt)
			if err == nil {
				twoWeeksAgo := time.Now().AddDate(0, 0, -14)
				if submittedAt.Before(twoWeeksAgo) {
					hasIssue = true
					issues = appendUniqueIssue(issues, "pending_too_long")
					fmt.Printf("Submission %s pending too long", submissionID)
				}
			}
		}

		if hasPaymentData {
			hasIssue = true
			if !contains(issues, "no_ticket_created") {
				issues = appendUniqueIssue(issues, "no_ticket_created")
				fmt.Printf("Submission %s has payment data but no ticket", submissionID)
			}
		}

		if submission.PaymentIntentID != "" && submission.CheckoutSessionID == "" {
			hasIssue = true
			if !contains(issues, "incomplete_payment_process") {
				issues = appendUniqueIssue(issues, "incomplete_payment_process")
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

	response := map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("Email updated from %s to %s", oldEmail, req.NewEmail),
		"submission": updatedSubmission,
	}

	if req.ResendEmail && submission.Status == "approved" {
		paymentIsPaid := false
		paymentLink := ""

		if submission.CheckoutSessionID != "" || submission.PaymentIntentID != "" {
			params := &stripe.CheckoutSessionParams{}

			checkoutSession, getErr := session.Get(
				submission.CheckoutSessionID,
				params,
			)

			if getErr != nil {
				log.Printf(
					"Failed to retrieve checkout session %s: %v",
					submission.CheckoutSessionID,
					getErr,
				)
			} else {
				paymentIsPaid = checkoutSession.PaymentStatus == "paid"

				if !paymentIsPaid {
					paymentLink = checkoutSession.URL
				}
			}
		}

		if !paymentIsPaid && paymentLink == "" {
			paymentLink, _ = createSubmissionPaymentLink(
				*updatedSubmission,
			)
		}

		message := buildApprovalEmail(
			*updatedSubmission,
			paymentLink,
		)

		err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
			if err := services.QueueEmailTx(
				r.Context(),
				tx,
				submissionID,
				message,
			); err != nil {
				return fmt.Errorf(
					"enqueue updated approval email: %w",
					err,
				)
			}

			return tx.Exec(`
				UPDATE vehicle_submissions
				SET approval_email_resent = TRUE,
					email_resent_count = COALESCE(email_resent_count, 0) + 1
				WHERE id = ?
			`, submissionID).Error
		})

		if err != nil {
			log.Printf(
				"Failed to queue updated approval email for %s: %v",
				submissionID,
				err,
			)
			response["email_resent"] = false
			response["email_error"] = "Failed to queue approval email"
		} else {
			response["email_resent"] = true
			response["emails_queued"] = []string{"approval"}
			response["message"] = fmt.Sprintf(
				"Email updated from %s to %s and approval email queued",
				oldEmail,
				req.NewEmail,
			)
		}
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
		"errors":             []string{},
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

	err = db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			UPDATE vehicle_submissions
			SET status = 'revoked',
				revoked_at = NOW(),
				revoked_by = 'admin',
				revocation_reason = ?
			WHERE id = ?
				AND status = 'approved'
		`,
			req.Reason,
			submissionID,
		)

		if result.Error != nil {
			return fmt.Errorf(
				"update revoked submission: %w",
				result.Error,
			)
		}

		if result.RowsAffected != 1 {
			return fmt.Errorf(
				"submission %s is no longer approved",
				submissionID,
			)
		}

		message := buildRevocationEmail(
			*submission,
			req.Reason,
		)

		if err := services.QueueEmailTx(
			r.Context(),
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf(
				"enqueue revocation email: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to revoke submission and queue email %s: %v",
			submissionID,
			err,
		)

		http.Error(
			w,
			"Failed to revoke submission",
			http.StatusInternalServerError,
		)
		return
	}

	result["submission_revoked"] = true

	updatedSubmission, _ := getSubmissionByID(submissionID)
	result["submission"] = updatedSubmission

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func RepairSubmissionPayment(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	sessionID := submission.CheckoutSessionID
	paymentIntentID := submission.PaymentIntentID

	if sessionID == "" || paymentIntentID == "" {
		recoveredSessionID, recoveredPaymentIntentID, recoveryErr :=
			recoverSubmissionPaymentReferences(
				r.Context(),
				submissionID,
			)

		if recoveryErr != nil {
			http.Error(
				w,
				fmt.Sprintf("Unable to recover Stripe payment: %v", recoveryErr),
				http.StatusUnprocessableEntity,
			)
			return
		}

		sessionID = recoveredSessionID
		paymentIntentID = recoveredPaymentIntentID
	}

	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to retrieve payment intent: %v", err),
			http.StatusBadGateway,
		)
		return
	}

	updates := map[string]interface{}{
		"checkout_session_id":  sessionID,
		"payment_intent_id":    paymentIntentID,
		"checkout_completed":   true,
		"checkout_completed_at": gorm.Expr("COALESCE(checkout_completed_at, NOW())"),
	}

	switch pi.Status {
	case stripe.PaymentIntentStatusRequiresCapture:
		updates["awaiting_approval"] = true
		updates["payment_succeeded_before_approval"] = true
		updates["payment_succeeded_at"] =
			gorm.Expr("COALESCE(payment_succeeded_at, NOW())")

	case stripe.PaymentIntentStatusSucceeded:
		updates["payment_succeeded_at"] =
			gorm.Expr("COALESCE(payment_succeeded_at, NOW())")

	default:
		updates["awaiting_approval"] = false
	}

	if err := services.GetDB().
		WithContext(r.Context()).
		Model(&models.VehicleSubmission{}).
		Where("id = ?", submissionID).
		Updates(updates).
		Error; err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to repair submission: %v", err),
			http.StatusInternalServerError,
		)
		return
	}

	updated, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(
			w,
			"Payment repaired but submission could not be reloaded",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submission":       updated,
		"payment_intent_id": paymentIntentID,
		"checkout_session_id": sessionID,
		"stripe_status":     pi.Status,
	})
}

func RetrySubmissionApproval(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.Status != "approved" {
		http.Error(
			w,
			"Submission must be approved before retrying payment",
			http.StatusBadRequest,
		)
		return
	}

	paymentCaptured := false
	paymentProcessing := false

	if submission.CheckoutSessionID != "" ||
		submission.PaymentIntentID != "" {
		var captureErr error

		paymentCaptured, paymentProcessing, captureErr =
			captureSubmissionPayment(submissionID)

		if captureErr != nil {
			http.Error(
				w,
				fmt.Sprintf("Payment retry failed: %v", captureErr),
				http.StatusUnprocessableEntity,
			)
			return
		}
	}

	updatedSubmission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(
			w,
			"Payment retry completed but submission could not be reloaded",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"message":          "Approval/payment processing retried",
		"paymentCaptured":  paymentCaptured,
		"paymentProcessing": paymentProcessing,
		"submission":       updatedSubmission,
	})
}

func RetrySubmissionTicket(w http.ResponseWriter, r *http.Request) {
	submissionID := mux.Vars(r)["submissionId"]

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if submission.PaymentIntentID == "" {
		http.Error(
			w,
			"Submission has no payment intent",
			http.StatusUnprocessableEntity,
		)
		return
	}

	if submission.TicketID != "" {
		http.Error(
			w,
			"Submission already has a ticket",
			http.StatusConflict,
		)
		return
	}

	pi, err := paymentintent.Get(
		submission.PaymentIntentID,
		nil,
	)
	if err != nil {
		http.Error(
			w,
			"Unable to retrieve payment intent",
			http.StatusBadGateway,
		)
		return
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		http.Error(
			w,
			fmt.Sprintf(
				"Payment is not complete; current status is %s",
				pi.Status,
			),
			http.StatusUnprocessableEntity,
		)
		return
	}

	event, err := findEventByID(
		r.Context(),
		submission.EventID,
	)
	if err != nil {
		http.Error(
			w,
			"Event not found",
			http.StatusInternalServerError,
		)
		return
	}

	checkoutSessionID := submission.CheckoutSessionID

	err = services.GetDB().
		WithContext(r.Context()).
		Transaction(func(tx *gorm.DB) error {
			ticketToken, err := insertParticipantTicketTx(
				r.Context(),
				tx,
				submission,
				event.Name,
				checkoutSessionID,
				submission.PaymentIntentID,
				"Participant",
				1,
			)
			if err != nil {
				return err
			}

			if err := tx.Exec(`
				UPDATE vehicle_submissions
				SET
					ticket_id = ?,
					ticket_created_at = NOW(),
					payment_captured = TRUE,
					payment_captured_at = COALESCE(
						payment_captured_at,
						NOW()
					)
				WHERE id = ?
			`,
				ticketToken,
				submissionID,
			).Error; err != nil {
				return fmt.Errorf(
					"update ticket state: %w",
					err,
				)
			}

			message, err := buildParticipantTicketEmail(
				submission,
				ticketToken,
				event.Name,
			)
			if err != nil {
				return err
			}

			if err := services.QueueEmailTx(
				r.Context(),
				tx,
				submissionID,
				message,
			); err != nil {
				return fmt.Errorf(
					"queue ticket email: %w",
					err,
				)
			}

			return nil
		})

	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to retry ticket processing: %v", err),
			http.StatusInternalServerError,
		)
		return
	}

	updatedSubmission, err := getSubmissionByID(submissionID)
	if err != nil {
		http.Error(
			w,
			"Ticket created but submission could not be reloaded",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Ticket and ticket email queued",
		"submission": updatedSubmission,
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

func getSubmissionByID(
	submissionID string,
) (*models.VehicleSubmissionDTO, error) {
	db := services.GetDB()

	var (
		imagesJSON []byte

		submittedAt time.Time
		createdAt   sql.NullTime

		reviewedAt          sql.NullTime
		checkoutCreatedAt   sql.NullTime
		checkoutCompletedAt sql.NullTime
		paymentSucceededAt  sql.NullTime
		paymentCapturedAt   sql.NullTime
		approvalEmailSentAt sql.NullTime
		ticketCreatedAt     sql.NullTime
		ticketEmailSentAt   sql.NullTime
		emailUpdatedAt      sql.NullTime
		refundIssuedAt      sql.NullTime
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
			created_at,

			reviewed_at,
			COALESCE(reviewed_by, ''),
			COALESCE(review_notes, ''),

			COALESCE(checkout_session_id, ''),
			checkout_created_at,
			COALESCE(checkout_completed, FALSE),
			checkout_completed_at,

			COALESCE(payment_intent_id, ''),
			COALESCE(payment_succeeded_before_approval, FALSE),
			payment_succeeded_at,
			COALESCE(payment_captured, FALSE),
			payment_captured_at,

			COALESCE(price_id, ''),
			COALESCE(
				(
					SELECT nickname
					FROM prices
					WHERE prices.id = vehicle_submissions.price_id
				),
				''
			),
			COALESCE(promotion_code, ''),

			COALESCE(requires_approval, TRUE),
			COALESCE(awaiting_approval, FALSE),

			COALESCE(approval_email_sent, FALSE),
			approval_email_sent_at,
			COALESCE(approval_email_resent, FALSE),

			COALESCE(ticket_id, ''),
			ticket_created_at,
			COALESCE(ticket_email_sent, FALSE),
			ticket_email_sent_at,

			email_updated_at,
			COALESCE(previous_email, ''),
			COALESCE(email_resent_count, 0),

			COALESCE(refund_id, ''),
			COALESCE(refund_amount, 0),
			refund_issued_at,

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
		&createdAt,

		&reviewedAt,
		&submission.ReviewedBy,
		&submission.ReviewNotes,

		&submission.CheckoutSessionID,
		&checkoutCreatedAt,
		&submission.CheckoutCompleted,
		&checkoutCompletedAt,

		&submission.PaymentIntentID,
		&submission.PaymentSucceededBeforeApproval,
		&paymentSucceededAt,
		&submission.PaymentCaptured,
		&paymentCapturedAt,

		&submission.PriceID,
		&submission.PriceNickname,
		&submission.PromotionCode,

		&submission.RequiresApproval,
		&submission.AwaitingApproval,

		&submission.ApprovalEmailSent,
		&approvalEmailSentAt,
		&submission.ApprovalEmailResent,

		&submission.TicketID,
		&ticketCreatedAt,
		&submission.TicketEmailSent,
		&ticketEmailSentAt,

		&emailUpdatedAt,
		&submission.PreviousEmail,
		&submission.EmailResentCount,

		&submission.RefundID,
		&submission.RefundAmount,
		&refundIssuedAt,

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

	var answers []models.SubmissionRequirementAnswer

	if err := db.
		Preload("Requirement").
		Where("submission_id = ?", submissionID).
		Order("created_at ASC, id ASC").
		Find(&answers).
		Error; err != nil {
			return nil, err
		}

	var requirementAnswers []models.SubmissionRequirementAnswerDTO

	for _, answer := range answers {
		if dto, err := requirementAnswerDTO(answer); err == nil {
			requirementAnswers = append(requirementAnswers, dto)
		}
	}

	submission.RequirementAnswers = requirementAnswers

	if err := json.Unmarshal(imagesJSON, &submission.Images); err != nil {
		submission.Images = []string{}
	}

	submission.SubmittedAt = submittedAt.UTC().Format(time.RFC3339)
	submission.CreatedAt = submissionFormatNullTime(createdAt)

	submission.ReviewedAt = submissionFormatNullTime(reviewedAt)
	submission.CheckoutCreatedAt = submissionFormatNullTime(checkoutCreatedAt)
	submission.CheckoutCompletedAt = submissionFormatNullTime(checkoutCompletedAt)
	submission.PaymentSucceededAt = submissionFormatNullTime(paymentSucceededAt)
	submission.PaymentCapturedAt = submissionFormatNullTime(paymentCapturedAt)
	submission.ApprovalEmailSentAt = submissionFormatNullTime(approvalEmailSentAt)
	submission.TicketCreatedAt = submissionFormatNullTime(ticketCreatedAt)
	submission.TicketEmailSentAt = submissionFormatNullTime(ticketEmailSentAt)
	submission.EmailUpdatedAt = submissionFormatNullTime(emailUpdatedAt)
	submission.RefundIssuedAt = submissionFormatNullTime(refundIssuedAt)
	submission.RevokedAt = submissionFormatNullTime(revokedAt)

	return submission, nil
}

func requirementAnswerDTO(
	answer models.SubmissionRequirementAnswer,
) (models.SubmissionRequirementAnswerDTO, error) {
	var value any

	if err := json.Unmarshal(answer.Value, &value); err != nil {
		return models.SubmissionRequirementAnswerDTO{}, err
	}

	result := models.SubmissionRequirementAnswerDTO{
		ID:            answer.ID,
		RequirementID: answer.RequirementID,
		Value:         value,
	}

	if answer.Requirement != nil {
		result.Key = answer.Requirement.Key
		result.Label = answer.Requirement.Label
		result.Type = answer.Requirement.Type
	}

	return result, nil
}

func captureSubmissionPayment(submissionID string) (bool, bool, error) {
	ctx := context.Background()

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		return false, false, fmt.Errorf("load submission: %w", err)
	}

	var pi *stripe.PaymentIntent

	// Prefer the persisted PaymentIntent. This avoids depending on a
	// Checkout Session being present in the local database.
	if strings.TrimSpace(submission.PaymentIntentID) != "" {
		pi, err = paymentintent.Get(
			strings.TrimSpace(submission.PaymentIntentID),
			nil,
		)
		if err != nil {
			return false, false, fmt.Errorf(
				"retrieve payment intent %s: %w",
				submission.PaymentIntentID,
				err,
			)
		}
	} else if strings.TrimSpace(submission.CheckoutSessionID) != "" {
		pi, err = retrieveCheckoutSessionPaymentIntent(
			ctx,
			submission.CheckoutSessionID,
		)
		if err != nil {
			return false, false, err
		}

		if err := persistPaymentIntentID(
			ctx,
			submissionID,
			pi.ID,
		); err != nil {
			return false, false, err
		}
	} else {
		return false, false, fmt.Errorf(
			"submission has neither checkout session nor payment intent",
		)
	}

	switch pi.Status {
	case stripe.PaymentIntentStatusRequiresCapture:
		if pi.CaptureMethod != stripe.PaymentIntentCaptureMethodManual {
			return false, false, fmt.Errorf(
				"payment intent %s requires capture but uses capture method %s",
				pi.ID,
				pi.CaptureMethod,
			)
		}

		log.Printf(
			"Capturing payment %s for submission %s",
			pi.ID,
			submissionID,
		)

		capturedPI, err := paymentintent.Capture(pi.ID, nil)
		if err != nil {
			return false, false, fmt.Errorf(
				"capture payment intent %s: %w",
				pi.ID,
				err,
			)
		}

		if err := services.GetDB().
			WithContext(ctx).
			Exec(`
				UPDATE vehicle_submissions
				SET
					payment_intent_id = ?,
					payment_captured = TRUE,
					payment_captured_at = NOW(),
					payment_succeeded_at = COALESCE(payment_succeeded_at, NOW())
				WHERE id = ?
			`,
				capturedPI.ID,
				submissionID,
			).Error; err != nil {
			return false, false, fmt.Errorf(
				"persist captured payment state: %w",
				err,
			)
		}

		return true, false, nil

	case stripe.PaymentIntentStatusSucceeded:
		if err := services.GetDB().
			WithContext(ctx).
			Exec(`
				UPDATE vehicle_submissions
				SET
					payment_intent_id = ?,
					payment_succeeded_at = COALESCE(payment_succeeded_at, NOW())
				WHERE id = ?
			`,
				pi.ID,
				submissionID,
			).Error; err != nil {
			return false, false, fmt.Errorf(
				"persist succeeded payment state: %w",
				err,
			)
		}

		return true, false, nil

	case stripe.PaymentIntentStatusProcessing:
		return false, true, nil

	default:
		return false, false, fmt.Errorf(
			"payment intent %s status is %s",
			pi.ID,
			pi.Status,
		)
	}
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

func buildSubmissionConfirmationEmail(
	submission models.VehicleSubmissionDTO,
) *services.EmailMessage {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails": fmt.Sprintf(
			"%s %s %s",
			submission.VehicleYear,
			submission.VehicleMake,
			submission.VehicleModel,
		),
		"EventID":      submission.EventID,
		"SubmissionID": submission.ID,
		"SubmittedAt":  submission.SubmittedAt,
	}

	return &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Received - Euro Haus",
		TemplateID:   "submission-received",
		TemplateData: emailData,
		BodyHTML:     generateSubmissionConfirmationHTML(emailData),
	}
}

func buildApprovalEmail(
	submission models.VehicleSubmissionDTO,
	paymentLink string,
) *services.EmailMessage {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails": fmt.Sprintf(
			"%s %s %s",
			submission.VehicleYear,
			submission.VehicleMake,
			submission.VehicleModel,
		),
		"EventID":     submission.EventID,
		"PaymentLink": paymentLink,
		"ReviewNotes": submission.ReviewNotes,
	}

	return &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Approved - Complete Your Registration",
		TemplateID:   "submission-approved",
		TemplateData: emailData,
		BodyHTML:     generateApprovalEmailHTML(emailData),
	}
}

func buildDenialEmail(
	submission models.VehicleSubmissionDTO,
	reason string,
) *services.EmailMessage {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails": fmt.Sprintf(
			"%s %s %s",
			submission.VehicleYear,
			submission.VehicleMake,
			submission.VehicleModel,
		),
		"EventID":      submission.EventID,
		"DenialReason": reason,
	}

	return &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Update - Euro Haus",
		TemplateID:   "submission-denied",
		TemplateData: emailData,
		BodyHTML:     generateDenialEmailHTML(emailData),
	}
}

func buildRevocationEmail(
	submission models.VehicleSubmissionDTO,
	reason string,
) *services.EmailMessage {
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails": fmt.Sprintf(
			"%s %s %s",
			submission.VehicleYear,
			submission.VehicleMake,
			submission.VehicleModel,
		),
		"EventID":      submission.EventID,
		"DenialReason": reason,
	}

	return &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Vehicle Submission Update - Euro Haus",
		TemplateID:   "submission-denied",
		TemplateData: emailData,
		BodyHTML:     generateDenialEmailHTML(emailData),
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

func recoverSubmissionPaymentReferences(
	ctx context.Context,
	submissionID string,
) (string, string, error) {
	submissionID = strings.TrimSpace(submissionID)
	if submissionID == "" {
		return "", "", fmt.Errorf("submission ID is required")
	}

	params := &stripe.CheckoutSessionListParams{}
	params.Filters.AddFilter(
		"limit",
		"",
		"100",
	)
	params.Filters.AddFilter(
		"metadata[submission_id]",
		"",
		submissionID,
	)

	iter := session.List(params)

	for iter.Next() {
		sess := iter.CheckoutSession()

		if sess == nil || sess.ID == "" {
			continue
		}

		paymentIntentID := paymentIntentIDFromSession(sess)

		if paymentIntentID == "" {
			expandedParams := &stripe.CheckoutSessionParams{}
			expandedParams.AddExpand("payment_intent")

			expanded, err := session.Get(sess.ID, expandedParams)
			if err != nil {
				log.Printf(
					"Failed to expand recovered checkout session %s: %v",
					sess.ID,
					err,
				)
				continue
			}

			paymentIntentID = paymentIntentIDFromSession(expanded)
		}

		if paymentIntentID == "" {
			continue
		}

		if err := persistCheckoutState(
			ctx,
			submissionID,
			sess.ID,
			paymentIntentID,
			sess.Status == stripe.CheckoutSessionStatusComplete,
			false,
			false,
		); err != nil {
			return "", "", err
		}

		return sess.ID, paymentIntentID, nil
	}

	if err := iter.Err(); err != nil {
		return "", "", fmt.Errorf(
			"list checkout sessions for submission %s: %w",
			submissionID,
			err,
		)
	}

	return "", "", fmt.Errorf(
		"no Stripe Checkout Session found for submission %s",
		submissionID,
	)
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

func appendUniqueIssue(issues []string, issue string) []string {
	for _, existing := range issues {
		if existing == issue {
			return issues
		}
	}

	return append(issues, issue)
}
