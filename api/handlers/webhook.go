package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"euro-haus-api/services"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Constants for recovery limits and timeframes
const (
	MaxRecoveryAttempts = 2
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

// isSubmissionEligibleForRecovery checks if a submission should have a recovery session created
func isSubmissionEligibleForRecovery(submissionData map[string]string) (bool, string) {
	// Check if payment was already completed
	if submissionData["payment_completed"] == "true" {
		return false, "Payment already completed"
	}

	// Check if ticket was already created
	if submissionData["ticket_id"] != "" {
		return false, "Ticket already exists"
	}

	// Check submission age
	if createdAt, ok := submissionData["created_at"]; ok {
		if created, err := time.Parse(time.RFC3339, createdAt); err == nil {
			if time.Since(created) > MaxSubmissionAge {
				return false, "Submission too old"
			}
		}
	}

	// Check recovery attempts
	if recoveryCount, ok := submissionData["checkout_recovery_count"]; ok {
		if count, err := strconv.Atoi(recoveryCount); err == nil && count >= MaxRecoveryAttempts {
			return false, fmt.Sprintf("Maximum recovery attempts (%d) reached", MaxRecoveryAttempts)
		}
	}

	// Check last recovery attempt time (prevent rapid retries)
	if lastRecovery, ok := submissionData["last_recovery_attempt_at"]; ok {
		if lastTime, err := time.Parse(time.RFC3339, lastRecovery); err == nil {
			if time.Since(lastTime) < MinRecoveryInterval {
				return false, "Too soon since last recovery attempt"
			}
		}
	}

	// Check if submission was cancelled or rejected
	if status := submissionData["status"]; status == "cancelled" || status == "rejected" {
		return false, fmt.Sprintf("Submission status is %s", status)
	}

	return true, ""
}

func handlePaymentIntentSucceeded(pi stripe.PaymentIntent) {
	// Handle successful payment
	fmt.Printf("PaymentIntent succeeded: %s, Amount: %d %s\n",
		pi.ID, pi.Amount, pi.Currency)

	// Check if this is a participant payment
	if submissionID, ok := pi.Metadata["submission_id"]; ok && submissionID != "" {
		handleParticipantPaymentSucceeded(pi, submissionID)
		return
	}

	// Get customer details from metadata
	var customerEmail, customerName string
	if metadata, ok := pi.Metadata["customer_name"]; ok {
		customerName = metadata
	}

	// Get receipt email from metadata or receipt email field
	if pi.ReceiptEmail != "" {
		customerEmail = pi.ReceiptEmail
	} else if email, ok := pi.Metadata["customer_email"]; ok {
		customerEmail = email
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

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending payment confirmation email: %v", err)
	}
}

// handleParticipantPaymentSucceeded handles payment success for participant submissions
func handleParticipantPaymentSucceeded(pi stripe.PaymentIntent, submissionID string) {
	log.Printf("Handling participant payment for submission: %s\n", submissionID)

	// Get Redis client
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get submission details
	submissionKey := fmt.Sprintf("submission:%s", submissionID)
	submissionData, err := rdb.HGetAll(ctx, submissionKey).Result()
	if err != nil {
		log.Printf("Error retrieving submission data: %v", err)
		return
	}

	// CRITICAL: Check if submission is approved before creating ticket
	if submissionData["status"] != "approved" {
		log.Printf("WARNING: Payment succeeded for unapproved submission %s (status: %s). Not creating ticket.",
			submissionID, submissionData["status"])

		// Update submission to note this unusual state
		rdb.HSet(ctx, submissionKey, map[string]interface{}{
			"payment_succeeded_before_approval": "true",
			"payment_intent_id":                 pi.ID,
			"payment_succeeded_at":              time.Now().Format(time.RFC3339),
		})

		// Don't create ticket - wait for approval
		return
	}

	// Check if ticket was already created (prevent duplicates)
	if existingTicketID := submissionData["ticket_id"]; existingTicketID != "" {
		log.Printf("Ticket already exists for submission %s: %s", submissionID, existingTicketID)
		return
	}

	// Check if approval email was already sent
	approvalEmailSent := submissionData["approval_email_sent"] == "true"

	// Create and store ticket
	ticketToken := generateUniqueToken()
	ticketKey := "ticket:" + ticketToken

	eventName := pi.Metadata["event_name"]
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	// Get ticket tier information
	ticketTier := submissionData["ticket_tier"]
	if ticketTier == "" {
		ticketTier = "Participant"
	}

	ticketData := map[string]interface{}{
		"customer_name":     submissionData["participant_name"],
		"customer_email":    submissionData["participant_email"],
		"event_name":        eventName,
		"stripe_product_id": submissionData["event_id"],
		"quantity":          submissionData["ticket_quantity"],
		"ticket_type":       ticketTier,
		"purchased_at":      time.Now().Format(time.RFC3339),
		"checked_in":        "false",
		"submission_id":     submissionID,
		"vehicle_details": fmt.Sprintf("%s %s %s",
			submissionData["vehicle_year"],
			submissionData["vehicle_make"],
			submissionData["vehicle_model"]),
		"payment_intent_id":    pi.ID,
		"approved_participant": "true",
	}

	// Store ticket in Redis
	if err := rdb.HSet(ctx, ticketKey, ticketData).Err(); err != nil {
		log.Printf("Error storing ticket: %v", err)
		return
	}

	// Update submission with ticket ID
	rdb.HSet(ctx, submissionKey, map[string]interface{}{
		"ticket_id":            ticketToken,
		"payment_completed":    "true",
		"payment_completed_at": time.Now().Format(time.RFC3339),
		"ticket_created_at":    time.Now().Format(time.RFC3339),
	})

	// Add to event attendees
	eventAttendeesKey := fmt.Sprintf("event:%s:attendees", submissionData["event_id"])
	rdb.SAdd(ctx, eventAttendeesKey, ticketToken)

	// Send participant ticket email
	sendParticipantTicketEmail(submissionData, ticketToken, eventName)

	// Mark that ticket email was sent
	rdb.HSet(ctx, submissionKey, map[string]interface{}{
		"ticket_email_sent":    "true",
		"ticket_email_sent_at": time.Now().Format(time.RFC3339),
	})

	log.Printf("Successfully created and sent ticket %s for approved participant submission %s",
		ticketToken, submissionID)

	// If approval email wasn't sent earlier, send a combined approval + ticket email
	if !approvalEmailSent {
		vehicleDetails := fmt.Sprintf("%s %s %s",
			submissionData["vehicle_year"],
			submissionData["vehicle_make"],
			submissionData["vehicle_model"])

		emailData := map[string]interface{}{
			"ParticipantName": submissionData["participant_name"],
			"VehicleDetails":  vehicleDetails,
			"EventID":         submissionData["event_id"],
			"TicketCode":      ticketToken,
			"ReviewNotes":     submissionData["review_notes"],
		}

		msg := &services.EmailMessage{
			To:           []string{submissionData["participant_email"]},
			Subject:      "Your Vehicle Submission Has Been Approved + Ticket - Euro Haus",
			TemplateID:   "submission-approved-with-ticket",
			TemplateData: emailData,
			BodyHTML:     generateApprovalWithTicketEmailHTML(emailData),
		}

		if err := services.SendEmail(msg); err != nil {
			log.Printf("Error sending approval email: %v", err)
		} else {
			// Update email sent status
			rdb.HSet(ctx, submissionKey, map[string]interface{}{
				"approval_email_sent":    "true",
				"approval_email_sent_at": time.Now().Format(time.RFC3339),
			})
		}
	}
}

func handleCheckoutSessionCompleted(checkoutSession stripe.CheckoutSession) {
	// Handle completed checkout session
	fmt.Printf("Checkout session completed: %s\n", checkoutSession.ID)

	// Check if this is a participant submission checkout
	if submissionID, ok := checkoutSession.Metadata["submission_id"]; ok && submissionID != "" {
		// This is a participant submission
		isParticipant := checkoutSession.Metadata["participant"] == "true"

		if isParticipant {
			// Check if this requires approval (has manual capture)
			if checkoutSession.PaymentIntent != nil {
				// Get the payment intent to check capture method
				pi, err := paymentintent.Get(checkoutSession.PaymentIntent.ID, nil)
				if err == nil && pi.CaptureMethod == "manual" {
					// This needs approval - DON'T create ticket yet
					fmt.Printf("Participant checkout pending approval for submission: %s\n", submissionID)

					// Just update the submission with checkout details
					rdb := services.GetRedisClient()
					ctx := context.Background()
					submissionKey := fmt.Sprintf("submission:%s", submissionID)

					updates := map[string]interface{}{
						"checkout_session_id":   checkoutSession.ID,
						"payment_intent_id":     checkoutSession.PaymentIntent.ID,
						"checkout_completed":    "true",
						"awaiting_approval":     "true",
						"checkout_completed_at": time.Now().Format(time.RFC3339),
					}

					if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
						log.Printf("Error updating submission with checkout details: %v", err)
					}

					// Don't create ticket - wait for approval and payment capture
					log.Printf("Submission %s checkout completed, awaiting approval before ticket creation", submissionID)
					return
				}
			}

			// If we get here, it's auto-approved or payment doesn't require manual capture
			// This would happen for auto-approved tiers
			handleParticipantCheckout(checkoutSession, submissionID)
			return
		}
	}

	// Expand the session to get line items for regular (non-participant) purchases
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items.data.price.product")

	fullSession, err := session.Get(checkoutSession.ID, params)
	if err != nil {
		log.Printf("Error expanding session: %v\n", err)
		return
	}

	// Process stock updates for all items in the checkout session
	processStockUpdates(fullSession)

	// Keep track of products for order confirmation email
	productItems := []map[string]interface{}{}
	hasEventTickets := false
	hasPhysicalProducts := false
	totalAmount := 0.0

	// Process each line item
	for _, lineItem := range fullSession.LineItems.Data {
		if lineItem.Price.Product == nil {
			continue
		}

		// Add to products list for order email
		productItems = append(productItems, map[string]interface{}{
			"Name":        lineItem.Price.Product.Name,
			"Description": lineItem.Price.Product.Description,
			"Quantity":    lineItem.Quantity,
			"Price":       float64(lineItem.Price.UnitAmount) / 100.0,
			"Currency":    string(lineItem.Price.Currency),
			"Subtotal":    float64(lineItem.AmountSubtotal) / 100.0,
			"Type":        lineItem.Price.Product.Metadata["type"],
		})

		totalAmount += float64(lineItem.AmountTotal) / 100.0

		// Check if this is an event ticket
		if lineItem.Price.Product.Metadata["type"] == "event" {
			// Check if this specific checkout is for a participant
			isParticipantCheckout := fullSession.Metadata["participant"] == "true"

			if !isParticipantCheckout {
				// Regular attendee - create ticket immediately
				storeTicketPurchase(*fullSession, *lineItem)
			} else {
				log.Printf("Skipping immediate ticket creation for participant checkout %s", fullSession.ID)
			}
		}

		// Check for physical products
		if lineItem.Price.Product.Metadata["type"] != "event" {
			hasPhysicalProducts = true
		}
	}

	// Process any bundled products if this was an event with add-ons
	if fullSession.Metadata != nil {
		// Check if this checkout included add-ons or tier bundles
		if fullSession.Metadata["has_addons"] == "true" || fullSession.Metadata["has_bundled_products"] == "true" {
			customerEmail := fullSession.CustomerEmail
			if customerEmail == "" && fullSession.Customer != nil {
				// Get customer email from customer object if needed
				customerEmail = fullSession.Customer.Email
			}

			customerName := fullSession.CustomerDetails.Name
			if customerName == "" && fullSession.Metadata["customer_name"] != "" {
				customerName = fullSession.Metadata["customer_name"]
			}

			// Process bundled products
			if err := ProcessBundledProducts(fullSession.ID, customerEmail, customerName); err != nil {
				log.Printf("Error processing bundled products for session %s: %v", fullSession.ID, err)
				// Don't fail the webhook, just log the error
			}
		}
	}

	// Get customer details
	customerEmail := fullSession.CustomerDetails.Email
	customerName := fullSession.CustomerDetails.Name

	// Don't send general order confirmation if we don't have an email address
	if customerEmail == "" {
		log.Printf("No email address available for checkout session %s", fullSession.ID)
		return
	}

	// Send order confirmation email for the entire order
	// Note: Individual tickets are sent separately in storeTicketPurchase
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

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Order Confirmation - Euro Haus",
		TemplateID:   "order-confirmation",
		TemplateData: orderData,
		// Fallback if template not found
		BodyHTML: generateOrderConfirmationHTML(orderData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending order confirmation email: %v", err)
	}

	fmt.Printf("Order processed for session: %s, Customer: %s\n", fullSession.ID, customerEmail)
}

// handleParticipantCheckout handles checkout completion for approved vehicle submissions
func handleParticipantCheckout(checkoutSession stripe.CheckoutSession, submissionID string) {
	fmt.Printf("Handling participant checkout for submission: %s\n", submissionID)

	// Get Redis client
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Update submission with payment details
	submissionKey := fmt.Sprintf("submission:%s", submissionID)
	updates := map[string]interface{}{
		"checkout_session_id":  checkoutSession.ID,
		"payment_intent_id":    checkoutSession.PaymentIntent.ID,
		"payment_completed_at": time.Now().Format(time.RFC3339),
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		log.Printf("Error updating submission payment details: %v", err)
	}

	// Get full submission details
	submissionData, err := rdb.HGetAll(ctx, submissionKey).Result()
	if err != nil {
		log.Printf("Error retrieving submission data: %v", err)
		return
	}

	// Create and store ticket
	ticketToken := generateUniqueToken() // Fixed: was generateTicketToken()
	ticketKey := "ticket:" + ticketToken

	eventName := checkoutSession.Metadata["event_name"]
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	ticketData := map[string]interface{}{
		"customer_name":     submissionData["participant_name"],
		"customer_email":    submissionData["participant_email"],
		"event_name":        eventName,
		"stripe_product_id": submissionData["event_id"],
		"quantity":          "1",
		"ticket_type":       "Participant",
		"purchased_at":      time.Now().Format(time.RFC3339),
		"checked_in":        "false",
		"submission_id":     submissionID,
		"vehicle_details":   fmt.Sprintf("%s %s %s", submissionData["vehicle_year"], submissionData["vehicle_make"], submissionData["vehicle_model"]),
	}

	// Store ticket in Redis
	if err := rdb.HSet(ctx, ticketKey, ticketData).Err(); err != nil {
		log.Printf("Error storing ticket: %v", err)
		return
	}

	// Update submission with ticket ID
	rdb.HSet(ctx, submissionKey, "ticket_id", ticketToken)

	// Add to event attendees
	eventAttendeesKey := fmt.Sprintf("event:%s:attendees", submissionData["event_id"])
	rdb.SAdd(ctx, eventAttendeesKey, ticketToken)

	// Send participant ticket email
	sendParticipantTicketEmail(submissionData, ticketToken, eventName)
}

func handlePaymentIntentFailed(pi stripe.PaymentIntent) {
	// Handle failed payment
	log.Printf("PaymentIntent failed: %s, Reason: %s\n", pi.ID, pi.LastPaymentError.Msg)

	// Get customer details from metadata or receipt email
	var customerEmail string
	if pi.ReceiptEmail != "" {
		customerEmail = pi.ReceiptEmail
	} else if email, ok := pi.Metadata["customer_email"]; ok {
		customerEmail = email
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
		"RecoveryURL":   os.Getenv("BASE_URL") + "/checkout/recover?session=" + pi.ID,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Payment Failed - Euro Haus",
		TemplateID:   "payment-failed",
		TemplateData: emailData,
		// Fallback if template not found
		BodyHTML: generatePaymentFailedHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending payment failure email: %v", err)
	}
}

func handleCheckoutSessionExpired(checkoutSession stripe.CheckoutSession) {
	// Handle expired checkout session
	fmt.Printf("Checkout session expired: %s\n", checkoutSession.ID)

	// Get Redis client for tracking
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Check if we've already processed this expired session (idempotency)
	processedKey := fmt.Sprintf("processed:expired:%s", checkoutSession.ID)
	if exists, _ := rdb.Exists(ctx, processedKey).Result(); exists > 0 {
		log.Printf("Already processed expired session %s, skipping", checkoutSession.ID)
		return
	}

	// Mark this session as processed (with 7-day TTL)
	rdb.SetEx(ctx, processedKey, "true", 7*24*time.Hour)

	// Check if this is a participant submission checkout
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

	// Get customer email
	customerEmail := checkoutSession.CustomerEmail
	if customerEmail == "" && checkoutSession.CustomerDetails != nil {
		customerEmail = checkoutSession.CustomerDetails.Email
	}

	if customerEmail == "" {
		log.Printf("No email address available for expired checkout session %s", checkoutSession.ID)
		return
	}

	// Handle based on type
	if submissionID != "" && isParticipant {
		handleExpiredParticipantCheckout(checkoutSession, submissionID, customerEmail)
	} else {
		// Handle regular abandoned cart (only send email, don't create new session)
		handleRegularAbandonedCart(checkoutSession, customerEmail)
	}
}

func handleExpiredParticipantCheckout(expiredSession stripe.CheckoutSession, submissionID string, customerEmail string) {
	fmt.Printf("Handling expired participant checkout for submission: %s\n", submissionID)

	// Get Redis client
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get submission data
	submissionKey := fmt.Sprintf("submission:%s", submissionID)
	submissionData, err := rdb.HGetAll(ctx, submissionKey).Result()
	if err != nil {
		log.Printf("Failed to get submission %s: %v", submissionID, err)
		return
	}

	// Check if submission exists and has data
	if len(submissionData) == 0 {
		log.Printf("Submission %s not found or empty", submissionID)
		return
	}

	// Check if submission is eligible for recovery
	eligible, reason := isSubmissionEligibleForRecovery(submissionData)
	if !eligible {
		log.Printf("Submission %s not eligible for recovery: %s", submissionID, reason)

		// Update submission to mark as abandoned if max attempts reached
		if strings.Contains(reason, "Maximum recovery attempts") {
			rdb.HSet(ctx, submissionKey, map[string]interface{}{
				"status":           "abandoned",
				"abandoned_at":     time.Now().Format(time.RFC3339),
				"abandoned_reason": reason,
			})

			// Send final abandonment notification
			sendFinalAbandonmentEmail(customerEmail, submissionData)
		}
		return
	}

	// Get current recovery count
	currentCount := 0
	if countStr, ok := submissionData["checkout_recovery_count"]; ok {
		currentCount, _ = strconv.Atoi(countStr)
	}

	// Instead of creating a new session automatically, just send a recovery email
	// with instructions for the user to manually restart the process

	// Update submission recovery tracking
	updates := map[string]interface{}{
		"last_expired_session_id":  expiredSession.ID,
		"last_recovery_attempt_at": time.Now().Format(time.RFC3339),
		"checkout_recovery_count":  strconv.Itoa(currentCount + 1),
		"awaiting_customer_action": "true",
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		log.Printf("Failed to update submission %s recovery tracking: %v", submissionID, err)
	}

	// Send recovery email with manual action required
	sendManualRecoveryEmail(customerEmail, submissionData, submissionID, currentCount+1)

	log.Printf("Sent recovery email for submission %s (attempt %d of %d)",
		submissionID, currentCount+1, MaxRecoveryAttempts)
}

// handleChargeRefunded processes refund events and invalidates associated tickets
func handleChargeRefunded(charge stripe.Charge) {
	fmt.Printf("Processing refund for charge: %s", charge.ID)

	// Get payment intent ID from charge
	paymentIntentID := charge.PaymentIntent.ID
	if paymentIntentID == "" {
		log.Printf("No payment intent associated with charge %s", charge.ID)
		return
	}

	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Find tickets associated with this payment intent
	pattern := "ticket:*"
	iter := rdb.Scan(ctx, 0, pattern, 0).Iterator()

	refundedTickets := []string{}
	customerEmail := ""
	customerName := ""
	eventName := ""

	for iter.Next(ctx) {
		key := iter.Val()
		ticketData, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}

		// Check if this ticket is associated with the refunded payment
		if ticketData["stripe_payment_intent_id"] == paymentIntentID {
			ticketToken := strings.TrimPrefix(key, "ticket:")

			// Store customer info for email
			if customerEmail == "" {
				customerEmail = ticketData["customer_email"]
				customerName = ticketData["customer_name"]
				eventName = ticketData["event_name"]
			}

			// Invalidate the ticket
			if err := InvalidateTicket(ticketToken, "Payment refunded"); err != nil {
				log.Printf("Error invalidating ticket %s: %v", ticketToken, err)
			} else {
				refundedTickets = append(refundedTickets, ticketToken)
			}
		}
	}

	if len(refundedTickets) > 0 && customerEmail != "" {
		// Send refund notification email
		sendRefundNotificationEmail(customerEmail, customerName, eventName, refundedTickets, charge.AmountRefunded)
		log.Printf("Invalidated %d tickets for refunded payment %s", len(refundedTickets), paymentIntentID)
	}
}

// handlePaymentIntentCanceled processes canceled payment intents
func handlePaymentIntentCanceled(pi stripe.PaymentIntent) {
	log.Printf("Processing canceled payment intent: %s", pi.ID)

	// Check if this is a participant submission
	if pi.Metadata != nil && pi.Metadata["submission_id"] != "" {
		submissionID := pi.Metadata["submission_id"]

		rdb := services.GetRedisClient()
		ctx := context.Background()

		// Update submission status
		submissionKey := fmt.Sprintf("submission:%s", submissionID)
		updates := map[string]interface{}{
			"payment_status": "canceled",
			"updated_at":     time.Now().Format(time.RFC3339),
		}

		if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
			log.Printf("Error updating submission %s: %v", submissionID, err)
		}
	}

	// Find and invalidate any tickets
	rdb := services.GetRedisClient()
	ctx := context.Background()

	pattern := "ticket:*"
	iter := rdb.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		ticketData, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}

		if ticketData["stripe_payment_intent_id"] == pi.ID {
			ticketToken := strings.TrimPrefix(key, "ticket:")
			if err := InvalidateTicket(ticketToken, "Payment canceled"); err != nil {
				log.Printf("Error invalidating ticket %s: %v", ticketToken, err)
			}
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

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending refund notification email to %s: %v", customerEmail, err)
	} else {
		log.Printf("Sent refund notification to %s for %d tickets", customerEmail, len(ticketTokens))
	}
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

// sendManualRecoveryEmail sends an email asking the user to manually restart checkout
func sendManualRecoveryEmail(customerEmail string, submissionData map[string]string, submissionID string, attemptNumber int) {
	participantName := submissionData["participant_name"]
	vehicleDetails := fmt.Sprintf("%s %s %s",
		submissionData["vehicle_year"],
		submissionData["vehicle_make"],
		submissionData["vehicle_model"])
	eventName := submissionData["event_name"]
	if eventName == "" {
		eventName = "Euro Haus Event"
	}

	baseURL := os.Getenv("BASE_URL")

	// Create a recovery link that goes to your frontend to restart the process
	recoveryLink := fmt.Sprintf("%s/submissions/recover?id=%s&email=%s",
		baseURL, submissionID, customerEmail)

	isLastAttempt := attemptNumber >= MaxRecoveryAttempts

	emailData := map[string]interface{}{
		"ParticipantName": participantName,
		"VehicleDetails":  vehicleDetails,
		"EventName":       eventName,
		"RecoveryLink":    recoveryLink,
		"SubmissionID":    submissionID,
		"AttemptNumber":   attemptNumber,
		"MaxAttempts":     MaxRecoveryAttempts,
		"IsLastAttempt":   isLastAttempt,
		"BaseURL":         baseURL,
	}

	subject := "Action Required: Complete Your Euro Haus Registration"
	if isLastAttempt {
		subject = "Final Reminder: Complete Your Euro Haus Registration"
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      subject,
		TemplateID:   "participant-manual-recovery",
		TemplateData: emailData,
		BodyHTML:     generateManualRecoveryHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending manual recovery email for submission %s: %v", submissionID, err)
	}
}

// sendFinalAbandonmentEmail sends a final email when max recovery attempts are reached
func sendFinalAbandonmentEmail(customerEmail string, submissionData map[string]string) {
	participantName := submissionData["participant_name"]
	vehicleDetails := fmt.Sprintf("%s %s %s",
		submissionData["vehicle_year"],
		submissionData["vehicle_make"],
		submissionData["vehicle_model"])

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

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending abandonment email: %v", err)
	}
}

// handleRegularAbandonedCart handles regular abandoned cart (non-participant)
func handleRegularAbandonedCart(checkoutSession stripe.CheckoutSession, customerEmail string) {
	// Send standard abandoned cart email
	emailData := map[string]interface{}{
		"SessionID":      checkoutSession.ID,
		"ExpirationTime": time.Now().Format(time.RFC1123),
		"RecoveryURL":    os.Getenv("BASE_URL") + "/checkout/recover?session=" + checkoutSession.ID,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Complete Your Euro Haus Purchase",
		TemplateID:   "checkout-abandoned",
		TemplateData: emailData,
		BodyHTML:     generateAbandonedCartHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending abandoned cart email: %v", err)
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

func updateEventInventory(productID string, quantitySold int64) {
	// Get current product
	p, err := product.Get(productID, nil)
	if err != nil {
		log.Printf("Error fetching product %s: %v\n", productID, err)
		return
	}

	// Parse current available spots
	currentSpots, err := strconv.Atoi(p.Metadata["available_spots"])
	if err != nil {
		log.Printf("Error parsing available_spots for product %s: %v\n", productID, err)
		return
	}

	// Calculate new available spots
	newSpots := currentSpots - int(quantitySold)
	if newSpots < 0 {
		newSpots = 0
	}

	// Prepare update params
	updateParams := &stripe.ProductParams{}
	updateParams.AddMetadata("available_spots", strconv.Itoa(newSpots))

	// Update status if sold out
	if newSpots == 0 {
		updateParams.AddMetadata("status", "soldout")
	}

	// Update the product
	_, err = product.Update(productID, updateParams)
	if err != nil {
		log.Printf("Error updating product %s inventory: %v\n", productID, err)
		return
	}

	log.Printf("Updated inventory for event %s: %d spots remaining\n", productID, newSpots)
}

// updateProductVariantStock updates the stock quantity for a specific product variant (price)
func updateProductVariantStock(priceID string, quantitySold int64) {
	// Get the current price with its metadata
	p, err := price.Get(priceID, nil)
	if err != nil {
		log.Printf("Error fetching price %s: %v\n", priceID, err)
		return
	}

	// Check if this price has stock tracking
	stockQuantityStr, hasStock := p.Metadata["stock_quantity"]
	if !hasStock || stockQuantityStr == "" {
		// No stock tracking for this variant, skip
		log.Printf("Price %s does not have stock tracking, skipping\n", priceID)
		return
	}

	// Parse current stock
	currentStock, err := strconv.Atoi(stockQuantityStr)
	if err != nil {
		log.Printf("Error parsing stock_quantity for price %s: %v\n", priceID, err)
		return
	}

	// Calculate new stock
	newStock := currentStock - int(quantitySold)
	if newStock < 0 {
		newStock = 0
	}

	// Prepare update params - we need to update ALL metadata fields
	// because Stripe replaces the entire metadata object
	updateParams := &stripe.PriceParams{
		Metadata: p.Metadata, // Start with existing metadata
	}

	// Update stock quantity
	updateParams.Metadata["stock_quantity"] = strconv.Itoa(newStock)

	// Update in_stock status based on new quantity
	if newStock == 0 {
		updateParams.Metadata["in_stock"] = "false"
	} else {
		updateParams.Metadata["in_stock"] = "true"
	}

	// Update the price
	_, err = price.Update(priceID, updateParams)
	if err != nil {
		log.Printf("Error updating price %s stock: %v\n", priceID, err)
		return
	}

	// Log the stock update
	variantName := p.Nickname
	if variantName == "" {
		variantName = p.Metadata["variant"]
		if variantName == "" {
			variantName = priceID
		}
	}

	log.Printf("Updated stock for variant %s: %d -> %d units remaining\n", variantName, currentStock, newStock)

	// Send low stock alert if needed
	// if newStock > 0 && newStock <= 5 {
	// 	sendLowStockAlert(p, newStock)
	// }
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

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending low stock alert: %v", err)
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
			productType := lineItem.Price.Product.Metadata["type"]

			switch productType {
			case "event":
				// Update event inventory
				updateEventInventory(lineItem.Price.Product.ID, quantitySold)

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
	p, err := product.Get(bundleProductID, nil)
	if err != nil {
		log.Printf("Error fetching bundle product %s: %v\n", bundleProductID, err)
		return
	}

	// Parse bundle items from metadata
	bundleItemsJSON, exists := p.Metadata["bundle_items"]
	if !exists || bundleItemsJSON == "" {
		log.Printf("Bundle %s has no bundle_items metadata\n", bundleProductID)
		return
	}

	// Parse the JSON to get bundle items
	var bundleItems []struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}

	if err := json.Unmarshal([]byte(bundleItemsJSON), &bundleItems); err != nil {
		log.Printf("Error parsing bundle_items for %s: %v\n", bundleProductID, err)
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

		// Calculate total quantity to decrement (bundle quantity * item quantity * quantity sold)
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

// storeTicketPurchase stores ticket info in Redis after purchase
func storeTicketPurchase(session stripe.CheckoutSession, lineItem stripe.LineItem) {
	// SAFETY CHECK: Don't create tickets for unapproved participant submissions
	if session.Metadata != nil {
		if submissionID, ok := session.Metadata["submission_id"]; ok && submissionID != "" {
			if session.Metadata["participant"] == "true" {
				// This is a participant checkout - verify approval status
				rdb := services.GetRedisClient()
				ctx := context.Background()

				submissionKey := fmt.Sprintf("submission:%s", submissionID)
				submissionData, err := rdb.HGetAll(ctx, submissionKey).Result()

				if err != nil {
					log.Printf("ERROR: Could not verify submission status for %s: %v", submissionID, err)
					return
				}

				if submissionData["status"] != "approved" {
					log.Printf("WARNING: Attempted to create ticket for unapproved participant submission %s (status: %s). Blocking ticket creation.",
						submissionID, submissionData["status"])
					return
				}

				// Check if ticket already exists
				if existingTicket := submissionData["ticket_id"]; existingTicket != "" {
					log.Printf("Ticket already exists for submission %s: %s", submissionID, existingTicket)
					return
				}

				fmt.Printf("Participant submission %s is approved, proceeding with ticket creation", submissionID)
			}
		}
	}

	// Generate unique token for this ticket
	token := generateUniqueToken()
	productID := lineItem.Price.Product.ID

	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get customer details
	customerEmail := session.CustomerDetails.Email
	customerName := session.CustomerDetails.Name

	// Store ticket information as a Redis Hash
	ticketData := map[string]interface{}{
		"stripe_checkout_session_id": session.ID,
		"stripe_payment_intent_id":   session.PaymentIntent.ID,
		"stripe_product_id":          productID,
		"customer_email":             customerEmail,
		"customer_name":              customerName,
		"quantity":                   lineItem.Quantity,
		"purchase_date":              time.Now().Format(time.RFC3339),
		"checked_in":                 "false",
		"event_name":                 lineItem.Price.Product.Name,
		"ticket_type":                "General Admission", // Default for non-participants
	}

	// Add submission info if this is a participant
	if submissionID, ok := session.Metadata["submission_id"]; ok && submissionID != "" {
		ticketData["submission_id"] = submissionID
		ticketData["ticket_type"] = "Participant"
	}

	// Check if this cvstomer already has a valid ticket for this event
	eventAttendeesKey := fmt.Sprintf("event:%s:attendees", productID)
	existingTokens, _ := rdb.SMembers(ctx, eventAttendeesKey).Result()

	for _, token := range existingTokens {
		ticketKey := fmt.Sprintf("ticket:%s", token)
		existingTicket, _ := rdb.HGetAll(ctx, ticketKey).Result()

		// Check if it's the same customer
		if existingTicket["customer_email"] == customerEmail {
			// Check if this is a valid/completed purchase
			if existingTicket["stripe_payment_intent_id"] != "" {
				fmt.Printf("Customer already has a valid ticket for event %s", productID)

				// Update the session ID if it's different (recovery scenario)
				if existingTicket["stripe_checkout_session_id"] != session.ID {
					rdb.HSet(ctx, ticketKey, "stripe_checkout_session_id", session.ID)
					fmt.Printf("Updated checkout session ID for existing ticket")
				}
				return // Don't create a new ticket (duplicate)
			} else {
				// Found an incomplete ticket - we should clean it up and create a new one
				log.Printf("Found incomplete ticket %s for customer %s, removing it", token, customerEmail)
				rdb.Del(ctx, ticketKey)
				rdb.SRem(ctx, eventAttendeesKey, token)
			}
		}
	}

	// Store the ticket data
	ticketKey := fmt.Sprintf("ticket:%s", token)
	if err := rdb.HSet(ctx, ticketKey, ticketData).Err(); err != nil {
		log.Printf("Error storing ticket in Redis: %v", err)
		return
	}

	// Extract event date from product metadata to set appropriate TTL
	eventDate := time.Now().Add(30 * 24 * time.Hour) // Default: 30 days from now
	if dateStr, ok := lineItem.Price.Product.Metadata["event_date"]; ok {
		if parsedDate, err := time.Parse(time.RFC3339, dateStr); err == nil {
			// Set expiration to 30 days after the event date
			eventDate = parsedDate.Add(30 * 24 * time.Hour)
		}
	}

	// Set TTL for the ticket data (30 days after event)
	rdb.ExpireAt(ctx, ticketKey, eventDate)

	// Add ticket to the event's attendee set with same TTL
	if err := rdb.SAdd(ctx, eventAttendeesKey, token).Err(); err != nil {
		log.Printf("Error adding ticket to event attendees: %v", err)
	}
	rdb.ExpireAt(ctx, eventAttendeesKey, eventDate)

	// Also set TTL for checked-in set
	checkedInKey := "event:" + productID + ":checked_in"
	rdb.ExpireAt(ctx, checkedInKey, eventDate)

	// Generate QR code for the ticket
	qrCode, err := generateQRCode(token)
	if err != nil {
		log.Printf("Error generating QR code: %v", err)
	}

	// Create event details map for the email
	eventDetails := map[string]interface{}{
		"name":     lineItem.Price.Product.Name,
		"metadata": lineItem.Price.Product.Metadata,
		"quantity": lineItem.Quantity,
	}

	// Send email with ticket info
	err = services.SendTicketEmail(customerEmail, customerName, token, eventDetails, qrCode)
	if err != nil {
		log.Printf("Error sending ticket email: %v", err)
	}

	fmt.Printf("Successfully created ticket %s for customer %s", token, customerEmail)
}

// sendParticipantTicketEmail sends a special ticket email for approved vehicle participants
func sendParticipantTicketEmail(submissionData map[string]string, ticketToken string, eventName string) {
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

	emailData := map[string]interface{}{
		"CustomerName":   submissionData["participant_name"],
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
	`, emailData["CustomerName"], vehicleDetails, emailData["EventName"], emailData["TicketCode"], emailData["QRCodeURL"])

	msg := &services.EmailMessage{
		To:           []string{submissionData["participant_email"]},
		Subject:      fmt.Sprintf("Event Participant Ticket - %s", eventName),
		TemplateID:   "participant-ticket",
		TemplateData: emailData,
		BodyHTML:     ticketHTML,
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending participant ticket email: %v", err)
	}
}

func sendGenericRecoveryEmail(customerEmail string, submissionID string) {
	baseUrl := os.Getenv("BASE_URL")

	// Create a recovery URL that leads to a page where they can re-initiate checkout
	recoveryURL := fmt.Sprintf("%s/checkout/recover?submission=%s", baseUrl, submissionID)

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

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending recovery email for submission %s: %v", submissionID, err)
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
