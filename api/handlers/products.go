package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/product"
)

type ProductResponse struct {
	Products []EnrichedProduct `json:"products"`
}

type EnrichedProduct struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  *string           `json:"description"`
	Images       []string          `json:"images"`
	Metadata     map[string]string `json:"metadata"`
	Active       bool              `json:"active"`
	DefaultPrice *PriceInfo        `json:"default_price"`
	Created      int64             `json:"created"`
	Updated      int64             `json:"updated"`
}

type PriceInfo struct {
	ID         string `json:"id"`
	UnitAmount int64  `json:"unit_amount"`
	Currency   string `json:"currency"`
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if we should include inactive products
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	// Fetch products from Stripe
	params := &stripe.ProductListParams{
		Expand: []*string{stripe.String("data.default_price")},
	}

	// Only filter by active status if we're not including inactive products
	if !includeInactive {
		params.Active = stripe.Bool(true)
	}

	params.Filters.AddFilter("limit", "", "100")

	iter := product.List(params)
	var enrichedProducts []EnrichedProduct

	for iter.Next() {
		p := iter.Product()

		var priceInfo *PriceInfo
		if p.DefaultPrice != nil {
			priceInfo = &PriceInfo{
				ID:         p.DefaultPrice.ID,
				UnitAmount: p.DefaultPrice.UnitAmount,
				Currency:   string(p.DefaultPrice.Currency),
			}
		}

		enrichedProduct := EnrichedProduct{
			ID:           p.ID,
			Name:         p.Name,
			Description:  &p.Description,
			Images:       p.Images,
			Metadata:     p.Metadata,
			Active:       p.Active,
			DefaultPrice: priceInfo,
			Created:      p.Created,
			Updated:      p.Updated,
		}

		enrichedProducts = append(enrichedProducts, enrichedProduct)
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error fetching products: %v", err)
		http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, even if empty
	if enrichedProducts == nil {
		enrichedProducts = []EnrichedProduct{}
	}

	response := ProductResponse{
		Products: enrichedProducts,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GetProduct returns a single product by ID
func GetProduct(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get product ID from URL path
	vars := mux.Vars(r)
	productID := vars["id"]

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	// Fetch product from Stripe
	params := &stripe.ProductParams{}
	params.AddExpand("default_price")

	p, err := product.Get(productID, params)
	if err != nil {
		log.Printf("Error fetching product %s: %v", productID, err)
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch product", http.StatusInternalServerError)
		return
	}

	// Check if product is active
	if !p.Active {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Build price info
	var priceInfo *PriceInfo
	if p.DefaultPrice != nil {
		priceInfo = &PriceInfo{
			ID:         p.DefaultPrice.ID,
			UnitAmount: p.DefaultPrice.UnitAmount,
			Currency:   string(p.DefaultPrice.Currency),
		}
	}

	// Build enriched product
	enrichedProduct := EnrichedProduct{
		ID:           p.ID,
		Name:         p.Name,
		Description:  &p.Description,
		Images:       p.Images,
		Metadata:     p.Metadata,
		Active:       p.Active,
		DefaultPrice: priceInfo,
		Created:      p.Created,
		Updated:      p.Updated,
	}

	// Return single product
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(enrichedProduct); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
