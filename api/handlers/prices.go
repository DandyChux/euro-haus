package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// GetProductPrices returns all prices for a specific product
func GetProductPrices(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get product ID from URL
	vars := mux.Vars(r)
	productID := vars["id"]

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	// Check if product exists first
	_, err := product.Get(productID, nil)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		log.Printf("Error checking product %s: %v", productID, err)
		http.Error(w, "Failed to verify product", http.StatusInternalServerError)
		return
	}

	// Fetch all active prices for this product
	params := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}
	params.Filters.AddFilter("limit", "", "100")

	iter := price.List(params)
	var prices []map[string]interface{}

	for iter.Next() {
		p := iter.Price()
		priceData := map[string]interface{}{
			"id":          p.ID,
			"product":     productID,
			"unit_amount": p.UnitAmount,
			"currency":    string(p.Currency),
			"nickname":    p.Nickname,
			"active":      p.Active,
			"metadata":    p.Metadata,
		}
		prices = append(prices, priceData)
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error fetching prices for product %s: %v", productID, err)
		http.Error(w, "Failed to fetch prices", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, even if empty
	if prices == nil {
		prices = []map[string]interface{}{}
	}

	response := map[string]interface{}{
		"prices": prices,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// CreateCheckoutSessionWithPrice creates a checkout session for a specific price
func CreateCheckoutSessionWithPrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PriceID  string            `json:"priceId"`
		Quantity int64             `json:"quantity"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create checkout session with the specific price
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
		SuccessURL: stripe.String("https://yourdomain.com/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://yourdomain.com/cancel"),
		Metadata:   req.Metadata,
	}

	session, err := session.New(params)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"sessionId": session.ID,
	}

	json.NewEncoder(w).Encode(response)
}
