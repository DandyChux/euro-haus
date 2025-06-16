package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

type CreatePaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// Updated to support cart checkout with multiple items
type CreateCheckoutSessionRequest struct {
	LineItems  []LineItem        `json:"line_items"`
	Mode       string            `json:"mode"`
	SuccessURL string            `json:"success_url"`
	CancelURL  string            `json:"cancel_url"`
	Metadata   map[string]string `json:"metadata"`
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
		log.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build line items for Stripe
	var stripeLineItems []*stripe.CheckoutSessionLineItemParams

	for _, item := range req.LineItems {
		lineItem := &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(item.Quantity),
		}

		if item.Price != "" {
			// Use existing Stripe Price ID
			lineItem.Price = stripe.String(item.Price)
		} else if item.PriceData != nil {
			// Create price data dynamically
			lineItem.PriceData = &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(item.PriceData.Currency),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(item.PriceData.ProductData.Name),
					Description: stripe.String(item.PriceData.ProductData.Description),
					Images:      stripe.StringSlice(item.PriceData.ProductData.Images),
					Metadata:    item.PriceData.ProductData.Metadata,
				},
				UnitAmount: stripe.Int64(item.PriceData.UnitAmount),
			}
		}

		stripeLineItems = append(stripeLineItems, lineItem)
	}

	// Create checkout session params
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:  stripeLineItems,
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Metadata:   req.Metadata,
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		ShippingAddressCollection: &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US", "CA"}),
		},
		BillingAddressCollection: stripe.String("required"),
	}

	// Add shipping options for physical products
	if hasPhysicalProducts(req.LineItems) {
		params.ShippingOptions = []*stripe.CheckoutSessionShippingOptionParams{
			{
				ShippingRateData: &stripe.CheckoutSessionShippingOptionShippingRateDataParams{
					Type: stripe.String("fixed_amount"),
					FixedAmount: &stripe.CheckoutSessionShippingOptionShippingRateDataFixedAmountParams{
						Amount:   stripe.Int64(999), // $9.99
						Currency: stripe.String("usd"),
					},
					DisplayName: stripe.String("Standard Shipping (5-7 business days)"),
					DeliveryEstimate: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateParams{
						Minimum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMinimumParams{
							Unit:  stripe.String("business_day"),
							Value: stripe.Int64(5),
						},
						Maximum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMaximumParams{
							Unit:  stripe.String("business_day"),
							Value: stripe.Int64(7),
						},
					},
				},
			},
			{
				ShippingRateData: &stripe.CheckoutSessionShippingOptionShippingRateDataParams{
					Type: stripe.String("fixed_amount"),
					FixedAmount: &stripe.CheckoutSessionShippingOptionShippingRateDataFixedAmountParams{
						Amount:   stripe.Int64(1999), // $19.99
						Currency: stripe.String("usd"),
					},
					DisplayName: stripe.String("Express Shipping (2-3 business days)"),
					DeliveryEstimate: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateParams{
						Minimum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMinimumParams{
							Unit:  stripe.String("business_day"),
							Value: stripe.Int64(2),
						},
						Maximum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMaximumParams{
							Unit:  stripe.String("business_day"),
							Value: stripe.Int64(3),
						},
					},
				},
			},
		}
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Failed to create checkout session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sessionId": s.ID,
		"url":       s.URL,
	})
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
