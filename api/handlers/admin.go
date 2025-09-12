package handlers

import (
	"encoding/json"
	"euro-haus-api/services"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       int64             `json:"price"` // in cents
	Currency    string            `json:"currency"`
	Images      []string          `json:"images"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateProductResponse represents the response body for creating a product
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

	// Process metadata to handle large fields
	processedMetadata, err := ProcessLargeMetadata(req.Metadata)
	if err != nil {
		log.Printf("Warning: Error processing metadata: %v", err)
		// Continue with original metadata if processing fails
		processedMetadata = req.Metadata
	}

	// Create product in Stripe
	productParams := &stripe.ProductParams{
		Name:     stripe.String(req.Name),
		Active:   stripe.Bool(true),
		Metadata: processedMetadata,
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
		updatedProduct = newProduct
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

	// Process metadata to handle large fields
	processedMetadata, err := ProcessLargeMetadata(req.Metadata, existingProduct.Metadata)
	if err != nil {
		log.Printf("Warning: Error processing metadata: %v", err)
		// Continue with original metadata if processing fails
		processedMetadata = req.Metadata
	}

	// Update product in Stripe
	productParams := &stripe.ProductParams{
		Name:     stripe.String(req.Name),
		Metadata: processedMetadata,
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

// CreatePrice creates a new price for a product
func CreatePrice(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		Product    string            `json:"product"`
		UnitAmount int64             `json:"unit_amount"`
		Currency   string            `json:"currency"`
		Nickname   string            `json:"nickname"`
		Metadata   map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process metadata for boolean fields
	finalMetadata := req.Metadata
	if finalMetadata == nil {
		finalMetadata = make(map[string]string)
	}

	// Ensure boolean values are stored as strings
	for k, v := range req.Metadata {
		if k == "requires_vehicle_submission" || k == "is_most_popular" || k == "requires_approval" {
			// Convert any value to proper "true" or "false" string
			boolValue, err := strconv.ParseBool(v)
			if err == nil {
				if boolValue {
					finalMetadata[k] = "true"
				} else {
					finalMetadata[k] = "false"
				}
			} else {
				// Default values
				if k == "requires_approval" {
					finalMetadata[k] = "true" // Default to requiring approval
				} else {
					finalMetadata[k] = "false"
				}
			}
		} else {
			finalMetadata[k] = v
		}
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(req.Product),
		UnitAmount: stripe.Int64(req.UnitAmount),
		Currency:   stripe.String(req.Currency),
		Nickname:   stripe.String(req.Nickname),
		Metadata:   req.Metadata,
	}

	newPrice, err := price.New(params)
	if err != nil {
		log.Printf("Error creating price: %v", err)
		http.Error(w, "Failed to create price", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":          newPrice.ID,
		"product":     newPrice.Product.ID,
		"unit_amount": newPrice.UnitAmount,
		"nickname":    newPrice.Nickname,
		"metadata":    newPrice.Metadata,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdatePrice updates the metadata and nickname of a price
func UpdatePrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get price ID from URL
	vars := mux.Vars(r)
	priceID := vars["id"]

	var req struct {
		Nickname string            `json:"nickname"`
		Metadata map[string]string `json:"metadata"`
		Active   *bool             `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the existing price first to preserve metadata
	existingPrice, err := price.Get(priceID, nil)
	if err != nil {
		log.Printf("Price not found: %v", err)
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	// Merge metadata
	finalMetadata := existingPrice.Metadata
	if finalMetadata == nil {
		finalMetadata = make(map[string]string)
	}

	// Ensure boolean values are stored as strings
	for k, v := range req.Metadata {
		if k == "requires_vehicle_submission" || k == "is_most_popular" || k == "requires_approval" {
			// Convert any value to proper "true" or "false" string
			boolValue, err := strconv.ParseBool(v)
			if err == nil {
				if boolValue {
					finalMetadata[k] = "true"
				} else {
					finalMetadata[k] = "false"
				}
			} else {
				// Default values
				if k == "requires_approval" {
					finalMetadata[k] = "true" // Default to requiring approval
				} else {
					finalMetadata[k] = "false"
				}
			}
		} else {
			finalMetadata[k] = v
		}
	}

	params := &stripe.PriceParams{
		Metadata: finalMetadata,
	}

	// Only update nickname if provided
	if req.Nickname != "" {
		params.Nickname = stripe.String(req.Nickname)
	}

	updatedPrice, err := price.Update(priceID, params)
	if err != nil {
		log.Printf("Error updating price %s: %v", priceID, err)
		http.Error(w, "Failed to update price: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated price %s with nickname: %s", priceID, updatedPrice.Nickname)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"nickname": updatedPrice.Nickname,
		"metadata": updatedPrice.Metadata,
	})
}

// ArchivePrice archives a price (sets active to false)
func ArchivePrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	productID := vars["id"]

	var req struct {
		PriceID string `json:"priceId"`
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
	if priceObj.Product == nil || priceObj.Product.ID != productID {
		log.Printf("Price product mismatch: price product=%v, requested product=%s", priceObj.Product, productID)
		http.Error(w, "Price does not belong to this product", http.StatusBadRequest)
		return
	}

	// Update the product's default price
	productParams := &stripe.ProductParams{
		DefaultPrice: stripe.String(req.PriceID),
	}

	updatedProduct, err := product.Update(productID, productParams)
	if err != nil {
		log.Printf("Error setting default price: %v", err)
		http.Error(w, "Failed to set default price", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully set default price %s for product %s", req.PriceID, productID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"defaultPrice": updatedProduct.DefaultPrice.ID,
	})
}

// ProcessLargeMetadata handles metadata fields that might exceed Stripe's limits
func ProcessLargeMetadata(metadata map[string]string, existingMetadata map[string]string) (map[string]string, error) {
	processedMetadata := make(map[string]string)

	// List of fields that might be large and should be stored externally
	largeFields := []string{"sponsors", "sponsor_tiers", "agenda", "includes"}

	for key, value := range metadata {
		// Check if this is a potentially large field
		isLargeField := false
		for _, field := range largeFields {
			if key == field {
				isLargeField = true
				break
			}
		}

		// If it's a large field and exceeds 400 chars (leaving buffer), store externally
		if isLargeField && len(value) > 400 {
			// Check if this field already has an external URL
			existingURL := ""
			if existingMetadata != nil {
				if url, exists := existingMetadata[key+"_url"]; exists {
					existingURL = url
				}
			}

			// If we have an existing URL, update that file instead of creating a new one
			if existingURL != "" {
				// Extract the key from the existing URL
				// URL format: https://bucket.endpoint/folder/filename.json
				urlParts := strings.Split(existingURL, "/")
				if len(urlParts) > 0 {
					filename := urlParts[len(urlParts)-1]
					folder := "product-metadata"
					if len(urlParts) > 1 {
						folder = urlParts[len(urlParts)-2]
					}

					// Parse and re-upload to the same location
					var jsonData interface{}
					if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
						jsonData = value
					}

					// Upload to the same filename
					jsonURL, err := services.UploadJSON(jsonData, strings.TrimSuffix(filename, ".json"), folder)
					if err != nil {
						log.Printf("Warning: Failed to update external metadata field %s: %v", key, err)
						processedMetadata[key] = value
					} else {
						processedMetadata[key+"_url"] = jsonURL
						processedMetadata[key+"_external"] = "true"
						if len(value) > 100 {
							processedMetadata[key+"_preview"] = value[:100] + "..."
						}
					}
				}
			} else {
				// No existing URL, create a new one
				metadataID := fmt.Sprintf("%s-%s", key, uuid.New().String()[:8])

				var jsonData interface{}
				if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
					jsonData = value
				}

				jsonURL, err := services.UploadJSON(jsonData, metadataID, "product-metadata")
				if err != nil {
					log.Printf("Warning: Failed to upload large metadata field %s: %v", key, err)
					processedMetadata[key] = value
				} else {
					processedMetadata[key+"_url"] = jsonURL
					processedMetadata[key+"_external"] = "true"
					if len(value) > 100 {
						processedMetadata[key+"_preview"] = value[:100] + "..."
					}
				}
			}
		} else {
			// Small enough to store directly
			processedMetadata[key] = value
		}
	}

	return processedMetadata, nil
}

// RetrieveLargeMetadata fetches externally stored metadata fields
func RetrieveLargeMetadata(metadata map[string]string) map[string]string {
	processedMetadata := make(map[string]string)

	for key, value := range metadata {
		// Check if this field is stored externally
		if strings.HasSuffix(key, "_external") && value == "true" {
			fieldName := strings.TrimSuffix(key, "_external")
			if urlKey := fieldName + "_url"; metadata[urlKey] != "" {
				// For now, just pass the URL through - the client can fetch it
				// In a production system, you might want to fetch and cache the data
				processedMetadata[fieldName] = metadata[urlKey]
			}
		} else if !strings.HasSuffix(key, "_url") && !strings.HasSuffix(key, "_preview") && !strings.HasSuffix(key, "_external") && !strings.HasSuffix(key, "_truncated") {
			// Regular metadata field
			processedMetadata[key] = value
		}
	}

	return processedMetadata
}
