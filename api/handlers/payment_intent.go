package handlers

import (
	"context"
	"encoding/json"
	"euro-haus-api/services"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
)

type CreatePaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// Updated to support cart checkout with multiple items
type CreateCheckoutSessionRequest struct {
	LineItems           []LineItem        `json:"line_items"`
	Mode                string            `json:"mode"`
	SuccessURL          string            `json:"success_url"`
	CancelURL           string            `json:"cancel_url"`
	Metadata            map[string]string `json:"metadata"`
	AllowPromotionCodes bool              `json:"allow_promotion_codes"`
	PromotionCode       string            `json:"promotion_code"`
	CouponID            string            `json:"coupon_id"`
	PriceID             string            `json:"priceId"`
	Quantity            int64             `json:"quantity"`
	AddOns              []struct {
		PriceID  string `json:"priceId"`
		Quantity int64  `json:"quantity"`
	} `json:"addons"`
}

type LineItem struct {
	Price     string     `json:"price,omitempty"`      // Stripe Price ID
	PriceData *PriceData `json:"price_data,omitempty"` // For dynamic pricing
	Quantity  int64      `json:"quantity"`
}

type PriceData struct {
	Currency    string       `json:"currency"`
	ProductData *ProductData `json:"product_data"`
	UnitAmount  int64        `json:"unit_amount"`
}

type ProductData struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Images      []string          `json:"images"`
	Metadata    map[string]string `json:"metadata"`
}

// CreatePaymentIntent creates a new payment intent for custom payment flows
func CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentIntentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set default currency if not provided
	if req.Currency == "" {
		req.Currency = "usd"
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"clientSecret": pi.ClientSecret,
	})
}

// CreateCheckoutSession creates a Stripe Checkout session for cart checkout
func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req CreateCheckoutSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build line items
	var lineItems []*stripe.CheckoutSessionLineItemParams

	// Handle line_items from cart
	if len(req.LineItems) > 0 {
		for _, item := range req.LineItems {
			if item.Price != "" {
				// Use existing price ID
				lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
					Price:    stripe.String(item.Price),
					Quantity: stripe.Int64(item.Quantity),
				})
			} else if item.PriceData != nil {
				// Use dynamic pricing
				lineItem := &stripe.CheckoutSessionLineItemParams{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String(item.PriceData.Currency),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name:        stripe.String(item.PriceData.ProductData.Name),
							Description: stripe.String(item.PriceData.ProductData.Description),
							Images:      stripe.StringSlice(item.PriceData.ProductData.Images),
							Metadata:    item.PriceData.ProductData.Metadata,
						},
						UnitAmount: stripe.Int64(item.PriceData.UnitAmount),
					},
					Quantity: stripe.Int64(item.Quantity),
				}
				lineItems = append(lineItems, lineItem)
			}
		}
	}

	// Also handle the old format with priceId for backward compatibility
	if req.PriceID != "" {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(req.PriceID),
			Quantity: stripe.Int64(req.Quantity),
		})
	}

	// Add any add-on products
	for _, addon := range req.AddOns {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(addon.PriceID),
			Quantity: stripe.Int64(addon.Quantity),
		})
	}

	// Validate we have line items
	if len(lineItems) == 0 {
		http.Error(w, "No line items provided", http.StatusBadRequest)
		return
	}

	// Set up metadata
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// Track if this checkout includes add-ons
	if len(req.AddOns) > 0 {
		metadata["has_addons"] = "true"
		metadata["addon_count"] = fmt.Sprintf("%d", len(req.AddOns))

		// Store add-on details for fulfillment
		addonsJSON, _ := json.Marshal(req.AddOns)
		metadata["addons"] = string(addonsJSON)
	}

	// Set default URLs
	baseURL := os.Getenv("BASE_URL")
	successURL := req.SuccessURL
	cancelURL := req.CancelURL

	if successURL == "" {
		successURL = baseURL + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"
	}
	if cancelURL == "" {
		cancelURL = baseURL + "/checkout/cancel"
	}

	// Create checkout session params
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(stripe.CheckoutSessionModePayment),
		LineItems:          lineItems,
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		Metadata:           metadata,
	}

	// Add customer email if provided
	if email, ok := metadata["customer_email"]; ok && email != "" {
		params.CustomerEmail = stripe.String(email)
	}

	// Add promotion codes if requested
	if req.AllowPromotionCodes {
		params.AllowPromotionCodes = stripe.Bool(true)
	}

	// Check if we need to collecting shipping address (for physical products)
	if hasPhysicalProducts(req.LineItems) || len(req.AddOns) > 0 || req.PriceID != "" {
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US", "CA", "GB", "DE", "FR", "IT", "ES", "NL", "BE"}),
		}
	}

	// Create the session
	session, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	// Store session details in Redis for webhook processing
	rdb := services.GetRedisClient()
	ctx := context.Background()

	sessionKey := fmt.Sprintf("checkout_session:%s", session.ID)
	sessionData := map[string]interface{}{
		"created_at": time.Now().Format(time.RFC3339),
		"metadata":   fmt.Sprintf("%v", metadata),
	}

	if req.PriceID != "" {
		sessionData["price_id"] = req.PriceID
	}

	rdb.HSet(ctx, sessionKey, sessionData)
	rdb.Expire(ctx, sessionKey, 24*time.Hour) // Expire after 24 hours

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":       session.URL,
		"sessionId": session.ID,
	})
}

// GetCheckoutSession retrieves details of a specific checkout session by ID
func GetCheckoutSession(w http.ResponseWriter, r *http.Request) {
	// Get session ID from query parameters
	session_id := r.URL.Query().Get("session_id")
	if session_id == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Expand related objects to include line items, customer details, etc.
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items")
	params.AddExpand("customer")
	params.AddExpand("payment_intent")

	session, err := session.Get(session_id, params)
	if err != nil {
		log.Printf("Failed to retrieve checkout session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Format and return the session data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     session.ID,
		"status": session.Status,
		"amount": session.AmountTotal,
		"customer": map[string]interface{}{
			"email": session.CustomerEmail,
			"name":  session.CustomerDetails.Name,
		},
		"items":   formatLineItems(session),
		"created": session.Created,
	})
}

func CreateEventCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PriceID       string `json:"priceId"`
		Quantity      int64  `json:"quantity"`
		EventID       string `json:"eventId"`
		AddOnProducts []struct {
			PriceID  string `json:"priceId"`
			Quantity int64  `json:"quantity"`
		} `json:"addOnProducts"` // Additional products customer wants to purchase
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get the tier price details
	tierPrice, err := price.Get(req.PriceID, nil)
	if err != nil {
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	// Build line items
	lineItems := []*stripe.CheckoutSessionLineItemParams{
		{
			Price:    stripe.String(req.PriceID),
			Quantity: stripe.Int64(req.Quantity),
		},
	}

	// Track included products for fulfillment
	var includedProducts []map[string]interface{}
	hasIncludedProducts := false

	// Check if this tier has included products
	if includedJSON, ok := tierPrice.Metadata["included_products"]; ok && includedJSON != "" {
		if err := json.Unmarshal([]byte(includedJSON), &includedProducts); err == nil {
			hasIncludedProducts = len(includedProducts) > 0
		}
	}

	// Add any additional products the customer is purchasing
	for _, addon := range req.AddOnProducts {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(addon.PriceID),
			Quantity: stripe.Int64(addon.Quantity),
		})
	}

	// Prepare metadata
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	metadata["event_id"] = req.EventID
	metadata["tier_price_id"] = req.PriceID

	if hasIncludedProducts {
		metadata["has_included_products"] = "true"
		includedJSON, _ := json.Marshal(includedProducts)
		metadata["included_products"] = string(includedJSON)
	}

	if len(req.AddOnProducts) > 0 {
		metadata["has_addon_products"] = "true"
		addonsJSON, _ := json.Marshal(req.AddOnProducts)
		metadata["addon_products"] = string(addonsJSON)
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes:  stripe.StringSlice([]string{"card"}),
		Mode:                stripe.String(stripe.CheckoutSessionModePayment),
		LineItems:           lineItems,
		SuccessURL:          stripe.String(os.Getenv("BASE_URL") + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:           stripe.String(os.Getenv("BASE_URL") + "/events/" + req.EventID),
		Metadata:            metadata,
		AllowPromotionCodes: stripe.Bool(true),
	}

	// Add customer email if provided
	if email, ok := metadata["customer_email"]; ok && email != "" {
		params.CustomerEmail = stripe.String(email)
	}

	// If there are physical products, collect shipping address
	if hasIncludedProducts || len(req.AddOnProducts) > 0 {
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US", "CA"}),
		}
	}

	session, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":           session.ID,
		"url":                 session.URL,
		"hasIncludedProducts": hasIncludedProducts,
		"includedProducts":    includedProducts,
	})
}

// Helper function to format line items from the session
func formatLineItems(session *stripe.CheckoutSession) []map[string]interface{} {
	items := []map[string]interface{}{}

	if session.LineItems != nil {
		for _, item := range session.LineItems.Data {
			items = append(items, map[string]interface{}{
				"id":       item.ID,
				"name":     item.Description,
				"quantity": item.Quantity,
				"amount":   item.AmountTotal,
			})
		}
	}

	return items
}

// Helper function to check if cart contains physical products
func hasPhysicalProducts(items []LineItem) bool {
	for _, item := range items {
		if item.PriceData != nil && item.PriceData.ProductData != nil {
			// Check if it's not an event ticket
			if item.PriceData.ProductData.Metadata["type"] != "event" {
				return true
			}
		}
	}
	return false
}
