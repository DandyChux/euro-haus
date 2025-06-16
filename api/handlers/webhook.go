package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

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
	signatureHeader := r.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	// TODO: Send confirmation email
	// TODO: Update order status in database
}

func handleCheckoutSessionCompleted(checkoutSession stripe.CheckoutSession) {
	// Handle completed checkout session
	log.Printf("Checkout session completed: %s\n", checkoutSession.ID)

	// Expand the session to get line items
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items.data.price.product")

	fullSession, err := session.Get(checkoutSession.ID, params)
	if err != nil {
		log.Printf("Error expanding session: %v\n", err)
		return
	}

	// Process each line item
	for _, lineItem := range fullSession.LineItems.Data {
		if lineItem.Price.Product == nil {
			continue
		}

		// Check if this is an event ticket
		if lineItem.Price.Product.Metadata["type"] == "event" {
			updateEventInventory(lineItem.Price.Product.ID, lineItem.Quantity)
		}
	}

	// TODO: Send order confirmation email
	// TODO: Create fulfillment records
	// TODO: Update customer records

	log.Printf("Order fulfilled for session: %s, Customer: %s\n",
		checkoutSession.ID, checkoutSession.CustomerEmail)
}

func handlePaymentIntentFailed(pi stripe.PaymentIntent) {
	// Handle failed payment
	log.Printf("PaymentIntent failed: %s\n", pi.ID)

	// TODO: Send failure notification to customer
	// TODO: Log the failure reason
	// TODO: Update order status
}

func handleCheckoutSessionExpired(checkoutSession stripe.CheckoutSession) {
	// Handle expired checkout session
	log.Printf("Checkout session expired: %s\n", checkoutSession.ID)

	// TODO: Restore inventory for any held items
	// TODO: Send abandonment email to customer
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
