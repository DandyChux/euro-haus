package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/shippingrate"
	"github.com/stripe/stripe-go/v82/tax/calculation"
)

type CreatePaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type CheckoutAddOn struct {
	PriceID  string `json:"price_id"`
	Quantity int64  `json:"quantity"`
}

type CheckoutCustomerAddress struct {
	Country    string `json:"country"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

type CreateCheckoutSessionRequest struct {
	LineItems           []LineItem `json:"line_items"`
	Mode                string     `json:"mode"`
	SuccessURL          string     `json:"success_url"`
	CancelURL           string     `json:"cancel_url"`
	AllowPromotionCodes bool       `json:"allow_promotion_codes"`
	PromotionCode       string     `json:"promotion_code"`
	CouponID            string     `json:"coupon_id"`
	PriceID             string     `json:"price_id"`
	Quantity            int64      `json:"quantity"`
	AddOns              []CheckoutAddOn `json:"add_ons"`

	EventID       string `json:"event_id"`
	CustomerEmail string `json:"customer_email"`

	SelectedShippingRate string                    `json:"selected_shipping_rate"`
	CustomerAddress      *CheckoutCustomerAddress `json:"customer_address,omitempty"`
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
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	Physical    bool     `json:"physical"`
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
	ID               string `json:"id"`
	DisplayName      string `json:"display_name"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	DeliveryEstimate string `json:"delivery_estimate,omitempty"`
	Promotion        string `json:"promotion,omitempty"`
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
	var subtotal int64

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

	metadata := make(map[string]string)

	if req.EventID != "" {
		metadata["event_id"] = req.EventID
	}

	if req.PriceID != "" {
		metadata["price_id"] = req.PriceID
	}

	if req.Quantity > 0 {
		metadata["quantity"] = strconv.FormatInt(
			req.Quantity,
			10,
		)
	}

	eventID := req.EventID

	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	successURL := baseURL +
		"/checkout/success?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := baseURL + "/checkout/cancel"

	if eventID != "" {
		successURL += "&event_id=" + url.QueryEscape(eventID)
		cancelURL += "?event_id=" + url.QueryEscape(eventID)
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(stripe.CheckoutSessionModePayment),
		LineItems:          lineItems,

		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata:            metadata,
		AllowPromotionCodes: stripe.Bool(true),
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

		// FETCH ACTUAL SHIPPING RATES FROM STRIPE
		shippingRateParams := &stripe.ShippingRateListParams{
			Active: stripe.Bool(true),
		}

		iter := shippingrate.List(shippingRateParams)
		shippingOptions := []*stripe.CheckoutSessionShippingOptionParams{}

		for iter.Next() {
			rate := iter.ShippingRate()

			// Use the actual shipping rate ID from your dashboard
			shippingOptions = append(shippingOptions, &stripe.CheckoutSessionShippingOptionParams{
				ShippingRate: stripe.String(rate.ID), // Use existing rate ID, not ShippingRateData
			})
		}

		if err := iter.Err(); err != nil {
			log.Printf("Error fetching shipping rates: %v", err)
			// Optionally fall back to your custom rates or return error
			http.Error(w, "Error configuring shipping", http.StatusInternalServerError)
			return
		}

		if len(shippingOptions) > 0 {
			params.ShippingOptions = shippingOptions
		} else {
			log.Println("Warning: No active shipping rates found")
			// Decide whether to proceed without shipping or create fallback rates
		}
	}

	// Create the session
	sess, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create checkout session: %v", err), http.StatusInternalServerError)
		return
	}

	// Store session details in Postgres for webhook processing
	db := services.GetDB()

	sessionMetadata := map[string]interface{}{}
	for k, v := range metadata {
		sessionMetadata[k] = v
	}
	sessionMetadata["created_at"] = time.Now().Format(time.RFC3339)
	sessionMetadata["subtotal"] = subtotal
	if req.PriceID != "" {
		sessionMetadata["price_id"] = req.PriceID
	}

	sessionJSON, err := json.Marshal(sessionMetadata)
	if err != nil {
		log.Printf("Failed to marshal checkout session metadata: %v", err)
	} else {
		err = db.WithContext(r.Context()).Exec(`
			INSERT INTO checkout_sessions (session_id, metadata, expires_at)
			VALUES (?, ?, NOW() + INTERVAL '24 hours')
			ON CONFLICT (session_id) DO UPDATE SET
				metadata = checkout_sessions.metadata || EXCLUDED.metadata,
				expires_at = EXCLUDED.expires_at
		`, sess.ID, sessionJSON).Error
		if err != nil {
			log.Printf("Failed to store checkout session metadata: %v", err)
		}

	}

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

type CreateEventCheckoutSessionRequest struct {
	PriceID       string         `json:"price_id"`
	Quantity      int64          `json:"quantity"`
	EventID       string         `json:"event_id"`
	AddOnProducts []CheckoutAddOn `json:"addon_products"`
	CustomerEmail string         `json:"customer_email"`
}


func CreateEventCheckoutSession(w http.ResponseWriter, r *http.Request) {
	// Keep existing implementation but add AutomaticTax
	var req CreateEventCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	event, err := findActiveEventByID(r.Context(), req.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be greater than zero", http.StatusBadRequest)
		return
	}

	if event.AvailableSpots < int(req.Quantity) {
		http.Error(w, "Not enough event inventory available", http.StatusConflict)
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
		http.Error(w, "Price does not belong to this event", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Unable to validate event price", http.StatusInternalServerError)
		return
	}

	requiresSubmission := eventPrice.RequiresSubmission

	maxQuantity := eventPrice.Quantity

	if requiresSubmission {
		maxQuantity = 1
	}

	if maxQuantity < 1 {
		maxQuantity = 1
	}

	requestedQuantity := req.Quantity

	if requestedQuantity < 1 {
		requestedQuantity = 1
	}

	if requestedQuantity > int64(maxQuantity) {
		requestedQuantity = int64(maxQuantity)
	}

	priceParams := &stripe.PriceParams{}
	priceParams.AddExpand("product")

	stripePrice, err := price.Get(req.PriceID, priceParams)
	if err != nil {
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	if stripePrice.Product == nil || stripePrice.Product.ID != event.StripeProductID {
		http.Error(w, "Price does not belong to this event", http.StatusBadRequest)
		return
	}

	// Build line items
	lineItems := []*stripe.CheckoutSessionLineItemParams{
		{
			Price:    stripe.String(req.PriceID),
			Quantity: stripe.Int64(requestedQuantity),
		},
	}

	var includedProducts []models.PriceIncludedProduct

	err = services.GetDB().
		WithContext(r.Context()).
		Where("price_id = ?", req.PriceID).
		Order("sort_order ASC, product_id ASC").
		Find(&includedProducts).
		Error

	if err != nil {
		http.Error(
			w,
			"Unable to retrieve included products",
			http.StatusInternalServerError,
		)
		return
	}

	hasIncludedProducts := len(includedProducts) > 0

	// Add any additional products the customer is purchasing
	for _, addon := range req.AddOnProducts {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(addon.PriceID),
			Quantity: stripe.Int64(addon.Quantity),
		})
	}

	metadata := map[string]string{
		"event_id": event.ID,
		"price_id": req.PriceID,
		"quantity": strconv.FormatInt(requestedQuantity, 10),
	}

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

	if req.CustomerEmail != "" {
		params.CustomerEmail = stripe.String(req.CustomerEmail)
	}

	// If there are physical products, collect shipping address
	if hasIncludedProducts || len(req.AddOnProducts) > 0 {
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US", "CA", "GB", "DE", "FR", "IT", "ES", "NL", "BE"}),
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
		"session_id":           sess.ID,
		"url":                 sess.URL,
		"has_included_products": hasIncludedProducts,
		"included_products":    includedProducts,
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
				Reference: stripe.String("product_shipment"),
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
			// Metadata: map[string]string{
			// 	"delivery_estimate": "5-7 business days",
			// 	"promotion":         "free_shipping_over_75",
			// },
			DeliveryEstimate: "5-7 business days",
			Promotion: "free_shipping_over_75",
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
		DeliveryEstimate: "5-7 business days",
	})

	// Express shipping
	shippingRates = append(shippingRates, ShippingRateResponse{
		ID:          "express_shipping",
		DisplayName: "Express Shipping (2-3 business days)",
		Amount:      1999, // $19.99
		Currency:    req.Currency,
		DeliveryEstimate: "2-3 business days",
	})

	// Overnight shipping
	shippingRates = append(shippingRates, ShippingRateResponse{
		ID:          "overnight_shipping",
		DisplayName: "Overnight Shipping (1 business day)",
		Amount:      3999, // $39.99
		Currency:    req.Currency,
		DeliveryEstimate: "1 business day",
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

	// FETCH ACTUAL SHIPPING RATES FROM STRIPE
	params := &stripe.ShippingRateListParams{
		Active: stripe.Bool(true), // Only get active rates
	}

	iter := shippingrate.List(params)
	rates := []ShippingRateResponse{}

	for iter.Next() {
		rate := iter.ShippingRate()

		// Optionally filter based on metadata or other criteria
		// For example, you could filter by delivery estimate or metadata tags
		// Also consider subtotal for conditional free shipping
		_ = subtotal // Use subtotal for future filtering logic

		rates = append(rates, ShippingRateResponse{
			ID:          rate.ID, // Use the actual Stripe rate ID (shr_xxxx)
			DisplayName: rate.DisplayName,
			Amount:      rate.FixedAmount.Amount,
			Currency:    string(rate.FixedAmount.Currency),
		})
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error fetching shipping rates: %v", err)
		http.Error(w, "Error fetching shipping rates", http.StatusInternalServerError)
		return
	}

	// If no rates found, you can optionally fall back to hardcoded rates
	// or return an error
	if len(rates) == 0 {
		log.Println("Warning: No active shipping rates found in Stripe dashboard")
		// Optionally return fallback rates here
	}

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

func hasPhysicalProducts(items []LineItem) bool {
	for _, item := range items {
		if item.PriceData != nil &&
			item.PriceData.ProductData != nil &&
			item.PriceData.ProductData.Physical {
			return true
		}

		if item.Price != "" {
			return true
		}
	}

	return false
}
