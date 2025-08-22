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
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/webhook"
)

// HandleWebhook processes Stripe webhook events
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Verify webhook signature
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		log.Printf("STRIPE_WEBHOOK_SECRET environment variable not set\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	signatureHeader := r.Header.Get("Stripe-Signature")
	if signatureHeader == "" {
		log.Printf("Missing Stripe-Signature header\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
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

	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func handlePaymentIntentSucceeded(pi stripe.PaymentIntent) {
	// Handle successful payment
	log.Printf("PaymentIntent succeeded: %s, Amount: %d %s\n",
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

// New function to handle participant payment success
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

	// Check if approval email was already sent
	approvalEmailSent := submissionData["approval_email_sent"] == "true"

	// If submission is approved but email wasn't sent, send it now
	if submissionData["status"] == "approved" && !approvalEmailSent {
		// Send approval email with payment confirmation
		vehicleDetails := fmt.Sprintf("%s %s %s",
			submissionData["vehicle_year"],
			submissionData["vehicle_make"],
			submissionData["vehicle_model"])

		emailData := map[string]interface{}{
			"ParticipantName": submissionData["participant_name"],
			"VehicleDetails":  vehicleDetails,
			"EventID":         submissionData["event_id"],
			"PaymentLink":     "Payment completed successfully!",
			"ReviewNotes":     submissionData["review_notes"],
		}

		msg := &services.EmailMessage{
			To:           []string{submissionData["participant_email"]},
			Subject:      "Your Vehicle Submission Has Been Approved! - Euro Haus",
			TemplateID:   "submission-approved",
			TemplateData: emailData,
			BodyHTML:     generateApprovalEmailHTML(emailData),
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

	// Create and store ticket
	ticketToken := generateUniqueToken()
	ticketKey := "ticket:" + ticketToken

	eventName := pi.Metadata["event_name"]
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
		"payment_intent_id": pi.ID,
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
	})

	// Add to event attendees
	eventAttendeesKey := fmt.Sprintf("event:%s:attendees", submissionData["event_id"])
	rdb.SAdd(ctx, eventAttendeesKey, ticketToken)

	// Send participant ticket email
	sendParticipantTicketEmail(submissionData, ticketToken, eventName)
}

func handleCheckoutSessionCompleted(checkoutSession stripe.CheckoutSession) {
	// Handle completed checkout session
	fmt.Printf("Checkout session completed: %s\n", checkoutSession.ID)

	// Check if this is a participant submission checkout
	if submissionID, ok := checkoutSession.Metadata["submission_id"]; ok && submissionID != "" {
		handleParticipantCheckout(checkoutSession, submissionID)
		return
	}

	// Expand the session to get line items
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items.data.price.product")

	fullSession, err := session.Get(checkoutSession.ID, params)
	if err != nil {
		log.Printf("Error expanding session: %v\n", err)
		return
	}

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
			hasEventTickets = true
			updateEventInventory(lineItem.Price.Product.ID, lineItem.Quantity)
			// Store ticket purchase information and send ticket email
			storeTicketPurchase(*fullSession, *lineItem)
		}

		// Check for physical products
		if lineItem.Price.Product.Metadata["type"] != "event" {
			hasPhysicalProducts = true
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

	fmt.Printf("Order fulfilled for session: %s, Customer: %s\n", fullSession.ID, customerEmail)
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

	// Check if this is a participant submission checkout (has manual capture)
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
	if customerEmail == "" {
		log.Printf("No email address available for expired checkout session %s", checkoutSession.ID)
		return
	}

	// If this is a participant submission with manual capture, create a new checkout session
	if submissionID != "" && isParticipant {
		handleExpiredParticipantCheckout(checkoutSession, submissionID, customerEmail)
	} else {
		// Handle regular abandoned cart
		handleRegularAbandonedCart(checkoutSession, customerEmail)
	}
}

// handleExpiredParticipantCheckout creates a new checkout session for an expired participant submission
func handleExpiredParticipantCheckout(expiredSession stripe.CheckoutSession, submissionID string, customerEmail string) {
	fmt.Printf("Handling expired participant checkout for submission: %s\n", submissionID)

	// Get submission from Redis
	submission, err := getSubmissionByID(submissionID)
	if err != nil {
		log.Printf("Failed to get submission %s: %v", submissionID, err)
		// Still send a recovery email even if we can't get the submission
		sendGenericRecoveryEmail(customerEmail, submissionID)
		return
	}

	// Get the line items from the expired session (need to expand it first)
	expandParams := &stripe.CheckoutSessionParams{}
	expandParams.AddExpand("line_items")
	fullSession, err := session.Get(expiredSession.ID, expandParams)
	if err != nil {
		log.Printf("Failed to expand expired session %s: %v", expiredSession.ID, err)
		sendGenericRecoveryEmail(customerEmail, submissionID)
		return
	}

	// Extract price ID and quantity from the expired session
	var priceID string
	var quantity int64 = 1

	if fullSession.LineItems != nil && len(fullSession.LineItems.Data) > 0 {
		lineItem := fullSession.LineItems.Data[0]
		if lineItem.Price != nil {
			priceID = lineItem.Price.ID
		}
		quantity = lineItem.Quantity
	}

	if priceID == "" {
		log.Printf("Could not extract price ID from expired session %s", expiredSession.ID)
		sendGenericRecoveryEmail(customerEmail, submissionID)
		return
	}

	// Create a new checkout session with the same parameters
	baseUrl := os.Getenv("BASE_URL")
	newParams := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(quantity),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s/checkout/pending?submission_id=%s", baseUrl, submissionID)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/checkout/cancel", baseUrl)),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			CaptureMethod: stripe.String("manual"),
			Metadata: map[string]string{
				"submission_id":          submissionID,
				"event_id":               submission.EventID,
				"event_name":             expiredSession.Metadata["event_name"],
				"participant":            "true",
				"recovered_from_session": expiredSession.ID,
			},
		},
		Metadata: map[string]string{
			"submission_id":          submissionID,
			"event_id":               submission.EventID,
			"event_name":             expiredSession.Metadata["event_name"],
			"participant":            "true",
			"recovered_from_session": expiredSession.ID,
		},
		CustomerEmail: stripe.String(customerEmail),
		ExpiresAt:     stripe.Int64(time.Now().Add(24 * time.Hour).Unix()), // Set explicit 24-hour expiration
	}

	// Create the new checkout session
	newSession, err := session.New(newParams)
	if err != nil {
		log.Printf("Failed to create recovery checkout session for submission %s: %v", submissionID, err)
		sendGenericRecoveryEmail(customerEmail, submissionID)
		return
	}

	// Update the submission with the new checkout session ID
	rdb := services.GetRedisClient()
	ctx := context.Background()
	submissionKey := fmt.Sprintf("submission:%s", submissionID)

	updates := map[string]interface{}{
		"checkout_session_id":          newSession.ID,
		"previous_checkout_session_id": expiredSession.ID,
		"checkout_recovered_at":        time.Now().Format(time.RFC3339),
		"checkout_recovery_count":      "1", // Track recovery attempts
	}

	if err := rdb.HSet(ctx, submissionKey, updates).Err(); err != nil {
		log.Printf("Failed to update submission %s with new checkout session: %v", submissionID, err)
	}

	// Send recovery email with the new payment link
	emailData := map[string]interface{}{
		"ParticipantName": submission.ParticipantName,
		"VehicleDetails":  fmt.Sprintf("%s %s %s", submission.VehicleYear, submission.VehicleMake, submission.VehicleModel),
		"EventName":       expiredSession.Metadata["event_name"],
		"PaymentLink":     newSession.URL,
		"ExpirationTime":  time.Now().Add(24 * time.Hour).Format(time.RFC1123),
		"SubmissionID":    submissionID,
	}

	msg := &services.EmailMessage{
		To:           []string{customerEmail},
		Subject:      "Action Required: Complete Your Euro Haus Registration",
		TemplateID:   "participant-checkout-recovery",
		TemplateData: emailData,
		BodyHTML:     generateParticipantRecoveryHTML(emailData),
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending recovery email for submission %s: %v", submissionID, err)
	}

	fmt.Printf("Successfully created recovery checkout session %s for submission %s", newSession.ID, submissionID)
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
		"customer_email":             session.CustomerDetails.Email,
		"customer_name":              session.CustomerDetails.Name,
		"quantity":                   lineItem.Quantity,
		"purchase_date":              time.Now().Format(time.RFC3339),
		"checked_in":                 "false",
		"event_name":                 lineItem.Price.Product.Name,
	}

	// Store the ticket data
	ticketKey := "ticket:" + token
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
	eventAttendeesKey := "event:" + productID + ":attendees"
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
