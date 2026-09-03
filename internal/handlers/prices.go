package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
	db := services.GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}

	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return errors.New("price ID is required")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleteResult := tx.
			Where("price_id = ?", priceID).
			Delete(&models.PriceIncludedProduct{})

		if deleteResult.Error != nil {
			return fmt.Errorf(
				"delete existing included products: %w",
				deleteResult.Error,
			)
		}

		links := make([]models.PriceIncludedProduct, 0, len(products))

		for index, includedProduct := range products {
			productID := strings.TrimSpace(includedProduct.ProductID)

			if productID == "" {
				return fmt.Errorf(
					"included product at index %d has no product ID",
					index,
				)
			}

			if includedProduct.Quantity < 1 {
				return fmt.Errorf(
					"included product %s must have a quantity greater than zero",
					productID,
				)
			}

			links = append(links, models.PriceIncludedProduct{
				PriceID:   priceID,
				ProductID: productID,
				Quantity:  includedProduct.Quantity,
				SortOrder: index,
			})
		}

		if len(links) == 0 {
			log.Printf(
				"No included products supplied for price %s; existing rows removed",
				priceID,
			)
			return nil
		}

		log.Printf(
			"Inserting included product rows for price %s: %+v",
			priceID,
			links,
		)

		insertResult := tx.Create(&links)
		if insertResult.Error != nil {
			return fmt.Errorf(
				"insert included products: %w",
				insertResult.Error,
			)
		}

		if insertResult.RowsAffected != int64(len(links)) {
			return fmt.Errorf(
				"expected to insert %d included products, inserted %d",
				len(links),
				insertResult.RowsAffected,
			)
		}

		var persisted []models.PriceIncludedProduct

		verifyResult := tx.
			Where("price_id = ?", priceID).
			Order("sort_order ASC, product_id ASC").
			Find(&persisted)

		if verifyResult.Error != nil {
			return fmt.Errorf(
				"verify inserted included products: %w",
				verifyResult.Error,
			)
		}

		if len(persisted) != len(links) {
			return fmt.Errorf(
				"expected to verify %d included products, found %d",
				len(links),
				len(persisted),
			)
		}

		log.Printf(
			"Persisted %d included products for price %s: %+v",
			len(persisted),
			priceID,
			persisted,
		)

		return nil
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
	eventID := strings.TrimSpace(vars["eventId"])
	priceID := strings.TrimSpace(vars["priceId"])

	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	if priceID == "" {
		http.Error(w, "Price ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		IncludedProducts []IncludedProductRequest `json:"included_products"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf(
		"Updating included products: event_id=%s price_id=%s products=%+v",
		eventID,
		priceID,
		req.IncludedProducts,
	)

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf(
			"Failed to find event %s: %v",
			eventID,
			err,
		)
		http.Error(
			w,
			"Unable to retrieve event",
			http.StatusInternalServerError,
		)
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
		log.Printf(
			"Price %s was not found for event %s with Stripe product %s",
			priceID,
			eventID,
			event.StripeProductID,
		)
		http.Error(w, "Tier not found for event", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf(
			"Failed to find price %s for event %s: %v",
			priceID,
			eventID,
			err,
		)
		http.Error(
			w,
			"Unable to retrieve tier",
			http.StatusInternalServerError,
		)
		return
	}

	if err := replacePriceIncludedProducts(
		r.Context(),
		tier.ID,
		req.IncludedProducts,
	); err != nil {
		log.Printf(
			"Failed to persist included products for price %s: %v",
			tier.ID,
			err,
		)
		http.Error(
			w,
			"Failed to update tier products",
			http.StatusInternalServerError,
		)
		return
	}

	var savedProducts []models.PriceIncludedProduct

	if err := services.GetDB().
		WithContext(r.Context()).
		Where("price_id = ?", tier.ID).
		Order("sort_order ASC, product_id ASC").
		Find(&savedProducts).
		Error; err != nil {
		log.Printf(
			"Included products were saved but could not be reloaded for price %s: %v",
			tier.ID,
			err,
		)
		http.Error(
			w,
			"Included products were saved but could not be reloaded",
			http.StatusInternalServerError,
		)
		return
	}

	log.Printf(
		"Persisted %d included products for price %s",
		len(savedProducts),
		tier.ID,
	)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":           true,
		"price_id":          tier.ID,
		"included_products": savedProducts,
	}); err != nil {
		log.Printf(
			"Failed to encode included products response for price %s: %v",
			tier.ID,
			err,
		)
	}
}
