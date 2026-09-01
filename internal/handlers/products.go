package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/gorm"
)

type ProductResponse struct {
	Products []EnrichedProduct `json:"products"`
}

type EnrichedProduct struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	Type        string   `json:"type"`

	Price    int64  `json:"price"`
	Currency string `json:"currency"`

	CompareAtPrice *int64 `json:"compare_at_price"`

	IsNew    bool `json:"is_new"`
	InStock bool `json:"in_stock"`
	Featured bool `json:"featured"`

	Category    string   `json:"category"`
	Subcategory string   `json:"subcategory"`
	Tags        []string `json:"tags"`
	MaxQuantity *int     `json:"max_quantity"`

	Active       bool         `json:"active"`
	DefaultPrice *StripePrice `json:"default_price"`

	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

type StripePrice struct {
	ID         string `json:"id"`
	UnitAmount int64  `json:"unit_amount"`
	Currency   string `json:"currency"`
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
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

		var localProduct models.Product

		err := services.GetDB().
			WithContext(r.Context()).
			Preload("BundleItems").
			Where("id = ?", p.ID).
			First(&localProduct).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Do not reconstruct application fields from Stripe metadata.
			// Either skip the product or return it as an unclassified product.
			continue
		}

		if err != nil {
			http.Error(
				w,
				"Failed to load local product data",
				http.StatusInternalServerError,
			)
			return
		}

		enrichedProducts = append(
			enrichedProducts,
			productToResponse(localProduct, p),
		)
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
	var priceInfo *StripePrice
	if p.DefaultPrice != nil {
		priceInfo = &StripePrice{
			ID:         p.DefaultPrice.ID,
			UnitAmount: p.DefaultPrice.UnitAmount,
			Currency:   string(p.DefaultPrice.Currency),
		}
	}

	// Build enriched product
	enrichedProduct := EnrichedProduct{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Images:       p.Images,
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

type UpdateVariantStockRequest struct {
	Variants []VariantStockUpdate `json:"variants"`
}

type VariantStockUpdate struct {
	ID            string `json:"id"`
	StockQuantity *int   `json:"stock_quantity"`
}

func UpdateVariantStock(w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(r)["productId"]

	if productID == "" {
		http.Error(
			w,
			"Product ID is required",
			http.StatusBadRequest,
		)
		return
	}

	var req UpdateVariantStockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	for _, update := range req.Variants {
		if update.ID == "" {
			http.Error(
				w,
				"Variant price ID is required",
				http.StatusBadRequest,
			)
			return
		}

		var priceInfo models.PriceInfo

		err := db.
			Where(
				"id = ? AND stripe_product_id = ?",
				update.ID,
				productID,
			).
			First(&priceInfo).
			Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(
					w,
					"Variant does not belong to this product",
					http.StatusBadRequest,
				)
				return
			}

			http.Error(
				w,
				"Failed to load variant",
				http.StatusInternalServerError,
			)
			return
		}

		if update.StockQuantity != nil &&
			*update.StockQuantity < 0 {
			http.Error(
				w,
				"Stock quantity cannot be negative",
				http.StatusBadRequest,
			)
			return
		}

		if err := db.
			Model(&models.PriceInfo{}).
			Where("id = ?", update.ID).
			Updates(map[string]interface{}{
				"stock_quantity": update.StockQuantity,
				"active": update.StockQuantity == nil ||
					*update.StockQuantity > 0,
			}).
			Error; err != nil {
			http.Error(
				w,
				"Failed to update stock",
				http.StatusInternalServerError,
			)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

func GetProductVariants(w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(r)["productId"]

	var variants []models.PriceInfo

	err := services.GetDB().
		WithContext(r.Context()).
		Where(
			"stripe_product_id = ? AND active = TRUE",
			productID,
		).
		Order("unit_amount ASC, id ASC").
		Find(&variants).
		Error

	if err != nil {
		http.Error(
			w,
			"Failed to load product variants",
			http.StatusInternalServerError,
		)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"variants": variants,
	})
}

func productToResponse(
	local models.Product,
	stripeProduct *stripe.Product,
) EnrichedProduct {
	var defaultPrice *StripePrice

	if stripeProduct.DefaultPrice != nil {
		defaultPrice = &StripePrice{
			ID:         stripeProduct.DefaultPrice.ID,
			UnitAmount: stripeProduct.DefaultPrice.UnitAmount,
			Currency:   string(stripeProduct.DefaultPrice.Currency),
		}
	}

	return EnrichedProduct{
		ID:          local.ID,
		Name:        local.Title,
		Description: local.Description,
		Images:      local.Images,
		Type:        local.Type,

		Price:    local.Price,
		Currency: local.Currency,

		CompareAtPrice: local.CompareAtPrice,

		IsNew:    local.IsNew,
		InStock: local.InStock,
		Featured: local.Featured,

		Category:    local.Category,
		Subcategory: local.Subcategory,
		Tags:        local.Tags,
		MaxQuantity: local.MaxQuantity,

		Active:       stripeProduct.Active,
		DefaultPrice: defaultPrice,
		Created:      stripeProduct.Created,
		Updated:      stripeProduct.Updated,
	}
}

func loadProductPrices(
	ctx context.Context,
	product *models.Product,
) error {
	var prices []models.PriceInfo

	err := services.GetDB().
		WithContext(ctx).
		Preload("IncludedProductLinks").
		Where(
			"stripe_product_id = ?",
			product.ID,
		).
		Order("active DESC, unit_amount ASC, id ASC").
		Find(&prices).
		Error

	if err != nil {
		return err
	}

	if prices == nil {
		prices = []models.PriceInfo{}
	}

	product.Prices = prices

	return nil
}
