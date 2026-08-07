package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/gorm"
)

func replacePriceIncludedProducts(
	ctx context.Context,
	priceID string,
	products []IncludedProductRequest,
) error {
	db := services.GetDB().WithContext(ctx)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("price_id = ?", priceID).
			Delete(&models.PriceIncludedProduct{}).
			Error; err != nil {
			return err
		}

		if len(products) == 0 {
			return nil
		}

		links := make([]models.PriceIncludedProduct, 0, len(products))

		for index, product := range products {
			if product.ProductID == "" || product.Quantity < 1 {
				continue
			}

			links = append(links, models.PriceIncludedProduct{
				PriceID:   priceID,
				ProductID: product.ProductID,
				Quantity:  product.Quantity,
				SortOrder: index,
			})
		}

		if len(links) == 0 {
			return nil
		}

		return tx.Create(&links).Error
	})
}

// GetProductPrices returns all prices for a specific product
func GetProductPrices(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

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

	var prices []models.PriceInfo

	err = services.GetDB().
		WithContext(r.Context()).
		Where("stripe_product_id = ? AND active = TRUE", productID).
		Order("unit_amount ASC, id ASC").
		Find(&prices).
		Error

	if err != nil {
		http.Error(w, "Failed to fetch prices", http.StatusInternalServerError)
		return
	}

	if prices == nil {
		prices = []models.PriceInfo{}
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"prices": prices,
	})

}

// CreateCheckoutSessionWithPrice creates a checkout session for a specific price
func CreateCheckoutSessionWithPrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PriceID  string            `json:"price_id"`
		Quantity int64             `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	baseUrl := os.Getenv("BASE_URL")

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
		SuccessURL: stripe.String(fmt.Sprintf("%s/success?session_id={CHECKOUT_SESSION_ID}", baseUrl)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/cancel", baseUrl)),
	}

	session, err := session.New(params)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"session_id": session.ID,
	}

	json.NewEncoder(w).Encode(response)
}

func UpdateTierIncludedProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["eventId"]
	priceID := vars["priceId"]

	var req struct {
		IncludedProducts []IncludedProductRequest `json:"included_products"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	var tier models.PriceInfo

	err = services.GetDB().
		WithContext(r.Context()).
		Where(
			"id = ? AND stripe_product_id = ? AND active = TRUE",
			priceID,
			event.StripeProductID,
		).
		First(&tier).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Tier not found for event", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve tier", http.StatusInternalServerError)
		return
	}

	if err := replacePriceIncludedProducts(
		r.Context(),
		tier.ID,
		req.IncludedProducts,
	); err != nil {
		log.Printf("Unable to update tier products: %v", err)
		http.Error(
			w,
			"Failed to update tier products",
			http.StatusInternalServerError,
		)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"price_id": tier.ID,
	})
}
