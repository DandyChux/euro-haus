package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/gorm"
)

type ProductWriteRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Images      []string `json:"images"`

	Type     string `json:"type"`
	Currency string `json:"currency"`

	Price          int64  `json:"price"`
	CompareAtPrice *int64 `json:"compare_at_price"`

	IsNew    bool `json:"is_new"`
	InStock bool `json:"in_stock"`
	Active bool `json:"active"`
	Featured bool `json:"featured"`

	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`

	Tags        models.ProductStringList `json:"tags"`
	MaxQuantity *int              `json:"max_quantity"`
}

// CreateProductResponse represents the response body for creating a product
type CreateProductResponse struct {
	Success   bool   `json:"success"`
	ProductID string `json:"product_id"`
	PriceID   string `json:"price_id"`
	Message   string `json:"message"`
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req ProductWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	productParams := &stripe.ProductParams{
		Name:   stripe.String(req.Name),
		Active: stripe.Bool(req.Active),
	}

	if req.Description != "" {
		productParams.Description = stripe.String(req.Description)
	}

	if len(req.Images) > 0 {
		productParams.Images = stripe.StringSlice(req.Images)
	}

	stripeProduct, err := product.New(productParams)
	if err != nil {
		log.Printf("Failed to create product: %v", err)
		http.Error(w, "Failed to create product: "+err.Error(), http.StatusInternalServerError)
		return
	}

	localProduct := models.Product{
		ID:          stripeProduct.ID,
		Title:       req.Name,
		Description: req.Description,
		Type:        req.Type,
		Images:      models.ProductStringList(req.Images),

		Price:          req.Price,
		Currency:       req.Currency,
		CompareAtPrice: req.CompareAtPrice,

		IsNew:    req.IsNew,
		InStock: req.InStock,
		Active: req.Active,
		Featured: req.Featured,

		Category:    req.Category,
		Subcategory: req.Subcategory,
		Tags:        req.Tags,

		MaxQuantity: req.MaxQuantity,
	}

	if err := services.GetDB().
		WithContext(r.Context()).
		Create(&localProduct).
		Error; err != nil {
		// Product was created remotely but not locally.
		// Clean it up because this is still pre-production.
		if _, deleteErr := product.Del(stripeProduct.ID, nil); deleteErr != nil {
			log.Printf(
				"Failed to clean up Stripe product %s: %v",
				stripeProduct.ID,
				deleteErr,
			)
		}

		http.Error(
			w,
			"Failed to save product to database",
			http.StatusInternalServerError,
		)
		return
	}

	response := CreateProductResponse{
		Success:   true,
		ProductID: stripeProduct.ID,
		Message:   "Product created successfully",
	}

	// If Price is provided, create a price and set it as default
	if req.Price > 0 {
		if len(req.Currency) != 3 {
			http.Error(
				w,
				"Currency must be a three-letter ISO currency code",
				http.StatusBadRequest,
			)
			return
		}

		req.Currency = strings.ToLower(req.Currency)
		priceParams := &stripe.PriceParams{
			Product:    stripe.String(stripeProduct.ID),
			UnitAmount: stripe.Int64(req.Price),
			Currency:   &req.Currency,
		}

		newPrice, err := price.New(priceParams)
		if err != nil {
			// Delete the product if price creation fails
			log.Printf("Failed to create price, deleting product %s: %v", stripeProduct.ID, err)
			_, delErr := product.Del(stripeProduct.ID, nil)
			if delErr != nil {
				log.Printf("Failed to delete product after price creation failure: %v", delErr)
			}
			http.Error(w, "Failed to create price: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update product with default price
		updateParams := &stripe.ProductParams{
			DefaultPrice: stripe.String(newPrice.ID),
		}
		_, err = product.Update(stripeProduct.ID, updateParams)
		if err != nil {
			log.Printf("Warning: Failed to set default price for product %s: %v", stripeProduct.ID, err)
			// Continue anyway, the product and price were created successfully
		}

		response.PriceID = newPrice.ID
		log.Printf("Successfully created product %s with price %s", stripeProduct.ID, newPrice.ID)
	} else {
		log.Printf("Successfully created product %s without a default price", stripeProduct.ID)
	}

	if err := services.SyncStripeProductPrices(
		r.Context(),
		stripeProduct.ID,
	); err != nil {
		log.Printf(
			"Warning: failed to sync prices for product %s: %v",
			stripeProduct.ID,
			err,
		)
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateProduct handles updating an existing Stripe product
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get product ID from URL
	vars := mux.Vars(r)
	productID := vars["id"]
	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req ProductWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing product to check if it exists
	existingProduct, err := product.Get(productID, nil)
	if err != nil {
		log.Printf("Product not found: %v", err)
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	productParams := &stripe.ProductParams{
		Name: stripe.String(req.Name),
	}

	if req.Description != "" {
		productParams.Description = stripe.String(req.Description)
	}

	if len(req.Images) > 0 {
		productParams.Images = stripe.StringSlice(req.Images)
	}

	// Update the product
	updatedProduct, err := product.Update(productID, productParams)
	if err != nil {
		log.Printf("Failed to update product: %v", err)
		http.Error(w, "Failed to update product: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = services.GetDB().
		WithContext(r.Context()).
		Model(&models.Product{}).
		Where("id = ?", productID).
		Updates(map[string]interface{}{
			"title":            req.Name,
			"description":     req.Description,
			"images":          req.Images,
			"type":             req.Type,
			"price":            req.Price,
			"currency":         req.Currency,
			"compare_at_price": req.CompareAtPrice,
			"is_new":           req.IsNew,
			"in_stock":         req.InStock,
			"featured":         req.Featured,
			"category":         req.Category,
			"subcategory":      req.Subcategory,
			"tags":             req.Tags,
			"max_quantity":     req.MaxQuantity,
		}).
		Error

	if err != nil {
		http.Error(
			w,
			"Failed to update local product",
			http.StatusInternalServerError,
		)
		return
	}

	// If price has changed, create a new price and update default price
	if req.Price > 0 && req.Currency != "" {
		// Check if the price has actually changed
		needNewPrice := true
		if existingProduct.DefaultPrice != nil &&
			existingProduct.DefaultPrice.UnitAmount == req.Price &&
			string(existingProduct.DefaultPrice.Currency) ==
				strings.ToLower(req.Currency) {
			needNewPrice = false
		}

		if needNewPrice {
			// Create new price
			priceParams := &stripe.PriceParams{
				Product:    stripe.String(productID),
				UnitAmount: stripe.Int64(req.Price),
				Currency:   stripe.String(req.Currency),
			}

			newPrice, err := price.New(priceParams)
			if err != nil {
				log.Printf("Failed to create new price: %v", err)
				http.Error(w, "Failed to create new price: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Update product with new default price
			updateParams := &stripe.ProductParams{
				DefaultPrice: stripe.String(newPrice.ID),
			}
			updatedProduct, err = product.Update(productID, updateParams)
			if err != nil {
				log.Printf("Warning: Failed to set default price for product %s: %v", productID, err)
			}

			// Archive old price if it exists
			if existingProduct.DefaultPrice != nil && existingProduct.DefaultPrice.ID != "" {
				archiveParams := &stripe.PriceParams{
					Active: stripe.Bool(false),
				}
				_, err = price.Update(existingProduct.DefaultPrice.ID, archiveParams)
				if err != nil {
					log.Printf("Warning: Failed to archive old price %s: %v", existingProduct.DefaultPrice.ID, err)
				}
			}
		}
	}

	log.Printf("Successfully updated product %s", productID)

	if err := services.SyncStripeProductPrices(
		r.Context(),
		productID,
	); err != nil {
		log.Printf(
			"Warning: failed to sync prices for product %s: %v",
			productID,
			err,
		)
	}

	// Return success response
	response := map[string]interface{}{
		"success":   true,
		"productID": updatedProduct.ID,
		"message":   "Product updated successfully",
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// DeleteProduct handles deleting a Stripe product
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	// Get product ID from URL
	vars := mux.Vars(r)
	productID := vars["id"]
	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	// Check if product exists
	_, err := product.Get(productID, nil)
	if err != nil {
		log.Printf("Product not found: %v", err)
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Archive all prices associated with this product first
	// This prevents the product from being purchased
	priceListParams := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}
	priceListParams.AddExpand("data.product")

	iter := price.List(priceListParams)
	for iter.Next() {
		p := iter.Price()
		archiveParams := &stripe.PriceParams{
			Active: stripe.Bool(false),
		}
		_, err := price.Update(p.ID, archiveParams)
		if err != nil {
			log.Printf("Warning: Failed to archive price %s: %v", p.ID, err)
		}
	}

	// Archive the product instead of deleting it
	// Stripe doesn't allow deletion of products that have been used in orders
	archiveParams := &stripe.ProductParams{
		Active: stripe.Bool(false),
	}

	archivedProduct, err := product.Update(productID, archiveParams)
	if err != nil {
		log.Printf("Failed to archive product: %v", err)
		http.Error(w, "Failed to archive product: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully archived product %s", productID)

	// Return success response
	response := map[string]interface{}{
		"success":   true,
		"message":   "Product deleted successfully",
		"productID": archivedProduct.ID,
		"archived":  true,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

type CreatePriceRequest struct {
	Product           string   `json:"product"`
	UnitAmount        int64    `json:"unit_amount"`
	Currency          string   `json:"currency"`
	Nickname          string   `json:"nickname"`
	Description       string   `json:"description"`
	Features          []string `json:"features"`
	Default           bool     `json:"default"`
	MostPopular       bool     `json:"most_popular"`
	RequiresApproval  bool     `json:"requires_approval"`
	RequiresSubmission bool    `json:"requires_submission"`
	Quantity          int      `json:"quantity"`
	StockQuantity     *int     `json:"stock_quantity"`
	Size              string   `json:"size"`
	Color             string   `json:"color"`
}

// CreatePrice creates a new price for a product
func CreatePrice(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers

	var req CreatePriceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(req.Product),
		UnitAmount: stripe.Int64(req.UnitAmount),
		Currency:   stripe.String(req.Currency),
		Nickname:   stripe.String(req.Nickname),
	}

	newPrice, err := price.New(params)
	if err != nil {
		log.Printf("Error creating price: %v", err)
		http.Error(w, "Failed to create price", http.StatusInternalServerError)
		return
	}

	featuresJSON, err := json.Marshal(req.Features)
	if err != nil {
		http.Error(w, "Invalid features", http.StatusBadRequest)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	priceInfo := &models.PriceInfo{
		ID:                 newPrice.ID,
		StripeProductID:   req.Product,
		UnitAmount:         newPrice.UnitAmount,
		Currency:           string(newPrice.Currency),
		Nickname:           req.Nickname,
		Description:        req.Description,
		Active:             true,
		Features:           featuresJSON,
		IsDefault:          req.Default,
		IsMostPopular:      req.MostPopular,
		RequiresApproval:   req.RequiresApproval,
		RequiresSubmission: req.RequiresSubmission,
		Quantity:           req.Quantity,
		StockQuantity:      req.StockQuantity,
		Size:               req.Size,
		Color:              req.Color,
	}

	if err := db.Create(priceInfo).Error; err != nil {
		http.Error(
			w,
			"Failed to update local price data",
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"id":          newPrice.ID,
		"product":     req.Product,
		"unit_amount": newPrice.UnitAmount,
		"currency": req.Currency,
		"nickname":    newPrice.Nickname,
		"description":    req.Description,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

type UpdatePriceRequest struct {
	ID                 string   `json:"id"`
	Nickname           string   `json:"nickname"`
	Description        string   `json:"description"`
	Features           []string `json:"features"`
	MostPopular        bool     `json:"most_popular"`
	RequiresApproval   bool     `json:"requires_approval"`
	RequiresSubmission bool     `json:"requires_submission"`
	StockQuantity      *int     `json:"stock_quantity"`
}

// UpdatePrice updates the metadata and nickname of a price
func UpdatePrice(w http.ResponseWriter, r *http.Request) {
	priceID := mux.Vars(r)["id"]

	if priceID == "" {
		http.Error(w, "Price ID is required", http.StatusBadRequest)
		return
	}

	var req UpdatePriceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	featuresJSON, err := json.Marshal(req.Features)
	if err != nil {
		http.Error(w, "Invalid features", http.StatusBadRequest)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	var existingPrice models.PriceInfo

	if err := db.
		Where("id = ?", priceID).
		First(&existingPrice).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price not found", http.StatusNotFound)
			return
		}

		http.Error(
			w,
			"Failed to load price",
			http.StatusInternalServerError,
		)
		return
	}

	// Stripe owns the nickname. Application metadata remains in PostgreSQL.
	updatedStripePrice, err := price.Update(
		priceID,
		&stripe.PriceParams{
			Nickname: stripe.String(req.Nickname),
		},
	)
	if err != nil {
		log.Printf("Error updating Stripe price %s: %v", priceID, err)

		http.Error(
			w,
			"Failed to update Stripe price",
			http.StatusInternalServerError,
		)
		return
	}

	err = db.
		Model(&models.PriceInfo{}).
		Where("id = ?", priceID).
		Updates(map[string]interface{}{
			"nickname":            req.Nickname,
			"description":         req.Description,
			"features":            featuresJSON,
			"is_most_popular":     req.MostPopular,
			"requires_approval":   req.RequiresApproval,
			"requires_submission": req.RequiresSubmission,
			"stock_quantity":      req.StockQuantity,
		}).
		Error

	if err != nil {
		http.Error(
			w,
			"Failed to update local price data",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"id":          updatedStripePrice.ID,
		"nickname":    req.Nickname,
		"description": req.Description,
		"features":    req.Features,
	})
}

// ArchivePrice archives a price (sets active to false)
func ArchivePrice(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	priceID := vars["id"]

	// Get the price to check if it's a default price
	priceObj, err := price.Get(priceID, nil)
	if err != nil {
		log.Printf("Price not found: %v", err)
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	// Get the product to check if this is the default price
	productObj, err := product.Get(priceObj.Product.ID, nil)
	if err != nil {
		log.Printf("Product not found: %v", err)
		http.Error(w, "Product not found", http.StatusInternalServerError)
		return
	}

	// Check if this is the default price
	if productObj.DefaultPrice != nil && productObj.DefaultPrice.ID == priceID {
		http.Error(w, "Cannot archive the default price. Please set another price as default first.", http.StatusBadRequest)
		return
	}

	params := &stripe.PriceParams{
		Active: stripe.Bool(false),
	}

	_, err = price.Update(priceID, params)
	if err != nil {
		log.Printf("Error archiving price: %v", err)
		http.Error(w, "Failed to archive price", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// SetDefaultPrice sets a price as the default for a product
func SetDefaultPrice(w http.ResponseWriter, r *http.Request) {

	var req struct {
		ProductID string `json:"productId"`
		PriceID   string `json:"priceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify the price exists - expand the product field
	params := &stripe.PriceParams{}
	params.AddExpand("product")
	priceObj, err := price.Get(req.PriceID, params)
	if err != nil {
		log.Printf("Price not found: %v", err)
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	// Check if the price belongs to the product
	if priceObj.Product == nil || priceObj.Product.ID != req.ProductID {
		log.Printf("Price product mismatch: price product=%v, requested product=%s", priceObj.Product, req.ProductID)
		http.Error(w, "Price does not belong to this product", http.StatusBadRequest)
		return
	}

	// Update the product's default price
	productParams := &stripe.ProductParams{
		DefaultPrice: stripe.String(req.PriceID),
	}

	updatedProduct, err := product.Update(req.ProductID, productParams)
	if err != nil {
		log.Printf("Error setting default price: %v", err)
		http.Error(w, "Failed to set default price", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully set default price %s for product %s", req.PriceID, req.ProductID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"defaultPrice": updatedProduct.DefaultPrice.ID,
	})
}
