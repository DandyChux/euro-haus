package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

type CreateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       int64             `json:"price"` // in cents
	Currency    string            `json:"currency"`
	Images      []string          `json:"images"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateProductResponse struct {
	Success   bool   `json:"success"`
	ProductID string `json:"product_id"`
	PriceID   string `json:"price_id"`
	Message   string `json:"message"`
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Simple auth check
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "eurohaus2024"
	}

	// Check authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		log.Printf("Missing or invalid authorization header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := authHeader[7:]
	if !VerifyAuth(token) {
		log.Printf("Invalid token: %s", token)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Price <= 0 {
		http.Error(w, "Name and price are required", http.StatusBadRequest)
		return
	}

	// Create product in Stripe
	productParams := &stripe.ProductParams{
		Name:     stripe.String(req.Name),
		Active:   stripe.Bool(true),
		Metadata: req.Metadata,
	}

	if req.Description != "" {
		productParams.Description = stripe.String(req.Description)
	}

	if len(req.Images) > 0 {
		productParams.Images = stripe.StringSlice(req.Images)
	}

	// Create the product
	newProduct, err := product.New(productParams)
	if err != nil {
		log.Printf("Failed to create product: %v", err)
		http.Error(w, "Failed to create product: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create price for the product
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(newProduct.ID),
		UnitAmount: stripe.Int64(req.Price),
		Currency:   stripe.String(req.Currency),
	}

	newPrice, err := price.New(priceParams)
	if err != nil {
		// Delete the product if price creation fails
		log.Printf("Failed to create price, deleting product %s: %v", newProduct.ID, err)
		_, delErr := product.Del(newProduct.ID, nil)
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
	updatedProduct, err := product.Update(newProduct.ID, updateParams)
	if err != nil {
		log.Printf("Warning: Failed to set default price for product %s: %v", newProduct.ID, err)
		// Continue anyway, the product and price were created successfully
	}

	log.Printf("Successfully created product %s with price %s", newProduct.ID, newPrice.ID)

	// Return success response
	response := CreateProductResponse{
		Success:   true,
		ProductID: updatedProduct.ID,
		PriceID:   newPrice.ID,
		Message:   "Product created successfully",
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       int64             `json:"price"` // in cents
	Currency    string            `json:"currency"`
	Images      []string          `json:"images"`
	Metadata    map[string]string `json:"metadata"`
}

// UpdateProduct handles updating an existing Stripe product
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		log.Printf("Missing or invalid authorization header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := authHeader[7:]
	if !VerifyAuth(token) {
		log.Printf("Invalid token: %s", token)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get product ID from URL
	vars := mux.Vars(r)
	productID := vars["id"]
	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req UpdateProductRequest
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

	// Update product in Stripe
	productParams := &stripe.ProductParams{
		Name:     stripe.String(req.Name),
		Metadata: req.Metadata,
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

	// If price has changed, create a new price and update default price
	if req.Price > 0 && req.Currency != "" {
		// Check if the price has actually changed
		needNewPrice := true
		if existingProduct.DefaultPrice != nil {
			if existingProduct.DefaultPrice.UnitAmount == req.Price &&
				existingProduct.DefaultPrice.Currency == stripe.Currency(req.Currency) {
				needNewPrice = false
			}
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		log.Printf("Missing or invalid authorization header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := authHeader[7:]
	if !VerifyAuth(token) {
		log.Printf("Invalid token: %s", token)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
