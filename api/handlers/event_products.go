package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// LinkProductsRequest represents a request to link products to an event
type LinkProductsRequest struct {
	EventID    string   `json:"eventId"`
	ProductIDs []string `json:"productIds"`
}

// TierProductsRequest represents products to include in a tier
type TierProductsRequest struct {
	PriceID  string                   `json:"priceId"`
	Products []IncludedProductRequest `json:"products"`
}

// IncludedProductRequest represents a product included in a tier
type IncludedProductRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// LinkProductsToEvent links products to an event (for add-ons/merchandise)
func LinkProductsToEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["eventId"]

	var req struct {
		ProductIDs []string `json:"productIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate event exists and is an event type
	eventParams := &stripe.ProductParams{}
	event, err := product.Get(eventID, eventParams)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event.Metadata["type"] != "event" {
		http.Error(w, "Product is not an event", http.StatusBadRequest)
		return
	}

	// Validate all products exist
	validProductIDs := []string{}
	for _, pid := range req.ProductIDs {
		prodParams := &stripe.ProductParams{}
		prod, err := product.Get(pid, prodParams)
		if err != nil {
			log.Printf("Product %s not found: %v", pid, err)
			continue
		}
		if prod.Active {
			validProductIDs = append(validProductIDs, pid)
		}
	}

	// Update event metadata with linked products
	updateParams := &stripe.ProductParams{}
	if event.Metadata == nil {
		updateParams.Metadata = map[string]string{}
	} else {
		// Copy existing metadata
		for k, v := range event.Metadata {
			updateParams.AddMetadata(k, v)
		}
	}

	updateParams.AddMetadata("linked_products", strings.Join(validProductIDs, ","))

	updatedEvent, err := product.Update(eventID, updateParams)
	if err != nil {
		log.Printf("Error updating event: %v", err)
		http.Error(w, "Failed to link products", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"eventId":        updatedEvent.ID,
		"linkedProducts": validProductIDs,
	})
}

// AddProductsToTier adds included products to a specific event tier
func AddProductsToTier(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req TierProductsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the price (tier)
	priceParams := &stripe.PriceParams{}
	tier, err := price.Get(req.PriceID, priceParams)
	if err != nil {
		http.Error(w, "Tier not found", http.StatusNotFound)
		return
	}

	// Build included products JSON
	includedProducts := []map[string]interface{}{}
	for _, p := range req.Products {
		// Validate product exists
		prodParams := &stripe.ProductParams{}
		prod, err := product.Get(p.ProductID, prodParams)
		if err != nil {
			log.Printf("Product %s not found: %v", p.ProductID, err)
			continue
		}

		includedProducts = append(includedProducts, map[string]interface{}{
			"id":       p.ProductID,
			"quantity": p.Quantity,
			"name":     prod.Name,
		})
	}

	// Convert to JSON string for metadata
	includedProductsJSON, err := json.Marshal(includedProducts)
	if err != nil {
		http.Error(w, "Failed to process products", http.StatusInternalServerError)
		return
	}

	// Update price metadata
	updateParams := &stripe.PriceParams{}
	if tier.Metadata == nil {
		updateParams.Metadata = map[string]string{}
	} else {
		// Copy existing metadata
		for k, v := range tier.Metadata {
			updateParams.AddMetadata(k, v)
		}
	}

	updateParams.AddMetadata("included_products", string(includedProductsJSON))

	updatedTier, err := price.Update(req.PriceID, updateParams)
	if err != nil {
		log.Printf("Error updating tier: %v", err)
		http.Error(w, "Failed to update tier", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"priceId":          updatedTier.ID,
		"includedProducts": includedProducts,
	})
}

// GetEventLinkedProducts retrieves all products linked to an event
func GetEventLinkedProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["eventId"]

	// Get the event
	eventParams := &stripe.ProductParams{}
	eventParams.AddExpand("default_price")

	event, err := product.Get(eventID, eventParams)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event.Metadata["type"] != "event" {
		http.Error(w, "Product is not an event", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"eventId":   event.ID,
		"eventName": event.Name,
	}

	// Get directly linked products (add-ons)
	linkedProducts := []map[string]interface{}{}
	if linkedProductIDs, ok := event.Metadata["linked_products"]; ok && linkedProductIDs != "" {
		productIDs := strings.Split(linkedProductIDs, ",")
		for _, pid := range productIDs {
			pid = strings.TrimSpace(pid)
			if pid == "" {
				continue
			}

			linkedParams := &stripe.ProductParams{}
			linkedParams.AddExpand("default_price")

			linkedProduct, err := product.Get(pid, linkedParams)
			if err != nil {
				continue
			}

			productInfo := map[string]interface{}{
				"id":          linkedProduct.ID,
				"name":        linkedProduct.Name,
				"description": linkedProduct.Description,
				"images":      linkedProduct.Images,
				"active":      linkedProduct.Active,
			}

			if linkedProduct.DefaultPrice != nil {
				productInfo["price"] = map[string]interface{}{
					"id":          linkedProduct.DefaultPrice.ID,
					"unit_amount": linkedProduct.DefaultPrice.UnitAmount,
					"currency":    linkedProduct.DefaultPrice.Currency,
				}
			}

			linkedProducts = append(linkedProducts, productInfo)
		}
	}
	response["linkedProducts"] = linkedProducts

	// Get products included in tiers
	tierProducts := []map[string]interface{}{}
	if event.Metadata["has_tiers"] == "true" {
		priceParams := &stripe.PriceListParams{
			Product: stripe.String(event.ID),
			Active:  stripe.Bool(true),
		}

		priceIter := price.List(priceParams)
		for priceIter.Next() {
			p := priceIter.Price()

			if p.Nickname == "" {
				continue
			}

			tierInfo := map[string]interface{}{
				"tierId":   p.ID,
				"tierName": p.Nickname,
			}

			// Parse included products
			if includedProductsJSON, ok := p.Metadata["included_products"]; ok && includedProductsJSON != "" {
				var includedProducts []map[string]interface{}
				if err := json.Unmarshal([]byte(includedProductsJSON), &includedProducts); err == nil {
					tierInfo["includedProducts"] = includedProducts
				}
			}

			if products, ok := tierInfo["includedProducts"]; ok && products != nil {
				tierProducts = append(tierProducts, tierInfo)
			}
		}
	}
	response["tierProducts"] = tierProducts

	json.NewEncoder(w).Encode(response)
}

// RemoveProductFromEvent removes a product link from an event
func RemoveProductFromEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["eventId"]
	productID := vars["productId"]

	// Get the event
	eventParams := &stripe.ProductParams{}
	event, err := product.Get(eventID, eventParams)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Get current linked products
	linkedProductIDs := ""
	if ids, ok := event.Metadata["linked_products"]; ok {
		linkedProductIDs = ids
	}

	// Remove the specified product
	productList := strings.Split(linkedProductIDs, ",")
	newProductList := []string{}
	for _, pid := range productList {
		pid = strings.TrimSpace(pid)
		if pid != "" && pid != productID {
			newProductList = append(newProductList, pid)
		}
	}

	// Update event metadata
	updateParams := &stripe.ProductParams{}
	// Copy existing metadata
	for k, v := range event.Metadata {
		updateParams.AddMetadata(k, v)
	}
	updateParams.AddMetadata("linked_products", strings.Join(newProductList, ","))

	_, err = product.Update(eventID, updateParams)
	if err != nil {
		log.Printf("Error updating event: %v", err)
		http.Error(w, "Failed to remove product", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Product %s removed from event %s", productID, eventID),
	})
}

func UpdateTierIncludedProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	priceID := vars["priceId"]

	var req struct {
		IncludedProducts []struct {
			ProductID string `json:"productId"`
			Quantity  int    `json:"quantity"`
		} `json:"includedProducts"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get the price tier
	priceObj, err := price.Get(priceID, nil)
	if err != nil {
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	// Build included products with details
	includedProducts := []map[string]interface{}{}
	totalValue := int64(0)

	for _, p := range req.IncludedProducts {
		// Fetch product details
		prodParams := &stripe.ProductParams{}
		prodParams.AddExpand("default_price")

		prod, err := product.Get(p.ProductID, prodParams)
		if err != nil {
			log.Printf("Product %s not found: %v", p.ProductID, err)
			continue
		}

		productInfo := map[string]interface{}{
			"id":       p.ProductID,
			"quantity": p.Quantity,
			"name":     prod.Name,
			"type":     prod.Metadata["type"], // "merchandise", "addon", etc.
		}

		// Add value if available
		if prod.DefaultPrice != nil {
			productInfo["value"] = prod.DefaultPrice.UnitAmount
			totalValue += prod.DefaultPrice.UnitAmount * int64(p.Quantity)
		}

		// Add image if available
		if len(prod.Images) > 0 {
			productInfo["image"] = prod.Images[0]
		}

		includedProducts = append(includedProducts, productInfo)
	}

	// Update price metadata
	updateParams := &stripe.PriceParams{}

	// Copy existing metadata
	if priceObj.Metadata != nil {
		for k, v := range priceObj.Metadata {
			updateParams.AddMetadata(k, v)
		}
	}

	// Update included products
	includedProductsJSON, _ := json.Marshal(includedProducts)
	updateParams.AddMetadata("included_products", string(includedProductsJSON))
	updateParams.AddMetadata("included_value", strconv.FormatInt(totalValue, 10))
	updateParams.AddMetadata("has_included_products", "true")

	updatedPrice, err := price.Update(priceID, updateParams)
	if err != nil {
		log.Printf("Error updating price: %v", err)
		http.Error(w, "Failed to update tier", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"priceId":          updatedPrice.ID,
		"includedProducts": includedProducts,
		"totalValue":       totalValue,
	})
}

// GetEventMerchandiseRecommendations returns recommended products for an event
func GetEventMerchandiseRecommendations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["eventId"]

	// Optional: exclude already included products
	excludePriceID := r.URL.Query().Get("excludePrice")

	// Get event
	event, err := product.Get(eventID, nil)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	recommendations := []map[string]interface{}{}
	excludedProductIDs := make(map[string]bool)

	// If a price ID is provided, get its included products to exclude them
	if excludePriceID != "" {
		priceObj, err := price.Get(excludePriceID, nil)
		if err == nil && priceObj.Metadata != nil {
			if includedJSON, ok := priceObj.Metadata["included_products"]; ok && includedJSON != "" {
				var included []map[string]interface{}
				if json.Unmarshal([]byte(includedJSON), &included) == nil {
					for _, p := range included {
						if id, ok := p["id"].(string); ok {
							excludedProductIDs[id] = true
						}
					}
				}
			}
		}
	}

	// Get linked products for this event
	if linkedProductIDs, ok := event.Metadata["linked_products"]; ok && linkedProductIDs != "" {
		productIDs := strings.Split(linkedProductIDs, ",")

		for _, pid := range productIDs {
			pid = strings.TrimSpace(pid)
			if pid == "" || excludedProductIDs[pid] {
				continue
			}

			// Fetch product details
			linkedParams := &stripe.ProductParams{}
			linkedParams.AddExpand("default_price")

			linkedProduct, err := product.Get(pid, linkedParams)
			if err != nil || !linkedProduct.Active {
				continue
			}

			// Only recommend merchandise and addons
			productType := linkedProduct.Metadata["type"]
			if productType != "merchandise" && productType != "addon" {
				continue
			}

			recommendation := map[string]interface{}{
				"id":          linkedProduct.ID,
				"name":        linkedProduct.Name,
				"description": linkedProduct.Description,
				"type":        productType,
				"images":      linkedProduct.Images,
			}

			if linkedProduct.DefaultPrice != nil {
				recommendation["price"] = map[string]interface{}{
					"id":          linkedProduct.DefaultPrice.ID,
					"unit_amount": linkedProduct.DefaultPrice.UnitAmount,
					"currency":    linkedProduct.DefaultPrice.Currency,
				}
			}

			// Add recommendation reason
			if productType == "merchandise" {
				recommendation["reason"] = "Official event merchandise"
			} else {
				recommendation["reason"] = "Enhance your event experience"
			}

			recommendations = append(recommendations, recommendation)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"eventId":         eventID,
		"eventName":       event.Name,
	})
}
