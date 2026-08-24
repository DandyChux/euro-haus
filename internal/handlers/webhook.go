package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"gorm.io/gorm"

	"github.com/skip2/go-qrcode"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Constants for recovery limits and timeframes
const (
	MaxRecoveryAttempts = 5
	MaxSubmissionAge    = 7 * 24 * time.Hour // 7 days
	MinRecoveryInterval = 1 * time.Hour      // Minimum time between recovery attempts
)

// HandleWebhook processes Stripe webhook events
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	// Add simple rate limiting check
	// rdb := services.GetRedisClient()
	// ctx := context.Background()

	// Use IP-based rate limiting
	// clientIP := r.Header.Get("X-Forwarded-For")
	// if clientIP == "" {
	// 	clientIP = r.RemoteAddr
	// }

	// // Allow max 100 webhook calls per minute per IP
	// rateLimitKey := fmt.Sprintf("webhook:ratelimit:%s:%d", clientIP, time.Now().Unix()/60)
	// count, _ := rdb.Incr(ctx, rateLimitKey).Result()
	// rdb.Expire(ctx, rateLimitKey, 2*time.Minute)

	// if count > 100 {
	// 	log.Printf("Rate limit exceeded for IP %s", clientIP)
	// 	w.WriteHeader(http.StatusTooManyRequests)
	// 	return
	// }

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify webhook signature
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		log.Printf("STRIPE_WEBHOOK_SECRET environment variable not set\n")
		w.WriteHeader(http.StatusOK)
		return
	}

	signatureHeader := r.Header.Get("Stripe-Signature")
	if signatureHeader == "" {
		log.Printf("Missing Stripe-Signature header\n")
		w.WriteHeader(http.StatusOK)
		return
	}

	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v\n", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Printf("Webhook event type: %s\n", event.Type)

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handlePaymentIntentSucceeded(paymentIntent)

	case "checkout.session.completed":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleCheckoutSessionCompleted(checkoutSession)

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handlePaymentIntentFailed(paymentIntent)

	case "checkout.session.expired":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleCheckoutSessionExpired(checkoutSession)

	case "charge.refunded":
		var charge stripe.Charge
		err := json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleChargeRefunded(charge)

	case "payment_intent.canceled":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handlePaymentIntentCanceled(paymentIntent)

	case "checkout.session.async_payment_failed":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleCheckoutSessionPaymentFailed(checkoutSession)

	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func markWebhookKeyProcessed(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	db := services.GetDB()
	result := db.WithContext(ctx).Exec(`
		INSERT INTO webhook_dedup_keys (key, expires_at)
		VALUES (?, ?)
		ON CONFLICT (key) DO NOTHING
	`, key, time.Now().UTC().Add(ttl))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func insertParticipantTicket(
	ctx context.Context,
	submission *models.VehicleSubmissionDTO,
	eventName string,
	checkoutSessionID string,
	paymentIntentID string,
	ticketType string,
	quantity int,
) (string, error) {
	if quantity <= 0 {
		quantity = 1
	}
	if ticketType == "" {
		ticketType = "Participant"
	}
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	token := generateUniqueToken()

	err := services.GetDB().WithContext(ctx).Exec(`
			INSERT INTO tickets (
				token,
				event_id,
				stripe_product_id,
				stripe_session_id,
				stripe_payment_intent_id,
				customer_email,
				customer_name,
				quantity,
				ticket_type,
				status
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, 'active'
			)
		`,
		token,
		submission.EventID,
		submission.EventID,
		submissionNullableString(checkoutSessionID),
		submissionNullableString(paymentIntentID),
		submission.ParticipantEmail,
		submission.ParticipantName,
		quantity,
		ticketType,
	).Error
	if err != nil {
		return "", err
	}

	return token, nil
}

func insertParticipantTicketTx(
	ctx context.Context,
	tx *gorm.DB,
	submission *models.VehicleSubmissionDTO,
	eventName string,
	checkoutSessionID string,
	paymentIntentID string,
	ticketType string,
	quantity int,
) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("database transaction is nil")
	}

	if submission == nil {
		return "", fmt.Errorf("submission is nil")
	}

	if quantity <= 0 {
		quantity = 1
	}

	if ticketType == "" {
		ticketType = "Participant"
	}

	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	token := generateUniqueToken()

	err := tx.WithContext(ctx).Exec(`
		INSERT INTO tickets (
			token,
			event_id,
			stripe_product_id,
			stripe_session_id,
			stripe_payment_intent_id,
			customer_email,
			customer_name,
			quantity,
			ticket_type,
			status
		)
		VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, 'active'
		)
	`,
		token,
		submission.EventID,
		submission.EventID,
		submissionNullableString(checkoutSessionID),
		submissionNullableString(paymentIntentID),
		submission.ParticipantEmail,
		submission.ParticipantName,
		quantity,
		ticketType,
	).Error

	if err != nil {
		return "", fmt.Errorf("insert participant ticket: %w", err)
	}

	return token, nil
}

func isSubmissionEligibleForRecovery(
	submission *models.VehicleSubmissionDTO,
) (bool, string) {
	if submission == nil {
		return false, "Submission is missing"
	}

	if submission.TicketID != "" {
		return false, "Ticket already exists"
	}

	switch submission.Status {
	case "cancelled", "rejected", "denied", "abandoned", "revoked":
		return false, fmt.Sprintf(
			"Submission status is %s",
			submission.Status,
		)
	}

	createdAt := submission.CreatedAt
	if createdAt == "" {
		createdAt = submission.SubmittedAt
	}

	if createdAt != "" {
		created, err := time.Parse(time.RFC3339, createdAt)
		if err == nil && time.Since(created) > MaxSubmissionAge {
			return false, "Submission too old"
		}
	}

	if submission.RecoveryAttempts >= MaxRecoveryAttempts {
		return false, "Maximum recovery attempts reached"
	}

	return true, ""
}

func claimSubmissionRecoveryAttempt(
	ctx context.Context,
	submissionID string,
) (int, bool, error) {
	db := services.GetDB()
	if db == nil {
		return 0, false, fmt.Errorf("database is unavailable")
	}

	var attempt int
	var claimed bool

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempts int

		err := tx.Raw(`
			SELECT COALESCE(recovery_attempts, 0)
			FROM vehicle_submissions
			WHERE id = ?
			FOR UPDATE
		`, submissionID).Scan(&attempts).Error
		if err != nil {
			return fmt.Errorf("load recovery attempts: %w", err)
		}

		if attempts >= MaxRecoveryAttempts {
			return nil
		}

		attempt = attempts + 1

		result := tx.Exec(`
			UPDATE vehicle_submissions
			SET
				recovery_attempts = ?,
				recovery_last_sent_at = NOW()
			WHERE id = ?
		`, attempt, submissionID)

		if result.Error != nil {
			return fmt.Errorf("update recovery attempts: %w", result.Error)
		}

		if result.RowsAffected != 1 {
			return fmt.Errorf(
				"submission %s was not found while claiming recovery",
				submissionID,
			)
		}

		claimed = true
		return nil
	})

	if err != nil {
		return 0, false, err
	}

	return attempt, claimed, nil
}

func handlePaymentIntentSucceeded(pi stripe.PaymentIntent) {
	// Handle successful payment
	fmt.Printf("PaymentIntent succeeded: %s, Amount: %d %s\n",
		pi.ID, pi.Amount, pi.Currency)

	ctx := context.Background()

	if submission, err := findSubmissionByPaymentIntentID(ctx, pi.ID); err == nil {
		handleParticipantPaymentSucceeded(pi, submission.ID)
		return
	}

	var customerEmail, customerName string

	if pi.Customer != nil && pi.Customer.ID != "" {
		customer, err := customer.Get(pi.Customer.ID, nil)
		if err != nil {
			log.Printf(
				"Unable to retrieve customer %s: %v",
				pi.Customer.ID,
				err,
			)
		} else {
			customerEmail = customer.Email
			customerName = customer.Name
		}
	}

	// Don't send email if we don't have an address
	if customerEmail == "" {
		log.Printf("No email address available for payment %s", pi.ID)
		return
	}

	// Format amount for display
	amount := float64(pi.Amount) / 100.0 // Convert from cents
	currency := string(pi.Currency)
	formattedAmount := fmt.Sprintf("%.2f %s", amount, currency)

	// Send payment confirmation email
	emailData := map[string]interface{}{
		"CustomerName":        customerName,
		"PaymentAmount":       formattedAmount,
		"PaymentID":           pi.ID,
		"PaymentDate":         time.Now().Format(time.RFC1123),
		"PaymentStatus":       "Succeeded",
		"TransactionID":       pi.ID,
		"PaymentMethod":       pi.PaymentMethod,
		"StatementDescriptor": pi.StatementDescriptor,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Payment Confirmation - Euro Haus",
		TemplateID:   "payment-confirmation",
		TemplateData: emailData,
		// Fallback if template not found
		BodyHTML: generatePaymentSuccessHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue payment confirmation email for %s: %v",
			customerEmail,
			err,
		)
	}
}

// handleParticipantPaymentSucceeded handles payment success for participant submissions
func handleParticipantPaymentSucceeded(pi stripe.PaymentIntent, submissionID string) {
	log.Printf("Handling participant payment for submission: %s\n", submissionID)

	db := services.GetDB()
	ctx := context.Background()

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		log.Printf("Error retrieving submission data: %v", err)
		return
	}

	if submission.Status != "approved" {
		err := db.WithContext(ctx).Exec(`
			UPDATE vehicle_submissions
			SET payment_succeeded_before_approval = TRUE,
				payment_intent_id = ?,
				payment_succeeded_at = NOW()
			WHERE id = ?
		`,
			pi.ID,
			submissionID,
		).Error

		if err != nil {
			log.Printf(
				"Failed to record payment before approval for submission %s: %v",
				submissionID,
				err,
			)
		}

		return
	}

	evt, err := findEventByID(ctx, submission.EventID)
	if err != nil {
		log.Printf("Error finding event: %v", err)
		return
	}

	if existingTicketID := submission.TicketID; existingTicketID != "" {
		log.Printf("Ticket already exists for submission %s: %s", submissionID, existingTicketID)
		return
	}

	approvalEmailSent := submission.ApprovalEmailSentAt != ""

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticketToken, err := insertParticipantTicketTx(
			ctx,
			tx,
			submission,
			evt.Name,
			submission.CheckoutSessionID,
			pi.ID,
			"Participant",
			1,
		)
		if err != nil {
			return err
		}

		if err := tx.Exec(`
			UPDATE vehicle_submissions
			SET ticket_id = ?,
				payment_intent_id = ?,
				ticket_created_at = NOW()
			WHERE id = ?
		`,
			ticketToken,
			pi.ID,
			submissionID,
		).Error; err != nil {
			return fmt.Errorf(
				"update submission after ticket creation: %w",
				err,
			)
		}

		ticketMessage, err := buildParticipantTicketEmail(
			submission,
			ticketToken,
			evt.Name,
		)
		if err != nil {
			return err
		}

		if err := services.QueueEmailTx(
			ctx,
			tx,
			submissionID,
			ticketMessage,
		); err != nil {
			return fmt.Errorf(
				"enqueue participant ticket email: %w",
				err,
			)
		}

		if !approvalEmailSent {
			approvalMessage := buildApprovalWithTicketEmail(
				submission,
				ticketToken,
			)

			if err := services.QueueEmailTx(
				ctx,
				tx,
				submissionID,
				approvalMessage,
			); err != nil {
				return fmt.Errorf(
					"enqueue approval-with-ticket email: %w",
					err,
				)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to create ticket and enqueue ticket emails for submission %s: %v",
			submissionID,
			err,
		)
		return
	}

	log.Printf(
		"Created ticket and queued ticket emails for submission %s",
		submissionID,
	)
}

func handleCheckoutSessionCompleted(checkoutSession stripe.CheckoutSession) {
	ctx := context.Background()

	log.Printf(
		"Checkout session completed: %s",
		checkoutSession.ID,
	)

	participantSubmission, err :=
		findSubmissionByCheckoutSessionID(ctx, checkoutSession.ID)

	if err == nil && participantSubmission != nil {
		paymentIntentID := paymentIntentIDFromSession(&checkoutSession)

		// The webhook payload may contain only a PaymentIntent ID. Retrieve
		// the session with payment_intent expanded when necessary.
		if paymentIntentID == "" {
			params := &stripe.CheckoutSessionParams{}
			params.AddExpand("payment_intent")

			expandedSession, expandErr :=
				session.Get(checkoutSession.ID, params)

			if expandErr != nil {
				log.Printf(
					"Failed to expand payment intent for checkout session %s: %v",
					checkoutSession.ID,
					expandErr,
				)
			} else {
				paymentIntentID =
					paymentIntentIDFromSession(expandedSession)
			}
		}

		paymentSucceeded := false
		paymentSucceededAt := false

		if paymentIntentID != "" {
			pi, piErr := paymentintent.Get(paymentIntentID, nil)
			if piErr != nil {
				log.Printf(
					"Failed to retrieve payment intent %s for submission %s: %v",
					paymentIntentID,
					participantSubmission.ID,
					piErr,
				)
			} else {
				paymentSucceeded =
					pi.Status == stripe.PaymentIntentStatusSucceeded ||
						pi.Status == stripe.PaymentIntentStatusRequiresCapture

				paymentSucceededAt = paymentSucceeded

				// Manual capture means the funds are authorized but not yet
				// captured. It must remain pending approval.
				if pi.CaptureMethod == stripe.PaymentIntentCaptureMethodManual &&
					pi.Status == stripe.PaymentIntentStatusRequiresCapture {
					if persistErr := persistCheckoutState(
						ctx,
						participantSubmission.ID,
						checkoutSession.ID,
						paymentIntentID,
						true,
						true,
						true,
					); persistErr != nil {
						logPersistenceFailure(
							"persist manual-capture checkout state",
							participantSubmission.ID,
							persistErr,
						)
					}

					log.Printf(
						"Submission %s checkout completed; awaiting approval",
						participantSubmission.ID,
					)
					return
				}
			}
		}

		if persistErr := persistCheckoutState(
			ctx,
			participantSubmission.ID,
			checkoutSession.ID,
			paymentIntentID,
			true,
			paymentSucceeded,
			paymentSucceededAt,
		); persistErr != nil {
			logPersistenceFailure(
				"persist checkout completion",
				participantSubmission.ID,
				persistErr,
			)
		}

		if paymentSucceeded {
			handleParticipantPaymentSucceededFromCheckout(
				checkoutSession,
				participantSubmission.ID,
				paymentIntentID,
			)
			return
		}

		handleParticipantCheckout(checkoutSession, participantSubmission.ID)
		return
	}

	// Existing non-submission checkout behavior remains unchanged.
	handleNonSubmissionCheckoutCompleted(checkoutSession)
}

func handleNonSubmissionCheckoutCompleted(
	checkoutSession stripe.CheckoutSession,
) {
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items.data.price.product")

	fullSession, err := session.Get(
		checkoutSession.ID,
		params,
	)
	if err != nil {
		log.Printf(
			"Error expanding non-submission checkout session %s: %v",
			checkoutSession.ID,
			err,
		)
		return
	}

	processStockUpdates(fullSession)

	productItems := []map[string]interface{}{}
	hasEventTickets := false
	hasPhysicalProducts := false
	totalAmount := 0.0

	for _, lineItem := range fullSession.LineItems.Data {
		if lineItem.Price == nil ||
			lineItem.Price.Product == nil {
			continue
		}

		stripeProductID := lineItem.Price.Product.ID

		_, eventErr := findEventByStripeProductID(
			context.Background(),
			stripeProductID,
		)

		isEventProduct := eventErr == nil
		productType := "product"

		if isEventProduct {
			productType = "event"
			hasEventTickets = true

			storeTicketPurchase(*fullSession, *lineItem)
		} else {
			hasPhysicalProducts = true
		}

		productItems = append(productItems, map[string]interface{}{
			"Name":        lineItem.Price.Product.Name,
			"Description": lineItem.Price.Product.Description,
			"Quantity":    lineItem.Quantity,
			"Price": float64(
				lineItem.Price.UnitAmount,
			) / 100.0,
			"Currency": string(
				lineItem.Price.Currency,
			),
			"Subtotal": float64(
				lineItem.AmountSubtotal,
			) / 100.0,
			"Type": productType,
		})

		totalAmount += float64(
			lineItem.AmountTotal,
		) / 100.0
	}

	customerEmail := fullSession.CustomerEmail

	if customerEmail == "" &&
		fullSession.Customer != nil {
		customerEmail = fullSession.Customer.Email
	}

	customerName := ""

	if fullSession.CustomerDetails != nil {
		customerName = fullSession.CustomerDetails.Name
	}

	if err := ProcessBundledProducts(
		fullSession.ID,
		customerEmail,
		customerName,
	); err != nil {
		log.Printf(
			"Error processing bundled products for session %s: %v",
			fullSession.ID,
			err,
		)
	}

	if customerEmail == "" {
		log.Printf(
			"No email address available for checkout session %s",
			fullSession.ID,
		)
		return
	}

	orderData := map[string]interface{}{
		"CustomerName":     customerName,
		"OrderID":          fullSession.ID,
		"OrderDate":        time.Now().Format(time.RFC1123),
		"Products":         productItems,
		"TotalAmount":      totalAmount,
		"Currency":         string(fullSession.Currency),
		"HasEventTickets":  hasEventTickets,
		"HasPhysicalItems": hasPhysicalProducts,
		"ShippingAddress":  formatShippingAddress(fullSession),
	}

	message := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Order Confirmation - Euro Haus",
		TemplateID:   "order-confirmation",
		TemplateData: orderData,
		BodyHTML:     generateOrderConfirmationHTML(orderData),
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		message,
	); err != nil {
		log.Printf(
			"Failed to queue order confirmation email for session %s: %v",
			fullSession.ID,
			err,
		)
	}
}

func handleParticipantPaymentSucceededFromCheckout(
	checkoutSession stripe.CheckoutSession,
	submissionID string,
	paymentIntentID string,
) {
	if strings.TrimSpace(paymentIntentID) == "" {
		log.Printf(
			"Cannot process completed checkout for submission %s: missing payment intent",
			submissionID,
		)
		return
	}

	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		log.Printf(
			"Failed to retrieve payment intent %s for submission %s: %v",
			paymentIntentID,
			submissionID,
			err,
		)
		return
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		log.Printf(
			"Payment intent %s for submission %s is %s; ticket will not be created",
			paymentIntentID,
			submissionID,
			pi.Status,
		)
		return
	}

	handleParticipantPaymentSucceededWithSession(
		*pi,
		submissionID,
		checkoutSession.ID,
	)
}

func handleParticipantPaymentSucceededWithSession(
	pi stripe.PaymentIntent,
	submissionID string,
	checkoutSessionID string,
) {
	log.Printf(
		"Handling successful participant payment for submission %s",
		submissionID,
	)

	ctx := context.Background()
	db := services.GetDB()

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		log.Printf(
			"Failed to load submission %s after payment success: %v",
			submissionID,
			err,
		)
		return
	}

	if err := persistCheckoutState(
		ctx,
		submissionID,
		checkoutSessionID,
		pi.ID,
		true,
		true,
		true,
	); err != nil {
		logPersistenceFailure(
			"persist successful participant payment",
			submissionID,
			err,
		)
		return
	}

	if submission.Status != "approved" {
		return
	}

	if submission.TicketID != "" {
		log.Printf(
			"Ticket already exists for submission %s: %s",
			submissionID,
			submission.TicketID,
		)
		return
	}

	event, err := findEventByID(ctx, submission.EventID)
	if err != nil {
		log.Printf(
			"Failed to load event for submission %s: %v",
			submissionID,
			err,
		)
		return
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticketToken, err := insertParticipantTicketTx(
			ctx,
			tx,
			submission,
			event.Name,
			checkoutSessionID,
			pi.ID,
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
				payment_intent_id = ?,
				payment_succeeded_before_approval = FALSE,
				payment_succeeded_at = COALESCE(payment_succeeded_at, NOW()),
				ticket_created_at = NOW()
			WHERE id = ?
		`,
			ticketToken,
			pi.ID,
			submissionID,
		).Error; err != nil {
			return fmt.Errorf(
				"update submission after ticket creation: %w",
				err,
			)
		}

		ticketMessage, err := buildParticipantTicketEmail(
			submission,
			ticketToken,
			event.Name,
		)
		if err != nil {
			return err
		}

		if err := services.QueueEmailTx(
			ctx,
			tx,
			submissionID,
			ticketMessage,
		); err != nil {
			return fmt.Errorf(
				"queue participant ticket email: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to create ticket and queue email for submission %s: %v",
			submissionID,
			err,
		)
	}
}

// handleParticipantCheckout handles checkout completion for approved vehicle submissions
func handleParticipantCheckout(checkoutSession stripe.CheckoutSession, submissionID string) {
	fmt.Printf("Handling participant checkout for submission: %s\n", submissionID)

	db := services.GetDB()
	ctx := context.Background()

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		log.Printf("Error retrieving submission data: %v", err)
		return
	}

	var paymentIntentID string
	if checkoutSession.PaymentIntent != nil {
		paymentIntentID = checkoutSession.PaymentIntent.ID
	}

	err = db.WithContext(ctx).Exec(`
		UPDATE vehicle_submissions
		SET checkout_session_id = ?,
			payment_intent_id = ?
		WHERE id = ?
	`,
		checkoutSession.ID,
		submissionNullableString(paymentIntentID),
		submissionID,
	).Error
	if err != nil {
		log.Printf("Error updating submission payment details: %v", err)
	}

	if submission.TicketID != "" {
		log.Printf(
			"Ticket already exists for submission %s: %s",
			submissionID,
			submission.TicketID,
		)
		return
	}

	eventName := checkoutSession.Metadata["event_name"]
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticketToken, err := insertParticipantTicketTx(
			ctx,
			tx,
			submission,
			eventName,
			checkoutSession.ID,
			paymentIntentID,
			"Participant",
			1,
		)
		if err != nil {
			return err
		}

		if err := tx.Exec(`
			UPDATE vehicle_submissions
			SET ticket_id = ?,
				ticket_created_at = NOW()
			WHERE id = ?
		`,
			ticketToken,
			submissionID,
		).Error; err != nil {
			return fmt.Errorf(
				"update submission with ticket ID: %w",
				err,
			)
		}

		ticketMessage, err := buildParticipantTicketEmail(
			submission,
			ticketToken,
			eventName,
		)
		if err != nil {
			return err
		}

		if err := services.QueueEmailTx(
			ctx,
			tx,
			submissionID,
			ticketMessage,
		); err != nil {
			return fmt.Errorf(
				"enqueue participant ticket email: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to create ticket and queue ticket email for submission %s: %v",
			submissionID,
			err,
		)
		return
	}

	log.Printf(
		"Created ticket and queued ticket email for submission %s",
		submissionID,
	)
}

func handlePaymentIntentFailed(pi stripe.PaymentIntent) {
	// Handle failed payment
	log.Printf("PaymentIntent failed: %s, Reason: %s\n", pi.ID, pi.LastPaymentError.Msg)

	// Get customer details from metadata or receipt email
	var customerEmail string
	if pi.ReceiptEmail != "" {
		customerEmail = pi.ReceiptEmail
	} else if pi.Customer != nil {
		customerEmail = pi.Customer.Email
	}

	// Don't send email if we don't have an address
	if customerEmail == "" {
		log.Printf("No email address available for failed payment %s", pi.ID)
		return
	}

	// Format amount for display
	amount := float64(pi.Amount) / 100.0 // Convert from cents
	currency := string(pi.Currency)
	formattedAmount := fmt.Sprintf("%.2f %s", amount, currency)

	// Get error reason
	errorMessage := "Your payment could not be processed."
	if pi.LastPaymentError != nil && pi.LastPaymentError.Msg != "" {
		errorMessage = pi.LastPaymentError.Msg
	}

	// Send payment failure notification
	emailData := map[string]interface{}{
		"PaymentAmount": formattedAmount,
		"PaymentID":     pi.ID,
		"ErrorMessage":  errorMessage,
		"PaymentDate":   time.Now().Format(time.RFC1123),
		"PaymentStatus": "Failed",
		"RecoveryURL":   os.Getenv("BASE_URL") + "/checkout/recover?id=" + pi.ID,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Payment Failed - Euro Haus",
		TemplateID:   "payment-failed",
		TemplateData: emailData,
		// Fallback if template not found
		BodyHTML: generatePaymentFailedHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue payment failure email for %s: %v",
			customerEmail,
			err,
		)
	}
}

func handleCheckoutSessionExpired(checkoutSession stripe.CheckoutSession) {
	fmt.Printf("Checkout session expired: %s\n", checkoutSession.ID)

	ctx := context.Background()
	processedKey := fmt.Sprintf("processed:expired:%s", checkoutSession.ID)

	inserted, err := markWebhookKeyProcessed(ctx, processedKey, 7*24*time.Hour)
	if err != nil {
		log.Printf("Failed to mark expired session as processed: %v", err)
		return
	}
	if !inserted {
		log.Printf("Already processed expired session %s, skipping", checkoutSession.ID)
		return
	}

	submissionID := ""
	isParticipant := false

	if checkoutSession.Metadata != nil {
		if sid, ok := checkoutSession.Metadata["submission_id"]; ok && sid != "" {
			submissionID = sid
		}
		if participant, ok := checkoutSession.Metadata["participant"]; ok && participant == "true" {
			isParticipant = true
		}
	}

	customerEmail := checkoutSession.CustomerEmail
	if customerEmail == "" && checkoutSession.CustomerDetails != nil {
		customerEmail = checkoutSession.CustomerDetails.Email
	}

	if customerEmail == "" {
		log.Printf("No email address available for expired checkout session %s", checkoutSession.ID)
		return
	}

	if submissionID != "" && isParticipant {
		handleExpiredParticipantCheckout(checkoutSession, submissionID, customerEmail)
	} else {
		handleRegularAbandonedCart(checkoutSession, customerEmail)
	}
}

func handleExpiredParticipantCheckout(
	expiredSession stripe.CheckoutSession,
	submissionID string,
	customerEmail string,
) {
	log.Printf(
		"Handling expired participant checkout for submission %s",
		submissionID,
	)

	ctx := context.Background()

	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		log.Printf(
			"Failed to load submission %s after checkout expiration: %v",
			submissionID,
			err,
		)
		return
	}

	// A ticket means the payment flow already completed successfully.
	if submission.TicketID != "" || submission.PaymentCaptured {
		log.Printf(
			"Submission %s already has a completed payment/ticket; skipping recovery",
			submissionID,
		)
		return
	}

	// If there is a PaymentIntent, avoid creating a second payment session
	// when the original payment actually succeeded.
	if submission.PaymentIntentID != "" {
		pi, paymentErr := paymentintent.Get(
			submission.PaymentIntentID,
			nil,
		)

		if paymentErr != nil {
			log.Printf(
				"Unable to retrieve PaymentIntent %s for submission %s: %v",
				submission.PaymentIntentID,
				submissionID,
				paymentErr,
			)
		} else {
			switch pi.Status {
			case stripe.PaymentIntentStatusSucceeded,
				stripe.PaymentIntentStatusRequiresCapture,
				stripe.PaymentIntentStatusProcessing:
				log.Printf(
					"Submission %s has PaymentIntent %s in status %s; skipping replacement checkout",
					submissionID,
					pi.ID,
					pi.Status,
				)
				return
			}
		}
	}

	eligible, reason := isSubmissionEligibleForRecovery(submission)
	if !eligible {
		log.Printf(
			"Submission %s is not eligible for recovery: %s",
			submissionID,
			reason,
		)

		if reason == "Maximum recovery attempts reached" {
			if err := services.GetDB().
				WithContext(ctx).
				Exec(`
					UPDATE vehicle_submissions
					SET
						status = 'abandoned',
						abandoned_at = NOW(),
						abandoned_reason = ?
					WHERE id = ?
						AND status NOT IN (
							'cancelled',
							'rejected',
							'denied',
							'abandoned',
							'revoked'
						)
				`, reason, submissionID).
				Error; err != nil {
				log.Printf(
					"Failed to mark submission %s abandoned: %v",
					submissionID,
					err,
				)
			} else {
				sendFinalAbandonmentEmail(customerEmail, submission)
			}
		}

		return
	}

	attempt, claimed, err := claimSubmissionRecoveryAttempt(
		ctx,
		submissionID,
	)
	if err != nil {
		log.Printf(
			"Failed to claim recovery attempt for submission %s: %v",
			submissionID,
			err,
		)
		return
	}

	if !claimed {
		log.Printf(
			"Submission %s has reached the recovery attempt limit",
			submissionID,
		)
		return
	}

	recoverySession, err := createSubmissionRecoverySession(
		ctx,
		submission,
	)
	if err != nil {
		log.Printf(
			"Failed to create recovery checkout for submission %s: %v",
			submissionID,
			err,
		)
		return
	}

	if customerEmail == "" {
		customerEmail = submission.ParticipantEmail
	}

	if customerEmail == "" {
		log.Printf(
			"No email address available for submission %s recovery",
			submissionID,
		)
		return
	}

	sendManualRecoveryEmail(
		customerEmail,
		submission,
		submissionID,
		attempt,
		recoverySession.URL,
	)

	log.Printf(
		"Created and emailed recovery checkout for submission %s, attempt %d of %d",
		submissionID,
		attempt,
		MaxRecoveryAttempts,
	)
}

// handleChargeRefunded processes refund events and invalidates associated tickets
func handleChargeRefunded(charge stripe.Charge) {
	fmt.Printf("Processing refund for charge: %s", charge.ID)

	paymentIntentID := charge.PaymentIntent.ID
	if paymentIntentID == "" {
		log.Printf("No payment intent associated with charge %s", charge.ID)
		return
	}

	db := services.GetDB()
	ctx := context.Background()

	rows, err := db.WithContext(ctx).Raw(`
		SELECT token, COALESCE(customer_email, ''), COALESCE(customer_name, ''), COALESCE(event_name, '')
		FROM tickets
		WHERE stripe_payment_intent_id = ?
	`, paymentIntentID).Rows()
	if err != nil {
		log.Printf("Error finding tickets for payment intent %s: %v", paymentIntentID, err)
		return
	}
	defer rows.Close()

	ticketTokens := []string{}
	var customerEmail, customerName, eventName string

	for rows.Next() {
		var token, email, name, event string
		if err := rows.Scan(&token, &email, &name, &event); err != nil {
			continue
		}

		ticketTokens = append(ticketTokens, token)
		customerEmail = email
		customerName = name
		eventName = event

		_ = db.WithContext(ctx).Exec(`
			UPDATE tickets
			SET status = 'refunded'
			WHERE token = ?
		`, token).Error

		if err := InvalidateTicket(token, "Payment refunded"); err != nil {
			log.Printf("Error invalidating ticket %s: %v", token, err)
		}
	}

	if len(ticketTokens) > 0 && customerEmail != "" {
		sendRefundNotificationEmail(customerEmail, customerName, eventName, ticketTokens, charge.AmountRefunded)
		log.Printf("Invalidated %d tickets for refunded payment %s", len(ticketTokens), paymentIntentID)
	}
}

// handlePaymentIntentCanceled processes canceled payment intents
func handlePaymentIntentCanceled(pi stripe.PaymentIntent) {
	log.Printf("Processing canceled payment intent: %s", pi.ID)

	db := services.GetDB()
	ctx := context.Background()

	submission, err := findSubmissionByPaymentIntentID(ctx, pi.ID)

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf(
				"Unable to resolve canceled payment intent %s: %v",
				pi.ID,
				err,
			)
		}

		return
	}

	err = db.WithContext(ctx).Exec(`
		UPDATE vehicle_submissions
		SET payment_status = 'canceled'
		WHERE id = ?
	`, submission.ID).Error

	if err != nil {
		log.Printf(
			"Error updating submission %s: %v",
			submission.ID,
			err,
		)
	}

	rows, err := db.WithContext(ctx).Raw(`
		SELECT token
		FROM tickets
		WHERE stripe_payment_intent_id = ?
		  AND invalidated = FALSE
	`, pi.ID).Rows()
	if err != nil {
		log.Printf("Failed to find tickets for canceled payment intent %s: %v", pi.ID, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			continue
		}

		err := db.WithContext(ctx).Exec(`
			UPDATE tickets
			SET status = 'cancelled'
			WHERE token = ?
		`, token).Error
		if err != nil {
			log.Printf("Error updating ticket status for %s: %v", token, err)
		}

		if err := InvalidateTicket(token, "Payment canceled"); err != nil {
			log.Printf("Error invalidating ticket %s: %v", token, err)
		}
	}
}

// handleCheckoutSessionPaymentFailed handles failed async payments
func handleCheckoutSessionPaymentFailed(session stripe.CheckoutSession) {
	log.Printf("Processing failed payment for checkout session: %s", session.ID)

	// Similar to refund, find and invalidate tickets
	if session.PaymentIntent != nil {
		pi, err := paymentintent.Get(session.PaymentIntent.ID, nil)
		if err == nil {
			handlePaymentIntentCanceled(*pi)
		}
	}
}

// sendRefundNotificationEmail sends an email to notify customer about refunded tickets
func sendRefundNotificationEmail(customerEmail, customerName, eventName string, ticketTokens []string, amountRefunded int64) {
	if customerName == "" {
		customerName = "Valued Customer"
	}

	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	// Format refund amount
	refundAmount := fmt.Sprintf("$%.2f", float64(amountRefunded)/100.0)

	emailData := map[string]interface{}{
		"CustomerName": customerName,
		"EventName":    eventName,
		"RefundAmount": refundAmount,
		"TicketCodes":  ticketTokens,
		"TicketCount":  len(ticketTokens),
		"RefundDate":   time.Now().Format("January 2, 2006"),
		"SupportEmail": "info@theeurohaus.com",
	}

	// Generate email HTML
	html := generateRefundNotificationHTML(emailData)

	msg := &services.EmailMessage{
		To:         []string{customerEmail},
		Subject:    fmt.Sprintf("Refund Processed - %s", eventName),
		BodyHTML:   html,
		TemplateID: "ticket-refunded",
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue refund notification email to %s: %v",
			customerEmail,
			err,
		)
		return
	}

	log.Printf(
		"Queued refund notification to %s for %d tickets",
		customerEmail,
		len(ticketTokens),
	)
}

// generateRefundNotificationHTML generates the HTML for refund notification emails
func generateRefundNotificationHTML(data map[string]interface{}) string {
	customerName := data["CustomerName"].(string)
	eventName := data["EventName"].(string)
	refundAmount := data["RefundAmount"].(string)
	ticketCodes := data["TicketCodes"].([]string)
	ticketCount := data["TicketCount"].(int)
	refundDate := data["RefundDate"].(string)
	supportEmail := data["SupportEmail"].(string)

	ticketsList := ""
	for _, code := range ticketCodes {
		ticketsList += fmt.Sprintf("<li style=\"font-family: monospace; margin: 5px 0;\">%s</li>", code)
	}

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Refund Notification - %s</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
				<div style="background-color: #dc3545; color: white; padding: 20px; text-align: center; border-radius: 10px 10px 0 0;">
					<h1 style="margin: 0;">Refund Processed</h1>
				</div>

				<div style="padding: 30px 20px; background-color: #f8f9fa;">
					<p>Dear %s,</p>

					<p>We're writing to confirm that your refund has been successfully processed.</p>

					<div style="background-color: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #dc3545;">
						<h3 style="margin-top: 0; color: #dc3545;">Refund Details</h3>
						<table style="width: 100%%;">
							<tr>
								<td style="padding: 5px 0;"><strong>Event:</strong></td>
								<td>%s</td>
							</tr>
							<tr>
								<td style="padding: 5px 0;"><strong>Refund Amount:</strong></td>
								<td>%s</td>
							</tr>
							<tr>
								<td style="padding: 5px 0;"><strong>Date Processed:</strong></td>
								<td>%s</td>
							</tr>
							<tr>
								<td style="padding: 5px 0;"><strong>Tickets Cancelled:</strong></td>
								<td>%d</td>
							</tr>
						</table>
					</div>

					<div style="background-color: #fff3cd; padding: 15px; border-radius: 8px; margin: 20px 0; border: 1px solid #ffc107;">
						<h4 style="margin-top: 0; color: #856404;">Cancelled Ticket Codes:</h4>
						<ul style="margin: 10px 0; padding-left: 20px;">
							%s
						</ul>
						<p style="margin: 10px 0 0 0; font-size: 14px; color: #856404;">
							These tickets are no longer valid and cannot be used for event entry.
						</p>
					</div>

					<p><strong>What happens next?</strong></p>
					<ul>
						<li>Your refund should appear in your account within 5-10 business days</li>
						<li>You'll receive a receipt from your payment provider</li>
						<li>Your tickets have been cancelled and removed from our system</li>
					</ul>

					<p>If you have any questions about this refund or would like to make a new purchase, please don't hesitate to contact us at <a href="mailto:%s">%s</a>.</p>

					<p>We hope to see you at a future Euro Haus event!</p>

					<p>Best regards,<br>
					<strong>The Euro Haus Events Team</strong></p>
				</div>

				<div style="text-align: center; font-size: 12px; color: #777; margin-top: 30px; padding: 20px;">
					<p>&copy; %d Euro Haus Events - Premium Automotive Experiences</p>
					<p>This is an automated notification regarding your refund.</p>
				</div>
			</div>
		</body>
		</html>
	`, eventName, customerName, eventName, refundAmount, refundDate, ticketCount, ticketsList, supportEmail, supportEmail, time.Now().Year())

	return html
}

func sendManualRecoveryEmail(
	customerEmail string,
	submissionData *models.VehicleSubmissionDTO,
	submissionID string,
	attemptNumber int,
	recoveryLink string,
) {
	participantName := submissionData.ParticipantName
	vehicleDetails := fmt.Sprintf(
		"%s %s %s",
		submissionData.VehicleYear,
		submissionData.VehicleMake,
		submissionData.VehicleModel,
	)

	eventName := submissionData.EventSlug
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "https://eurohaus.shop"
	}

	emailData := map[string]interface{}{
		"ParticipantName": participantName,
		"VehicleDetails":  vehicleDetails,
		"EventName":       eventName,
		"RecoveryLink":    recoveryLink,
		"SubmissionID":    submissionID,
		"AttemptNumber":   attemptNumber,
		"BaseURL":         baseURL,
	}

	subject := "Action Required: Complete Your Euro Haus Registration"
	if attemptNumber >= MaxRecoveryAttempts {
		subject = "Final Reminder: Complete Your Euro Haus Registration"
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      subject,
		TemplateID:   "participant-manual-recovery",
		TemplateData: emailData,
		BodyHTML:     generateManualRecoveryHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		submissionID,
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue recovery email for submission %s: %v",
			submissionID,
			err,
		)
	}
}

// sendFinalAbandonmentEmail sends a final email when max recovery attempts are reached
func sendFinalAbandonmentEmail(customerEmail string, submissionData *models.VehicleSubmissionDTO) {
	participantName := submissionData.ParticipantName
	vehicleDetails := fmt.Sprintf("%s %s %s",
		submissionData.VehicleYear,
		submissionData.VehicleMake,
		submissionData.VehicleModel)

	baseURL := os.Getenv("BASE_URL")
	resubmitLink := fmt.Sprintf("%s/events/submit", baseURL)

	emailData := map[string]interface{}{
		"ParticipantName": participantName,
		"VehicleDetails":  vehicleDetails,
		"ResubmitLink":    resubmitLink,
		"ContactEmail":    "info@theeurohaus.com",
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Registration Expired - Euro Haus",
		TemplateID:   "submission-abandoned",
		TemplateData: emailData,
		BodyHTML:     generateAbandonmentHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		submissionData.ID,
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue abandonment email for submission %s: %v",
			submissionData.ID,
			err,
		)
	}
}

// handleRegularAbandonedCart handles regular abandoned cart (non-participant)
func handleRegularAbandonedCart(checkoutSession stripe.CheckoutSession, customerEmail string) {
	// Send standard abandoned cart email
	emailData := map[string]interface{}{
		"SessionID":      checkoutSession.ID,
		"ExpirationTime": time.Now().Format(time.RFC1123),
		"RecoveryURL":    os.Getenv("BASE_URL") + "/checkout/recover?id=" + checkoutSession.ID,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Complete Your Euro Haus Purchase",
		TemplateID:   "checkout-abandoned",
		TemplateData: emailData,
		BodyHTML:     generateAbandonedCartHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		checkoutSession.ID,
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue abandoned cart email for session %s: %v",
			checkoutSession.ID,
			err,
		)
	}
}

// formatShippingAddress formats the shipping address from checkout session
func formatShippingAddress(session *stripe.CheckoutSession) string {
	// Use ShippingDetails if available, otherwise check CustomerDetails
	var address *stripe.Address

	if session.Customer.Shipping != nil && session.Customer.Shipping.Address != nil {
		address = session.Customer.Shipping.Address
	} else if session.CustomerDetails != nil && session.CustomerDetails.Address != nil {
		address = session.CustomerDetails.Address
	} else {
		return "No shipping address provided"
	}

	formatted := ""
	if address.Line1 != "" {
		formatted += address.Line1 + "\n"
	}
	if address.Line2 != "" {
		formatted += address.Line2 + "\n"
	}
	if address.City != "" {
		formatted += address.City
	}
	if address.State != "" {
		if formatted != "" && formatted[len(formatted)-1] != '\n' {
			formatted += ", "
		}
		formatted += address.State
	}
	if address.PostalCode != "" {
		formatted += " " + address.PostalCode + "\n"
	} else if formatted != "" && formatted[len(formatted)-1] != '\n' {
		formatted += "\n"
	}
	if address.Country != "" {
		formatted += address.Country
	}

	return formatted
}

// Generate HTML fallback templates for when email templates aren't available
func generateOrderConfirmationHTML(data map[string]interface{}) string {
	customerName, _ := data["CustomerName"].(string)
	orderId, _ := data["OrderID"].(string)
	orderDate, _ := data["OrderDate"].(string)
	totalAmount, _ := data["TotalAmount"].(float64)
	currency, _ := data["Currency"].(string)
	products, _ := data["Products"].([]map[string]interface{})
	hasPhysicalItems, _ := data["HasPhysicalItems"].(bool)
	shippingAddress, _ := data["ShippingAddress"].(string)

	// Build product items HTML
	productsHTML := ""
	for _, product := range products {
		name, _ := product["Name"].(string)
		quantity, _ := product["Quantity"].(int64)
		price, _ := product["Price"].(float64)
		subtotal, _ := product["Subtotal"].(float64)

		productsHTML += fmt.Sprintf(`
			<tr>
				<td style="padding: 10px; border-bottom: 1px solid #eee;">%s</td>
				<td style="padding: 10px; border-bottom: 1px solid #eee;">%d</td>
				<td style="padding: 10px; border-bottom: 1px solid #eee;">%.2f %s</td>
				<td style="padding: 10px; border-bottom: 1px solid #eee;">%.2f %s</td>
			</tr>
		`, name, quantity, price, currency, subtotal, currency)
	}

	// Build shipping section if needed
	shippingHTML := ""
	if hasPhysicalItems {
		shippingHTML = fmt.Sprintf(`
			<div style="margin-top: 20px; padding: 20px; border: 1px solid #eee; border-radius: 5px;">
				<h3 style="margin-top: 0;">Shipping Information</h3>
				<p style="white-space: pre-line;">%s</p>
				<p>We'll send you another email when your order ships.</p>
			</div>
		`, shippingAddress)
	}

	// Complete HTML email
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Order Confirmation - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Thank You For Your Order</h1>
			</div>

			<p>Hello %s,</p>
			<p>Thank you for your purchase! We've received your order and are processing it now.</p>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Order Summary</h2>
				<p><strong>Order ID:</strong> %s</p>
				<p><strong>Date:</strong> %s</p>

				<table style="width: 100%%; border-collapse: collapse;">
					<thead>
						<tr>
							<th style="text-align: left; padding: 10px; border-bottom: 2px solid #eee;">Item</th>
							<th style="text-align: left; padding: 10px; border-bottom: 2px solid #eee;">Quantity</th>
							<th style="text-align: left; padding: 10px; border-bottom: 2px solid #eee;">Price</th>
							<th style="text-align: left; padding: 10px; border-bottom: 2px solid #eee;">Subtotal</th>
						</tr>
					</thead>
					<tbody>
						%s
					</tbody>
					<tfoot>
						<tr>
							<td colspan="3" style="text-align: right; padding: 10px;"><strong>Total:</strong></td>
							<td style="padding: 10px;"><strong>%.2f %s</strong></td>
						</tr>
					</tfoot>
				</table>
			</div>

			%s

			<div style="margin-top: 30px;">
				<p>If you have any questions about your order, please contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a>.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, customerName, orderId, orderDate, productsHTML, totalAmount, currency, shippingHTML, time.Now().Year())

	return html
}

func generatePaymentSuccessHTML(data map[string]interface{}) string {
	customerName, _ := data["CustomerName"].(string)
	paymentAmount, _ := data["PaymentAmount"].(string)
	paymentId, _ := data["PaymentID"].(string)
	paymentDate, _ := data["PaymentDate"].(string)

	// Complete HTML email
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Payment Confirmation - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Payment Confirmation</h1>
			</div>

			<p>Hello %s,</p>
			<p>Thank you for your payment! We've successfully processed your payment of %s.</p>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Payment Details</h2>
				<p><strong>Payment ID:</strong> %s</p>
				<p><strong>Date:</strong> %s</p>
				<p><strong>Amount:</strong> %s</p>
				<p><strong>Status:</strong> Successful</p>
			</div>

			<div style="margin-top: 30px;">
				<p>If you have any questions about your payment, please contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a>.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, customerName, paymentAmount, paymentId, paymentDate, paymentAmount, time.Now().Year())

	return html
}

func generatePaymentFailedHTML(data map[string]interface{}) string {
	paymentAmount, _ := data["PaymentAmount"].(string)
	paymentId, _ := data["PaymentID"].(string)
	paymentDate, _ := data["PaymentDate"].(string)
	errorMessage, _ := data["ErrorMessage"].(string)
	recoveryURL, _ := data["RecoveryURL"].(string)

	// Complete HTML email
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Payment Failed - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Payment Failed</h1>
			</div>

			<p>We're sorry, but your payment could not be processed.</p>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Payment Details</h2>
				<p><strong>Payment ID:</strong> %s</p>
				<p><strong>Date:</strong> %s</p>
				<p><strong>Amount:</strong> %s</p>
				<p><strong>Status:</strong> Failed</p>
				<p><strong>Reason:</strong> %s</p>
			</div>

			<div style="margin-top: 20px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #4CAF50; color: white; padding: 12px 20px; text-decoration: none; border-radius: 4px;">Try Payment Again</a>
			</div>

			<div style="margin-top: 30px;">
				<p>If you continue to experience issues with your payment, please contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a>.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, paymentId, paymentDate, paymentAmount, errorMessage, recoveryURL, time.Now().Year())

	return html
}

func generateAbandonedCartHTML(data map[string]interface{}) string {
	sessionId, _ := data["SessionID"].(string)
	expirationTime, _ := data["ExpirationTime"].(string)
	recoveryURL, _ := data["RecoveryURL"].(string)

	// Complete HTML email
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Complete Your Purchase - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Don't Miss Out!</h1>
			</div>

			<p>We noticed that you started checkout but didn't complete your purchase.</p>
			<p>Your cart items are still available, but they may sell out soon!</p>

			<div style="margin-top: 20px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #4CAF50; color: white; padding: 12px 20px; text-decoration: none; border-radius: 4px;">Complete Your Purchase</a>
			</div>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<p><strong>Session ID:</strong> %s</p>
				<p><strong>Expiration:</strong> %s</p>
			</div>

			<div style="margin-top: 30px;">
				<p>If you have any questions or need assistance with your purchase, please contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a>.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
				<p>If you believe you received this email by mistake, please disregard it.</p>
			</div>
		</body>
		</html>
	`, recoveryURL, sessionId, expirationTime, time.Now().Year())

	return html
}

// generateParticipantRecoveryHTML generates the HTML for participant checkout recovery emails
func generateParticipantRecoveryHTML(data map[string]interface{}) string {
	participantName, _ := data["ParticipantName"].(string)
	vehicleDetails, _ := data["VehicleDetails"].(string)
	eventName, _ := data["EventName"].(string)
	paymentLink, _ := data["PaymentLink"].(string)
	expirationTime, _ := data["ExpirationTime"].(string)
	submissionID, _ := data["SubmissionID"].(string)

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Complete Your Registration - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1 style="color: #e74c3c;">⚠️ Your Payment Session Expired</h1>
			</div>

			<p>Hello %s,</p>

			<p>Your payment session for the vehicle registration has expired. Don't worry - your submission is still pending approval, and we've created a new payment link for you.</p>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Registration Details</h2>
				<p><strong>Event:</strong> %s</p>
				<p><strong>Vehicle:</strong> %s</p>
				<p><strong>Submission ID:</strong> %s</p>
			</div>

			<div style="background-color: #fff3cd; padding: 20px; border-radius: 5px; margin: 20px 0; border: 1px solid #ffc107;">
				<h3 style="margin-top: 0; color: #856404;">Important Information</h3>
				<ul style="margin: 10px 0;">
					<li>Your vehicle submission is still pending admin approval</li>
					<li>Payment will only be captured after your submission is approved</li>
					<li>This new payment link will expire in 24 hours (%s)</li>
					<li>You will receive a confirmation email once approved</li>
				</ul>
			</div>

			<div style="margin-top: 30px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #4CAF50; color: white; padding: 15px 30px; text-decoration: none; border-radius: 4px; font-size: 16px; font-weight: bold;">Complete Registration</a>
			</div>

			<div style="margin-top: 30px;">
				<h3>What happens next?</h3>
				<ol>
					<li>Click the button above to complete your payment information</li>
					<li>Your card will be authorized but not charged immediately</li>
					<li>Our team will review your submission</li>
					<li>Once approved, your payment will be processed and you'll receive your event ticket</li>
				</ol>
			</div>

			<div style="margin-top: 30px;">
				<p>If you have any questions or continue to experience issues, please contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a> with your submission ID: %s</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, participantName, eventName, vehicleDetails, submissionID, expirationTime, paymentLink, submissionID, time.Now().Year())

	return html
}

func updateEventInventory(eventID string, quantity int64) error {
	if quantity <= 0 {
		return nil
	}

	var availableSpots int

	err := services.GetDB().Raw(`
		UPDATE events
		SET
			available_spots = available_spots - ?,
			status = CASE
				WHEN available_spots - ? <= 0 THEN 'sold_out'
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = ?
		  AND active = TRUE
		  AND available_spots >= ?
		RETURNING available_spots
	`, quantity, quantity, eventID, quantity).
		Row().
		Scan(&availableSpots)

	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("insufficient event inventory")
	}

	if err != nil {
		return err
	}

	BroadcastEventUpdate(eventID, map[string]interface{}{
		"action":         "inventory_updated",
		"eventId":        eventID,
		"availableSpots": availableSpots,
	})

	return nil
}

func updateProductVariantStock(
	priceID string,
	quantitySold int64,
) {
	if quantitySold <= 0 {
		return
	}

	db := services.GetDB().WithContext(context.Background())

	var remainingStock int

	err := db.Raw(`
		UPDATE prices
		SET
			stock_quantity = stock_quantity - ?,
			active = CASE
				WHEN stock_quantity - ? <= 0 THEN FALSE
				ELSE active
			END,
			updated_at = NOW()
		WHERE id = ?
		  AND stock_quantity IS NOT NULL
		  AND stock_quantity >= ?
		RETURNING stock_quantity
	`,
		quantitySold,
		quantitySold,
		priceID,
		quantitySold,
	).
		Row().
		Scan(&remainingStock)

	if errors.Is(err, sql.ErrNoRows) {
		// Either unlimited inventory or insufficient inventory.
		return
	}

	if err != nil {
		log.Printf(
			"Failed to decrement stock for price %s: %v",
			priceID,
			err,
		)
		return
	}

	log.Printf(
		"Price %s has %d units remaining",
		priceID,
		remainingStock,
	)
}

// sendLowStockAlert sends an email notification when stock is low
func sendLowStockAlert(p *stripe.Price, remainingStock int) {
	// Get product details for the alert
	prod, err := product.Get(p.Product.ID, nil)
	if err != nil {
		log.Printf("Error fetching product for low stock alert: %v\n", err)
		return
	}

	variantName := p.Nickname
	if variantName == "" {
		variantName = p.Metadata["variant"]
	}

	// Send alert email to admin
	adminEmail := os.Getenv("ADMIN_NOTIFICATION_EMAIL")
	if adminEmail == "" {
		adminEmail = "info@eurohaus.com" // Fallback
	}

	emailData := map[string]interface{}{
		"ProductName":    prod.Name,
		"VariantName":    variantName,
		"Size":           p.Metadata["size"],
		"Color":          p.Metadata["color"],
		"RemainingStock": remainingStock,
		"PriceID":        p.ID,
		"ProductID":      prod.ID,
	}

	msg := &services.EmailMessage{
		To:       []string{adminEmail},
		Subject:  fmt.Sprintf("Low Stock Alert: %s - %s", prod.Name, variantName),
		BodyHTML: generateLowStockAlertHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue low stock alert for product %s: %v",
			prod.ID,
			err,
		)
	}
}

// generateLowStockAlertHTML generates the HTML for low stock alert emails
func generateLowStockAlertHTML(data map[string]interface{}) string {
	productName := data["ProductName"].(string)
	variantName := data["VariantName"].(string)
	remainingStock := data["RemainingStock"].(int)

	sizeInfo := ""
	if size, ok := data["Size"].(string); ok && size != "" {
		sizeInfo = fmt.Sprintf(" (Size: %s)", size)
	}

	colorInfo := ""
	if color, ok := data["Color"].(string); ok && color != "" {
		colorInfo = fmt.Sprintf(" (Color: %s)", color)
	}

	return fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.header { background: #ff6b6b; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
			.content { background: #f9f9f9; padding: 20px; border: 1px solid #ddd; border-radius: 0 0 5px 5px; }
			.alert { background: #fff3cd; border: 1px solid #ffc107; padding: 15px; border-radius: 5px; margin: 20px 0; }
			.details { background: white; padding: 15px; border-radius: 5px; margin: 20px 0; }
			.action-btn { background: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block; margin-top: 10px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>⚠️ Low Stock Alert</h1>
			</div>
			<div class="content">
				<div class="alert">
					<h2>Immediate Attention Required</h2>
					<p>The following product variant is running low on stock:</p>
				</div>

				<div class="details">
					<h3>Product Details:</h3>
					<ul>
						<li><strong>Product:</strong> %s</li>
						<li><strong>Variant:</strong> %s%s%s</li>
						<li><strong>Remaining Stock:</strong> <span style="color: #ff6b6b; font-weight: bold;">%d units</span></li>
					</ul>
				</div>

				<p>Please restock this item as soon as possible to avoid stockouts.</p>

				<a href="%s" class="action-btn">View in Stripe Dashboard</a>
			</div>
		</div>
	</body>
	</html>
	`, productName, variantName, sizeInfo, colorInfo, remainingStock,
		fmt.Sprintf("https://dashboard.stripe.com/products/%s", data["ProductID"]))
}

// processStockUpdates handles stock updates for all items in a checkout session
func processStockUpdates(checkoutSession *stripe.CheckoutSession) {
	// Process each line item
	for _, lineItem := range checkoutSession.LineItems.Data {
		// Skip if no price ID
		if lineItem.Price == nil {
			continue
		}

		priceID := lineItem.Price.ID
		quantitySold := lineItem.Quantity

		// Get the product type from metadata
		if lineItem.Price.Product != nil {
			var localProduct models.Product

			productErr := services.GetDB().
				WithContext(context.Background()).
				Where("id = ?", lineItem.Price.Product.ID).
				First(&localProduct).
				Error

			if errors.Is(productErr, gorm.ErrRecordNotFound) {
				log.Printf(
					"Product %s has no local catalog record",
					lineItem.Price.Product.ID,
				)
				continue
			}

			if productErr != nil {
				log.Printf(
					"Failed to load local product %s: %v",
					lineItem.Price.Product.ID,
					productErr,
				)
				continue
			}

			productType := localProduct.Type

			switch productType {
			case "event":
				event, err := findEventByStripeProductID(
					context.Background(),
					lineItem.Price.Product.ID,
				)

				if errors.Is(err, gorm.ErrRecordNotFound) {
					// This was not an event purchase.
					return
				}

				if err != nil {
					log.Printf("Unable to resolve event: %v", err)
					return
				}

				if err := updateEventInventory(
					event.ID,
					lineItem.Quantity,
				); err != nil {
					log.Printf("Unable to update event inventory: %v", err)
				}

			case "product":
				// Update product variant stock
				updateProductVariantStock(priceID, quantitySold)

			case "bundle":
				// For bundles, we need to update stock for each bundled item
				// This is more complex as we need to parse the bundle metadata
				handleBundleStockUpdate(lineItem.Price.Product.ID, quantitySold)

			default:
				// Regular product without specific type - still try to update stock
				updateProductVariantStock(priceID, quantitySold)
			}
		} else {
			// No product info, just try to update as variant stock
			updateProductVariantStock(priceID, quantitySold)
		}
	}
}

// handleBundleStockUpdate processes stock updates for bundle products
func handleBundleStockUpdate(bundleProductID string, quantitySold int64) {
	// Get the bundle product
	_, err := product.Get(bundleProductID, nil)
	if err != nil {
		log.Printf("Error fetching bundle product %s: %v\n", bundleProductID, err)
		return
	}

	var bundleItems []models.BundleItem

	err = services.GetDB().
		WithContext(context.Background()).
		Where("bundle_product_id = ?", bundleProductID).
		Order("sort_order ASC").
		Find(&bundleItems).
		Error

	if err != nil {
		log.Printf(
			"Unable to load bundle items for %s: %v",
			bundleProductID,
			err,
		)
		return
	}

	// For each item in the bundle, update its stock
	for _, item := range bundleItems {
		// Get the product's default price or prices
		bundledProduct, err := product.Get(item.ProductID, nil)
		if err != nil {
			log.Printf("Error fetching bundled product %s: %v\n", item.ProductID, err)
			continue
		}

		totalQuantity := int64(item.Quantity) * quantitySold

		// If product has variants, we need to handle this differently
		// For simplicity, we'll update the default price if it exists
		if bundledProduct.DefaultPrice != nil {
			updateProductVariantStock(bundledProduct.DefaultPrice.ID, totalQuantity)
		} else {
			log.Printf("Bundle item %s has no default price to update stock\n", item.ProductID)
		}
	}

	log.Printf("Processed stock updates for bundle %s\n", bundleProductID)
}

// generateUniqueToken creates a unique token for a ticket
func generateUniqueToken() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// generateQRCode creates a QR code image for a ticket token
func generateQRCode(token string) (string, error) {
	qr, err := qrcode.Encode(token, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	// Base64 encode the QR code for embedding in email
	return base64.StdEncoding.EncodeToString(qr), nil
}

func eventNameForStripeProduct(ctx context.Context, stripeProductID string) string {
	var event models.Event

	err := services.GetDB().
		WithContext(ctx).
		Where("stripe_product_id = ?", stripeProductID).
		First(&event).
		Error

	if err != nil {
		return "Euro Haus Event"
	}

	return event.Name
}

// storeTicketPurchase stores ticket info in Redis after purchase
func storeTicketPurchase(session stripe.CheckoutSession, lineItem stripe.LineItem) {
	if session.Metadata != nil {
		if submissionID, ok := session.Metadata["submission_id"]; ok && submissionID != "" {
			if session.Metadata["participant"] == "true" {
				submission, err := getSubmissionByID(submissionID)
				if err != nil {
					log.Printf("ERROR: Could not verify submission status for %s: %v", submissionID, err)
					return
				}

				if submission.Status != "approved" {
					log.Printf("WARNING: Attempted to create ticket for unapproved participant submission %s (status: %s). Blocking ticket creation.",
						submissionID, submission.Status)
					return
				}

				if submission.TicketID != "" {
					log.Printf("Ticket already exists for submission %s: %s", submissionID, submission.TicketID)
					return
				}

				fmt.Printf("Participant submission %s is approved, proceeding with ticket creation", submissionID)
			}
		}
	}

	token := generateUniqueToken()
	productID := ""
	if lineItem.Price != nil && lineItem.Price.Product != nil {
		productID = lineItem.Price.Product.ID
	}

	db := services.GetDB()
	ctx := context.Background()

	customerEmail := session.CustomerDetails.Email
	customerName := session.CustomerDetails.Name

	var paymentIntentID string
	if session.PaymentIntent != nil {
		paymentIntentID = session.PaymentIntent.ID
	}

	var existingToken, existingPaymentIntentID, existingCheckoutSessionID string
	err := db.WithContext(ctx).Raw(`
		SELECT token,
		       COALESCE(stripe_payment_intent_id, ''),
		       COALESCE(stripe_session_id, '')
		FROM tickets
		WHERE stripe_session_id = ? OR stripe_payment_intent_id = ?
	`, session.ID, submissionNullableString(paymentIntentID)).Row().Scan(&existingToken, &existingPaymentIntentID, &existingCheckoutSessionID)

	if err == nil && existingToken != "" {
		if existingCheckoutSessionID != session.ID {
			_ = db.WithContext(ctx).Exec(`
				UPDATE tickets
				SET stripe_session_id = ?
				WHERE token = ?
			`, session.ID, existingToken).Error
		}
		return
	}

	_ = db.WithContext(ctx).Exec(`DELETE FROM tickets WHERE token = ?`, existingToken).Error

	ticketType := "General Admission"
	var submissionID string
	if sid, ok := session.Metadata["submission_id"]; ok && sid != "" {
		submissionID = sid
		ticketType = "Participant"
	}

	eventID := lineItem.Price.Product.ID

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if lineItem.Price == nil || lineItem.Price.Product == nil {
			return fmt.Errorf(
				"ticket purchase %s has no product information",
				session.ID,
			)
		}

		qrCode, err := generateQRCode(token)
		if err != nil {
			return fmt.Errorf("generate ticket QR code: %w", err)
		}

		err = tx.Exec(`
			INSERT INTO tickets (
				token,
				event_id,
				stripe_product_id,
				stripe_session_id,
				stripe_payment_intent_id,
				customer_email,
				customer_name,
				quantity,
				ticket_type,
				status
			)
			VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, 'active'
			)
		`,
			token,
			eventID,
			productID,
			session.ID,
			submissionNullableString(paymentIntentID),
			customerEmail,
			customerName,
			lineItem.Quantity,
			ticketType,
		).Error

		if err != nil {
			return fmt.Errorf("insert ticket in Postgres: %w", err)
		}

		eventDetails := map[string]interface{}{
			"name":     lineItem.Price.Product.Name,
			"metadata": lineItem.Price.Product.Metadata,
			"quantity": lineItem.Quantity,
		}

		message, err := services.BuildTicketEmail(
			customerEmail,
			customerName,
			token,
			eventDetails,
			qrCode,
		)
		if err != nil {
			return fmt.Errorf("build ticket email: %w", err)
		}

		if err := services.QueueEmailTx(
			ctx,
			tx,
			submissionID,
			message,
		); err != nil {
			return fmt.Errorf("enqueue ticket email: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf(
			"Failed to create ticket and queue ticket email for session %s: %v",
			session.ID,
			err,
		)
		return
	}

	fmt.Printf(
		"Created ticket %s and queued ticket email for customer %s",
		token,
		customerEmail,
	)
}

func buildParticipantTicketEmail(
	submission *models.VehicleSubmissionDTO,
	ticketToken string,
	eventName string,
) (*services.EmailMessage, error) {
	if submission == nil {
		return nil, fmt.Errorf("submission is nil")
	}

	qrCodeURL, err := generateQRCode(ticketToken)
	if err != nil {
		return nil, fmt.Errorf("generate QR code: %w", err)
	}

	vehicleDetails := fmt.Sprintf(
		"%s %s %s",
		submission.VehicleYear,
		submission.VehicleMake,
		submission.VehicleModel,
	)

	emailData := map[string]interface{}{
		"CustomerName":   submission.ParticipantName,
		"EventName":      eventName,
		"TicketCode":     ticketToken,
		"QRCodeURL":      qrCodeURL,
		"VehicleDetails": vehicleDetails,
		"TicketType":     "Event Participant",
		"CheckInURL": fmt.Sprintf(
			"%s/events/checkin?ticket=%s",
			strings.TrimRight(os.Getenv("BASE_URL"), "/"),
			url.QueryEscape(ticketToken),
		),
	}

	ticketHTML := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
				<h1 style="color: #007bff;">
					Your Event Participant Ticket
				</h1>

				<p>Dear %s,</p>

				<p>
					Your vehicle registration is complete and your vehicle
					has been approved:
				</p>

				<p style="font-size: 18px; font-weight: bold;">
					%s
				</p>

				<div style="
					background-color: #f8f9fa;
					padding: 20px;
					border-radius: 10px;
					margin: 20px 0;
				">
					<h2>Event Details</h2>
					<p><strong>Event:</strong> %s</p>
					<p><strong>Ticket Type:</strong> Event Participant</p>
					<p>
						<strong>Ticket Code:</strong>
						<span style="font-family: monospace; font-size: 18px;">
							%s
						</span>
					</p>
				</div>

				<div style="text-align: center; margin: 30px 0;">
					<img
						src="%s"
						alt="QR Code"
						style="width: 200px; height: 200px;"
					>
					<p style="font-size: 12px; color: #666;">
						Show this QR code at check-in
					</p>
				</div>

				<h3>Important Information for Participants:</h3>

				<ul>
					<li>
						Please arrive at least 30 minutes before the event
						start time.
					</li>
					<li>
						Have your vehicle clean and ready for display.
					</li>
					<li>
						Bring this ticket, printed or on your phone.
					</li>
					<li>
						Follow all event guidelines and staff instructions.
					</li>
				</ul>

				<p>
					We're excited to have you showcase your vehicle at our event!
				</p>

				<p>
					Best regards,<br>
					The Euro Haus Events Team
				</p>
			</div>
		</body>
		</html>
	`,
		submission.ParticipantName,
		vehicleDetails,
		eventName,
		ticketToken,
		qrCodeURL,
	)

	return &services.EmailMessage{
		To: []string{submission.ParticipantEmail},
		Subject: fmt.Sprintf(
			"Event Participant Ticket - %s",
			eventName,
		),
		TemplateID:   "participant-ticket",
		TemplateData: emailData,
		BodyHTML:     ticketHTML,
	}, nil
}

func buildApprovalWithTicketEmail(
	submission *models.VehicleSubmissionDTO,
	ticketToken string,
) *services.EmailMessage {
	vehicleDetails := fmt.Sprintf(
		"%s %s %s",
		submission.VehicleYear,
		submission.VehicleMake,
		submission.VehicleModel,
	)

	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails":  vehicleDetails,
		"EventID":         submission.EventID,
		"TicketCode":      ticketToken,
		"ReviewNotes":     submission.ReviewNotes,
	}

	return &services.EmailMessage{
		To:           []string{submission.ParticipantEmail},
		Subject:      "Your Vehicle Submission Has Been Approved + Ticket - Euro Haus",
		TemplateID:   "submission-approved-with-ticket",
		TemplateData: emailData,
		BodyHTML:     generateApprovalWithTicketEmailHTML(emailData),
	}
}

func sendGenericRecoveryEmail(customerEmail string, submissionID string) {
	baseUrl := os.Getenv("BASE_URL")

	// Create a recovery URL that leads to a page where they can re-initiate checkout
	recoveryURL := fmt.Sprintf("%s/checkout/recover?id=%s", baseUrl, submissionID)

	emailData := map[string]interface{}{
		"SessionID":      submissionID, // Using submission ID as reference
		"ExpirationTime": "Your payment session has expired",
		"RecoveryURL":    recoveryURL,
		"SubmissionID":   submissionID,
		"IsSubmission":   true, // Flag to indicate this is for a submission recovery
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Complete Your Euro Haus Registration",
		TemplateID:   "checkout-recovery",
		TemplateData: emailData,
		// Use modified abandoned cart HTML for consistency
		BodyHTML: generateSubmissionRecoveryHTML(emailData),
	}

	if err := services.QueueEmail(
		context.Background(),
		submissionID,
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue recovery email for submission %s: %v",
			submissionID,
			err,
		)
	}
}

// generateSubmissionRecoveryHTML generates recovery email HTML similar to abandoned cart
func generateSubmissionRecoveryHTML(data map[string]interface{}) string {
	submissionID, _ := data["SubmissionID"].(string)
	recoveryURL, _ := data["RecoveryURL"].(string)

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Complete Your Registration - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Complete Your Registration!</h1>
			</div>

			<p>We noticed that your payment session expired before you could complete your vehicle registration.</p>
			<p>Don't worry - your submission is still saved and waiting for you!</p>

			<div style="margin-top: 20px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #4CAF50; color: white; padding: 12px 20px; text-decoration: none; border-radius: 4px;">Complete Your Registration</a>
			</div>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<p><strong>Submission Reference:</strong> %s</p>
				<p><strong>Status:</strong> Payment session expired - awaiting new payment</p>
			</div>

			<div style="margin-top: 30px;">
				<h3>What happens next?</h3>
				<ol style="line-height: 1.8;">
					<li>Click the button above to restart your checkout</li>
					<li>Your vehicle submission details are saved</li>
					<li>Complete the payment process</li>
					<li>Wait for admin approval of your submission</li>
					<li>Once approved, your payment will be processed</li>
				</ol>
			</div>

			<div style="margin-top: 30px;">
				<p>If you continue to experience issues, please try again later or use a different browser.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
				<p>If you believe you received this email by mistake, please disregard it.</p>
			</div>
		</body>
		</html>
	`, recoveryURL, submissionID, time.Now().Year())

	return html
}

// generateApprovalWithTicketEmailHTML generates email HTML for approved participants with ticket
func generateApprovalWithTicketEmailHTML(data map[string]interface{}) string {
	participantName, _ := data["ParticipantName"].(string)
	vehicleDetails, _ := data["VehicleDetails"].(string)
	eventID, _ := data["EventID"].(string)
	ticketCode, _ := data["TicketCode"].(string)
	reviewNotes, _ := data["ReviewNotes"].(string)

	reviewNotesHTML := ""
	if reviewNotes != "" {
		reviewNotesHTML = fmt.Sprintf(`
			<div style="background-color: #fff3cd; padding: 15px; border-radius: 5px; margin: 20px 0; border: 1px solid #ffc107;">
				<h3 style="margin-top: 0; color: #856404;">Review Notes from Our Team</h3>
				<p style="margin: 0;">%s</p>
			</div>
		`, reviewNotes)
	}

	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Vehicle Approved + Your Ticket - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1 style="color: #28a745;">✓ Vehicle Approved & Ticket Issued!</h1>
			</div>

			<p>Dear %s,</p>

			<p>Excellent news! Your vehicle submission has been approved and your payment has been processed successfully:</p>

			<div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0; color: #007bff;">Vehicle Details</h2>
				<p style="font-size: 18px; font-weight: bold; margin: 10px 0;">%s</p>
				<p><strong>Event ID:</strong> %s</p>
			</div>

			%s

			<div style="background-color: #d4edda; padding: 20px; border-radius: 5px; margin: 20px 0; border: 2px solid #28a745;">
				<h2 style="margin-top: 0; color: #155724;">Your Event Ticket</h2>
				<p style="margin: 10px 0;">Your ticket has been generated and your vehicle is confirmed for the event!</p>
				<p style="font-size: 20px; font-family: monospace; background-color: white; padding: 10px; border-radius: 5px; text-align: center;">
					<strong>Ticket Code:</strong> %s
				</p>
				<p style="font-size: 14px; color: #666; text-align: center;">
					A separate email with your QR code and full ticket details has been sent.
				</p>
			</div>

			<div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h3 style="margin-top: 0;">Important Information for Participants</h3>
				<ul style="margin: 10px 0; padding-left: 20px;">
					<li>Please arrive at least 30 minutes before the event start time</li>
					<li>Have your vehicle clean and ready for display</li>
					<li>Bring your ticket (printed or on your phone) for check-in</li>
					<li>Follow all event guidelines and instructions from staff</li>
					<li>Be prepared to position your vehicle as directed by event coordinators</li>
				</ul>
			</div>

			<div style="margin-top: 30px; padding: 20px; background-color: #e8f4f8; border-radius: 5px;">
				<h3 style="margin-top: 0;">What's Next?</h3>
				<ol style="margin: 10px 0; padding-left: 20px;">
					<li>Check your email for your event ticket with QR code</li>
					<li>Save the ticket to your phone or print it out</li>
					<li>Prepare your vehicle for the show</li>
					<li>Arrive early on event day for participant check-in</li>
				</ol>
			</div>

			<div style="margin-top: 30px;">
				<p>We're thrilled to have you and your vehicle as part of our event! If you have any questions, please don't hesitate to contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a>.</p>
			</div>

			<div style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #dee2e6; text-align: center; font-size: 12px; color: #6c757d;">
				<p><strong>Euro Haus Events</strong><br>
				Premium Automotive Experiences</p>
				<p>&copy; %d Euro Haus. All rights reserved.</p>
				<p style="margin-top: 10px;">
					<em>This email confirms your approved participation status and successful payment processing.</em>
				</p>
			</div>
		</body>
		</html>
	`, participantName, vehicleDetails, eventID, reviewNotesHTML, ticketCode, time.Now().Year())
}

// generateManualRecoveryHTML generates HTML for manual recovery emails
func generateManualRecoveryHTML(data map[string]interface{}) string {
	participantName, _ := data["ParticipantName"].(string)
	vehicleDetails, _ := data["VehicleDetails"].(string)
	eventName, _ := data["EventName"].(string)
	recoveryLink, _ := data["RecoveryLink"].(string)
	submissionID, _ := data["SubmissionID"].(string)
	attemptNumber, _ := data["AttemptNumber"].(int)
	maxAttempts, _ := data["MaxAttempts"].(int)
	isLastAttempt, _ := data["IsLastAttempt"].(bool)

	urgencyHTML := ""
	if isLastAttempt {
		urgencyHTML = `
			<div style="background-color: #f8d7da; padding: 15px; border-radius: 5px; margin: 20px 0; border: 1px solid #f5c6cb;">
				<h3 style="margin-top: 0; color: #721c24;">⚠️ Final Reminder</h3>
				<p style="margin: 0; color: #721c24;">This is your last opportunity to complete your registration.
				After this, you will need to submit a new application.</p>
			</div>
		`
	}

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Complete Your Registration - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Complete Your Vehicle Registration</h1>
			</div>

			<p>Hello %s,</p>

			<p>Your payment session for the vehicle registration has expired. We're holding your spot, but you need to complete the payment process.</p>

			%s

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Registration Details</h2>
				<p><strong>Event:</strong> %s</p>
				<p><strong>Vehicle:</strong> %s</p>
				<p><strong>Submission ID:</strong> %s</p>
				<p><strong>Recovery Attempt:</strong> %d of %d</p>
			</div>

			<div style="margin: 30px 0; padding: 20px; background-color: #e8f4f8; border-radius: 5px;">
				<h3 style="margin-top: 0;">How to Complete Your Registration:</h3>
				<ol>
					<li>Click the button below when you're ready to complete payment</li>
					<li>You'll be taken to a secure checkout page</li>
					<li>Complete your payment information</li>
					<li>Your registration will be finalized pending approval</li>
				</ol>
			</div>

			<div style="margin-top: 30px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #28a745; color: white; padding: 15px 30px; text-decoration: none; border-radius: 4px; font-size: 16px; font-weight: bold;">Complete Registration Now</a>
			</div>

			<div style="margin-top: 30px;">
				<p style="color: #666;">If the button doesn't work, copy and paste this link into your browser:</p>
				<p style="word-break: break-all; color: #007bff;">%s</p>
			</div>

			<div style="margin-top: 30px;">
				<p>Need help? Contact us at <a href="mailto:info@theeurohaus.com">info@theeurohaus.com</a> with your submission ID: %s</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, participantName, urgencyHTML, eventName, vehicleDetails, submissionID, attemptNumber, maxAttempts,
		recoveryLink, recoveryLink, submissionID, time.Now().Year())

	return html
}

// generateAbandonmentHTML generates HTML for final abandonment emails
func generateAbandonmentHTML(data map[string]interface{}) string {
	participantName, _ := data["ParticipantName"].(string)
	vehicleDetails, _ := data["VehicleDetails"].(string)
	resubmitLink, _ := data["ResubmitLink"].(string)
	contactEmail, _ := data["ContactEmail"].(string)

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Registration Expired - Euro Haus</title>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="text-align: center; margin-bottom: 30px;">
				<h1>Registration Expired</h1>
			</div>

			<p>Hello %s,</p>

			<p>Unfortunately, your vehicle registration has expired after multiple payment attempts.</p>

			<div style="background-color: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h2 style="margin-top: 0;">Expired Registration</h2>
				<p><strong>Vehicle:</strong> %s</p>
				<p><strong>Status:</strong> Expired</p>
			</div>

			<div style="margin: 30px 0; padding: 20px; background-color: #fff3cd; border-radius: 5px; border: 1px solid #ffc107;">
				<h3 style="margin-top: 0; color: #856404;">What Now?</h3>
				<p>If you're still interested in participating in our events, you'll need to submit a new application.</p>
				<p>Your previous submission details have been archived and cannot be recovered.</p>
			</div>

			<div style="margin-top: 30px; text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #007bff; color: white; padding: 12px 20px; text-decoration: none; border-radius: 4px;">Submit New Application</a>
			</div>

			<div style="margin-top: 30px;">
				<p>If you have questions or believe this is an error, please contact us at <a href="mailto:%s">%s</a>.</p>
			</div>

			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #777;">
				<p>&copy; %d Euro Haus. All rights reserved.</p>
			</div>
		</body>
		</html>
	`, participantName, vehicleDetails, resubmitLink, contactEmail, contactEmail, time.Now().Year())

	return html
}

func findSubmissionByCheckoutSessionID(
	ctx context.Context,
	sessionID string,
) (*models.VehicleSubmissionDTO, error) {
	if sessionID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var submissionID string

	err := services.GetDB().
		WithContext(ctx).
		Raw(`
			SELECT id
			FROM vehicle_submissions
			WHERE checkout_session_id = ?
			LIMIT 1
		`, sessionID).
		Scan(&submissionID).
		Error

	if err != nil {
		return nil, err
	}

	if submissionID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	return getSubmissionByID(submissionID)
}

func findSubmissionByPaymentIntentID(
	ctx context.Context,
	paymentIntentID string,
) (*models.VehicleSubmissionDTO, error) {
	if paymentIntentID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var submissionID string

	err := services.GetDB().
		WithContext(ctx).
		Raw(`
			SELECT id
			FROM vehicle_submissions
			WHERE payment_intent_id = ?
			LIMIT 1
		`, paymentIntentID).
		Scan(&submissionID).
		Error

	if err != nil {
		return nil, err
	}

	if submissionID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	return getSubmissionByID(submissionID)
}

func createSubmissionRecoverySession(
	ctx context.Context,
	submission *models.VehicleSubmissionDTO,
) (*stripe.CheckoutSession, error) {
	if submission == nil {
		return nil, fmt.Errorf("submission is nil")
	}

	priceID := strings.TrimSpace(submission.PriceID)
	if priceID == "" {
		return nil, fmt.Errorf("submission has no price ID")
	}

	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "https://eurohaus.shop"
	}

	requiresApproval := submission.Status != "approved"

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(
			fmt.Sprintf(
				"%s/checkout/pending?submission_id=%s&event_id=%s",
				baseURL,
				url.QueryEscape(submission.ID),
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
			"submission_id":     submission.ID,
			"event_id":          submission.EventID,
			"event_slug":        submission.EventSlug,
			"participant":       "true",
			"requires_approval": strconv.FormatBool(requiresApproval),
			"recovery":          "true",
		},
	}

	if submission.ParticipantEmail != "" {
		params.CustomerEmail = stripe.String(submission.ParticipantEmail)
	}

	if requiresApproval {
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			CaptureMethod: stripe.String("manual"),
			Metadata: map[string]string{
				"submission_id":     submission.ID,
				"event_id":          submission.EventID,
				"participant":       "true",
				"requires_approval": "true",
				"recovery":          "true",
			},
		}
	} else {
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"submission_id":     submission.ID,
				"event_id":          submission.EventID,
				"participant":       "true",
				"requires_approval": "false",
				"recovery":          "true",
			},
		}
	}

	recoverySession, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf(
			"create replacement checkout session: %w",
			err,
		)
	}

	if err := persistCheckoutSessionAfterCreation(
		ctx,
		submission.ID,
		recoverySession.ID,
		priceID,
		submission.PromotionCode,
		requiresApproval,
	); err != nil {
		return nil, fmt.Errorf(
			"persist replacement checkout session: %w",
			err,
		)
	}

	return recoverySession, nil
}
