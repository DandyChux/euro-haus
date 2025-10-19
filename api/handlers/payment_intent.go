package handlers

import (
	"context"
	"encoding/json"
	"euro-haus-api/services"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/tax/calculation"
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
	SelectedShippingRate string `json:"selected_shipping_rate"` // Track which shipping rate was selected
	CustomerAddress      *struct {
		Country    string `json:"country"`
		State      string `json:"state"`
		PostalCode string `json:"postal_code"`
	} `json:"customer_address,omitempty"`
}

type LineItem struct {
	Price     string     `json:"price,omitempty"`      // Stripe Price ID
	PriceData *PriceData `json:"price_data,omitempty"` // For dynamic pricing
	Quantity  int64      `json:"quantity"`
	TaxCode   string     `json:"tax_code,omitempty"`
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

type TaxCalculationRequest struct {
	LineItems []TaxLineItem      `json:"line_items"`
	Address   *TaxAddress        `json:"address,omitempty"`
	Currency  string             `json:"currency"`
	Shipping  *ShippingCostInput `json:"shipping,omitempty"`
}

type TaxLineItem struct {
	Amount    int64  `json:"amount"` // Amount in cents
	Reference string `json:"reference,omitempty"`
	TaxCode   string `json:"tax_code,omitempty"`
}

type TaxAddress struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type ShippingCostInput struct {
	Amount  int64  `json:"amount"` // Amount in cents
	TaxCode string `json:"tax_code,omitempty"`
}

type TaxCalculationResponse struct {
	TaxAmount      int64                  `json:"tax_amount"`      // in cents
	ShippingAmount int64                  `json:"shipping_amount"` // in cents
	Subtotal       int64                  `json:"subtotal"`        // in cents
	Total          int64                  `json:"total"`           // in cents
	ShippingRates  []ShippingRateResponse `json:"shipping_rates"`
	TaxBreakdown   []TaxBreakdownItem     `json:"tax_breakdown,omitempty"`
}

type ShippingRateResponse struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name"`
	Amount      int64             `json:"amount"` // in cents
	Currency    string            `json:"currency"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type TaxBreakdownItem struct {
	Amount       int64  `json:"amount"`
	Inclusive    bool   `json:"inclusive"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
	Percentage   string `json:"percentage,omitempty"`
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

	// Build line items and calculate subtotal
	var lineItems []*stripe.CheckoutSessionLineItemParams
	var subtotal int64 // Track subtotal for shipping calculation

	// Handle line_items from cart
	if len(req.LineItems) > 0 {
		for _, item := range req.LineItems {
			if item.Price != "" {
				// Use existing price ID
				lineItemParam := &stripe.CheckoutSessionLineItemParams{
					Price:    stripe.String(item.Price),
					Quantity: stripe.Int64(item.Quantity),
				}
				// Note: Tax codes cannot be set when using price IDs
				// They should be set on the Price or Product in Stripe
				lineItems = append(lineItems, lineItemParam)

				// For price IDs, we'd need to fetch the price to calculate subtotal
				// This is a limitation when using price IDs
			} else if item.PriceData != nil {
				// Use dynamic pricing
				productData := &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(item.PriceData.ProductData.Name),
					Description: stripe.String(item.PriceData.ProductData.Description),
					Images:      stripe.StringSlice(item.PriceData.ProductData.Images),
					Metadata:    item.PriceData.ProductData.Metadata,
				}

				// Add tax code to product data if provided
				if item.TaxCode != "" {
					productData.TaxCode = stripe.String(item.TaxCode)
				} else {
					// Default tax code for general merchandise
					productData.TaxCode = stripe.String("txcd_99999999")
				}

				lineItem := &stripe.CheckoutSessionLineItemParams{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency:    stripe.String(item.PriceData.Currency),
						ProductData: productData,
						UnitAmount:  stripe.Int64(item.PriceData.UnitAmount),
					},
					Quantity: stripe.Int64(item.Quantity),
				}

				lineItems = append(lineItems, lineItem)

				// Add to subtotal calculation
				subtotal += item.PriceData.UnitAmount * item.Quantity
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

	// Enable automatic tax calculation (requires Stripe Tax to be enabled in your account)
	params.AutomaticTax = &stripe.CheckoutSessionAutomaticTaxParams{
		Enabled: stripe.Bool(true),
	}

	// Check if we need to collect shipping address (for physical products)
	needsShipping := hasPhysicalProducts(req.LineItems) || len(req.AddOns) > 0 || req.PriceID != ""

	if needsShipping {
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US", "CA", "GB", "DE", "FR", "IT", "ES", "NL", "BE"}),
		}

		// Build dynamic shipping options based on subtotal
		shippingOptions := []*stripe.CheckoutSessionShippingOptionParams{}

		// Store select rate from request
		selectedRate := req.SelectedShippingRate

		// Create all possible shipping options
		var options = make(map[string]*stripe.CheckoutSessionShippingOptionParams)

		// Check if order qualifies for free shipping (over $75)
		if subtotal >= 7500 { // $75 in cents
			options["free_standard"] = &stripe.CheckoutSessionShippingOptionParams{
				ShippingRateData: &stripe.CheckoutSessionShippingOptionShippingRateDataParams{
					Type: stripe.String("fixed_amount"),
					FixedAmount: &stripe.CheckoutSessionShippingOptionShippingRateDataFixedAmountParams{
						Amount:   stripe.Int64(0), // Free
						Currency: stripe.String("usd"),
					},
					DisplayName: stripe.String("✨ FREE Standard Shipping (5-7 business days) - Orders over $75"),
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
					TaxBehavior: stripe.String("exclusive"),
					TaxCode:     stripe.String("txcd_92010001"),
				},
			}
		} else {
			options["standard"] = &stripe.CheckoutSessionShippingOptionParams{
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
					TaxBehavior: stripe.String("exclusive"),
					TaxCode:     stripe.String("txcd_92010001"),
				},
			}
		}

		// Always add Express and Overnight options
		options["express"] = &stripe.CheckoutSessionShippingOptionParams{
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
				TaxBehavior: stripe.String("exclusive"),
				TaxCode:     stripe.String("txcd_92010001"),
			},
		}

		options["overnight"] = &stripe.CheckoutSessionShippingOptionParams{
			ShippingRateData: &stripe.CheckoutSessionShippingOptionShippingRateDataParams{
				Type: stripe.String("fixed_amount"),
				FixedAmount: &stripe.CheckoutSessionShippingOptionShippingRateDataFixedAmountParams{
					Amount:   stripe.Int64(3999), // $39.99
					Currency: stripe.String("usd"),
				},
				DisplayName: stripe.String("Overnight Shipping (1 business day)"),
				DeliveryEstimate: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateParams{
					Minimum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMinimumParams{
						Unit:  stripe.String("business_day"),
						Value: stripe.Int64(1),
					},
					Maximum: &stripe.CheckoutSessionShippingOptionShippingRateDataDeliveryEstimateMaximumParams{
						Unit:  stripe.String("business_day"),
						Value: stripe.Int64(1),
					},
				},
				TaxBehavior: stripe.String("exclusive"),
				TaxCode:     stripe.String("txcd_92010001"),
			},
		}

		// Add selected option first (Stripe auto-selects the first option)
		if selectedRate != "" && options[selectedRate] != nil {
			shippingOptions = append(shippingOptions, options[selectedRate])
			delete(options, selectedRate) // Remove from map so we don't add it twice
		}

		// Add remaining options in a logical order
		// Priority order: free/standard, express, overnight
		if opt, exists := options["free_standard"]; exists {
			shippingOptions = append(shippingOptions, opt)
		}
		if opt, exists := options["standard"]; exists {
			shippingOptions = append(shippingOptions, opt)
		}
		if opt, exists := options["express"]; exists {
			shippingOptions = append(shippingOptions, opt)
		}
		if opt, exists := options["overnight"]; exists {
			shippingOptions = append(shippingOptions, opt)
		}

		params.ShippingOptions = shippingOptions
	}

	// Create the session
	sess, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create checkout session: %v", err), http.StatusInternalServerError)
		return
	}

	// Store session details in Redis for webhook processing
	rdb := services.GetRedisClient()
	ctx := context.Background()

	sessionKey := fmt.Sprintf("checkout_session:%s", sess.ID)
	sessionData := map[string]interface{}{
		"created_at": time.Now().Format(time.RFC3339),
		"metadata":   fmt.Sprintf("%v", metadata),
		"subtotal":   subtotal, // Store subtotal for reference
	}

	if req.PriceID != "" {
		sessionData["price_id"] = req.PriceID
	}

	rdb.HSet(ctx, sessionKey, sessionData)
	rdb.Expire(ctx, sessionKey, 24*time.Hour) // Expire after 24 hours

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":       sess.URL,
		"sessionId": sess.ID,
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
	params.AddExpand("total_details")

	sess, err := session.Get(session_id, params)
	if err != nil {
		log.Printf("Failed to retrieve checkout session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Format and return the session data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     sess.ID,
		"status": sess.Status,
		"amount": sess.AmountTotal,
		"customer": map[string]interface{}{
			"email": sess.CustomerEmail,
			"name":  sess.CustomerDetails.Name,
		},
		"items":         formatLineItems(sess),
		"created":       sess.Created,
		"total_details": sess.TotalDetails,
	})
}

// Rest of the functions remain the same...
func CreateEventCheckoutSession(w http.ResponseWriter, r *http.Request) {
	// Keep existing implementation but add AutomaticTax
	var req struct {
		PriceID       string `json:"priceId"`
		Quantity      int64  `json:"quantity"`
		EventID       string `json:"eventId"`
		AddOnProducts []struct {
			PriceID  string `json:"priceId"`
			Quantity int64  `json:"quantity"`
		} `json:"addOnProducts"`
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
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},
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
		params.ShippingOptions = []*stripe.CheckoutSessionShippingOptionParams{
			{
				ShippingRateData: &stripe.CheckoutSessionShippingOptionShippingRateDataParams{
					Type: stripe.String("fixed_amount"),
					FixedAmount: &stripe.CheckoutSessionShippingOptionShippingRateDataFixedAmountParams{
						Amount:   stripe.Int64(999),
						Currency: stripe.String("usd"),
					},
					DisplayName: stripe.String("Standard Shipping"),
					TaxBehavior: stripe.String("exclusive"),
					TaxCode:     stripe.String("txcd_92010001"),
				},
			},
		}
	}

	sess, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":           sess.ID,
		"url":                 sess.URL,
		"hasIncludedProducts": hasIncludedProducts,
		"includedProducts":    includedProducts,
	})
}

// CalculateTaxAndShipping calculates tax and retrieves available shipping rates
func CalculateTaxAndShipping(w http.ResponseWriter, r *http.Request) {
	var req TaxCalculationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default to USD if not provided
	if req.Currency == "" {
		req.Currency = "usd"
	}

	// Calculate subtotal from line items
	var subtotal int64
	for _, item := range req.LineItems {
		subtotal += item.Amount
	}

	// Determine shipping cost based on subtotal
	shippingAmount := getShippingCost(subtotal)

	// Calculate tax
	var taxAmount int64
	var taxBreakdown []TaxBreakdownItem

	if req.Address != nil && req.Address.Country != "" && req.Address.PostalCode != "" {
		// Use Stripe Tax Calculation API for accurate tax rates
		lineItems := make([]*stripe.TaxCalculationLineItemParams, 0, len(req.LineItems))

		for _, item := range req.LineItems {
			lineItem := &stripe.TaxCalculationLineItemParams{
				Amount:    stripe.Int64(item.Amount),
				Reference: stripe.String(item.Reference),
			}
			if item.TaxCode != "" {
				lineItem.TaxCode = stripe.String(item.TaxCode)
			} else {
				// Default tax code for general merchandise
				lineItem.TaxCode = stripe.String("txcd_99999999")
			}
			lineItems = append(lineItems, lineItem)
		}

		// Add shipping as a line item for tax calculation if applicable
		if shippingAmount > 0 {
			lineItems = append(lineItems, &stripe.TaxCalculationLineItemParams{
				Amount:    stripe.Int64(shippingAmount),
				Reference: stripe.String("shipping"),
				TaxCode:   stripe.String("txcd_92010001"), // Shipping tax code
			})
		}

		// Build tax calculation params
		params := &stripe.TaxCalculationParams{
			Currency:  stripe.String(req.Currency),
			LineItems: lineItems,
			CustomerDetails: &stripe.TaxCalculationCustomerDetailsParams{
				Address: &stripe.AddressParams{
					Line1:      stripe.String(req.Address.Line1),
					Line2:      stripe.String(req.Address.Line2),
					City:       stripe.String(req.Address.City),
					State:      stripe.String(req.Address.State),
					PostalCode: stripe.String(req.Address.PostalCode),
					Country:    stripe.String(req.Address.Country),
				},
				AddressSource: stripe.String("shipping"),
			},
		}

		// Try to calculate tax using Stripe Tax API
		calc, err := calculation.New(params)
		if err != nil {
			log.Printf("Stripe Tax calculation error (falling back to simple calculation): %v", err)
			// Fall back to simple tax calculation
			taxAmount = calculateSimpleTax(subtotal + shippingAmount)
		} else {
			taxAmount = calc.TaxAmountExclusive

			// Get tax breakdown if available
			if calc.TaxBreakdown != nil {
				for _, breakdown := range calc.TaxBreakdown {
					taxBreakdown = append(taxBreakdown, TaxBreakdownItem{
						Amount:       breakdown.Amount,
						Inclusive:    breakdown.Inclusive,
						Jurisdiction: string(breakdown.TaxRateDetails.TaxType),
						Percentage:   breakdown.TaxRateDetails.PercentageDecimal + "%",
					})
				}
			}
		}
	} else {
		// No address provided, use simple calculation
		taxAmount = calculateSimpleTax(subtotal + shippingAmount)
	}

	// Calculate total
	total := subtotal + shippingAmount + taxAmount

	// Build shipping rates response
	shippingRates := []ShippingRateResponse{}

	// Add appropriate shipping rates based on subtotal
	if subtotal >= 7500 { // Free shipping for orders over $75
		shippingRates = append(shippingRates, ShippingRateResponse{
			ID:          "free_shipping",
			DisplayName: "FREE Standard Shipping (5-7 business days)",
			Amount:      0,
			Currency:    req.Currency,
			Metadata: map[string]string{
				"delivery_estimate": "5-7 business days",
				"promotion":         "free_shipping_over_75",
			},
		})
	}

	// Standard shipping (always available)
	standardShippingAmount := int64(999) // $9.99
	if subtotal >= 7500 {
		standardShippingAmount = 0 // Free for orders over $75
	}
	shippingRates = append(shippingRates, ShippingRateResponse{
		ID:          "standard_shipping",
		DisplayName: "Standard Shipping (5-7 business days)",
		Amount:      standardShippingAmount,
		Currency:    req.Currency,
		Metadata: map[string]string{
			"delivery_estimate": "5-7 business days",
		},
	})

	// Express shipping
	shippingRates = append(shippingRates, ShippingRateResponse{
		ID:          "express_shipping",
		DisplayName: "Express Shipping (2-3 business days)",
		Amount:      1999, // $19.99
		Currency:    req.Currency,
		Metadata: map[string]string{
			"delivery_estimate": "2-3 business days",
		},
	})

	// Overnight shipping
	shippingRates = append(shippingRates, ShippingRateResponse{
		ID:          "overnight_shipping",
		DisplayName: "Overnight Shipping (1 business day)",
		Amount:      3999, // $39.99
		Currency:    req.Currency,
		Metadata: map[string]string{
			"delivery_estimate": "1 business day",
		},
	})

	// Build and send response
	response := TaxCalculationResponse{
		TaxAmount:      taxAmount,
		ShippingAmount: shippingAmount,
		Subtotal:       subtotal,
		Total:          total,
		ShippingRates:  shippingRates,
		TaxBreakdown:   taxBreakdown,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getShippingCost determines shipping cost based on subtotal
func getShippingCost(subtotal int64) int64 {
	// Free shipping over $75
	if subtotal >= 7500 {
		return 0
	}
	// Flat rate shipping
	return 999 // $9.99 in cents
}

// calculateSimpleTax calculates a simple tax percentage (fallback)
func calculateSimpleTax(amount int64) int64 {
	// 8% tax rate
	return amount * 8 / 100
}

// GetShippingRates retrieves available shipping rates for the cart
func GetShippingRates(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}

	subtotalStr := r.URL.Query().Get("subtotal")
	var subtotal int64
	if subtotalStr != "" {
		if val, err := strconv.ParseInt(subtotalStr, 10, 64); err == nil {
			subtotal = val
		}
	}

	// Build shipping rates based on subtotal
	rates := []ShippingRateResponse{}

	// Check if eligible for free shipping
	if subtotal >= 7500 { // $75 in cents
		rates = append(rates, ShippingRateResponse{
			ID:          "free_standard",
			DisplayName: "FREE Standard Shipping (5-7 business days)",
			Amount:      0,
			Currency:    "usd",
			Metadata: map[string]string{
				"delivery_days": "5-7",
				"eligible":      "true",
			},
		})
	} else {
		rates = append(rates, ShippingRateResponse{
			ID:          "standard",
			DisplayName: "Standard Shipping (5-7 business days)",
			Amount:      999, // $9.99
			Currency:    "usd",
			Metadata: map[string]string{
				"delivery_days": "5-7",
			},
		})
	}

	// Always offer express shipping
	rates = append(rates, ShippingRateResponse{
		ID:          "express",
		DisplayName: "Express Shipping (2-3 business days)",
		Amount:      1999, // $19.99
		Currency:    "usd",
		Metadata: map[string]string{
			"delivery_days": "2-3",
		},
	})

	// Overnight option
	rates = append(rates, ShippingRateResponse{
		ID:          "overnight",
		DisplayName: "Overnight Shipping (1 business day)",
		Amount:      3999, // $39.99
		Currency:    "usd",
		Metadata: map[string]string{
			"delivery_days": "1",
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rates)
}

// Helper function to format line items from the session
func formatLineItems(sess *stripe.CheckoutSession) []map[string]interface{} {
	items := []map[string]interface{}{}

	if sess.LineItems != nil {
		for _, item := range sess.LineItems.Data {
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

// hasPhysicalProducts checks if the line items contain physical products that need shipping
func hasPhysicalProducts(items []LineItem) bool {
	for _, item := range items {
		if item.PriceData != nil && item.PriceData.ProductData != nil {
			// Check if it's not an event ticket or digital product
			productType := item.PriceData.ProductData.Metadata["type"]
			if productType != "event" && productType != "digital" {
				return true
			}
		}
		// If we don't have metadata, assume it needs shipping (conservative approach)
		if item.Price != "" {
			return true
		}
	}
	return false
}
