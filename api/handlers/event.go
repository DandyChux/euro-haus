package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"euro-haus-api/services"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// ValidateTicketRequest contains the ticket token to validate
type ValidateTicketRequest struct {
	Token string `json:"token"`
}

// TicketInfo contains ticket details returned to the client
type TicketInfo struct {
	Valid         bool   `json:"valid"`
	CustomerName  string `json:"customerName"`
	CustomerEmail string `json:"customerEmail"`
	EventName     string `json:"eventName"`
	EventID       string `json:"eventId"`
	ProductID     string `json:"productId"`
	Quantity      int    `json:"quantity"`
	CheckedIn     bool   `json:"checkedIn"`
	CheckedInAt   string `json:"checkedInAt,omitempty"`
	TicketType    string `json:"ticketType"`
	TicketCode    string `json:"ticketCode"`
}

// type StripeResponse struct {

// }

// ValidateTicket checks if a ticket token is valid
func ValidateTicket(w http.ResponseWriter, r *http.Request) {
	var req ValidateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Check if ticket exists
	ticketKey := "ticket:" + req.Token
	exists, err := rdb.Exists(ctx, ticketKey).Result()
	if err != nil || exists == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "Ticket not found",
		})
		return
	}

	// Get ticket details
	ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
	if err != nil {
		http.Error(w, "Error retrieving ticket data", http.StatusInternalServerError)
		return
	}

	// Parse quantity
	quantity, _ := strconv.Atoi(ticketData["quantity"])

	// Parse checked-in status
	checkedIn := ticketData["checked_in"] == "true"

	// Get ticket type
	ticketType := ticketData["ticket_type"]
	if ticketType == "" {
		ticketType = "General"
	}

	// Format response with all necessary fields
	response := TicketInfo{
		Valid:         true,
		CustomerName:  ticketData["customer_name"],
		CustomerEmail: ticketData["customer_email"],
		EventName:     ticketData["event_name"],
		EventID:       ticketData["stripe_product_id"],
		ProductID:     ticketData["stripe_product_id"],
		Quantity:      quantity,
		CheckedIn:     checkedIn,
		TicketType:    ticketType,
		TicketCode:    req.Token,
	}

	if checkedIn {
		response.CheckedInAt = ticketData["checked_in_at"]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckInTicket marks a ticket as checked in
func CheckInTicket(w http.ResponseWriter, r *http.Request) {
	var req ValidateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Check if ticket exists
	ticketKey := "ticket:" + req.Token
	exists, err := rdb.Exists(ctx, ticketKey).Result()
	if err != nil || exists == 0 {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	// Get ticket data
	ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
	if err != nil {
		http.Error(w, "Error retrieving ticket data", http.StatusInternalServerError)
		return
	}

	// Check if already checked in
	if ticketData["checked_in"] == "true" {
		// Return the ticket info with already checked in status
		quantity, _ := strconv.Atoi(ticketData["quantity"])

		response := TicketInfo{
			Valid:         true,
			CustomerName:  ticketData["customer_name"],
			CustomerEmail: ticketData["customer_email"],
			EventName:     ticketData["event_name"],
			EventID:       ticketData["stripe_product_id"],
			ProductID:     ticketData["stripe_product_id"],
			Quantity:      quantity,
			CheckedIn:     true,
			CheckedInAt:   ticketData["checked_in_at"],
			TicketType:    ticketData["ticket_type"],
			TicketCode:    req.Token,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Mark as checked in with timestamp
	now := time.Now().Format(time.RFC3339)
	updates := map[string]interface{}{
		"checked_in":    "true",
		"checked_in_at": now,
	}

	if err := rdb.HSet(ctx, ticketKey, updates).Err(); err != nil {
		http.Error(w, "Error updating ticket", http.StatusInternalServerError)
		return
	}

	// Get product ID and customer name for the ticket
	productID := ticketData["stripe_product_id"]
	customerName := ticketData["customer_name"]

	if productID != "" {
		// Add to checked-in set
		rdb.SAdd(ctx, "event:"+productID+":checked_in", req.Token)

		// Publish real-time update for all check-in stations
		updateMsg := map[string]interface{}{
			"action":    "check_in",
			"ticket":    req.Token,
			"customer":  customerName,
			"timestamp": now,
		}
		updateJSON, _ := json.Marshal(updateMsg)
		rdb.Publish(ctx, "event:"+productID+":updates", string(updateJSON))

		log.Printf("Ticket %s checked in for event %s by %s", req.Token, productID, customerName)
	}

	// Return updated ticket info
	quantity, _ := strconv.Atoi(ticketData["quantity"])

	response := TicketInfo{
		Valid:         true,
		CustomerName:  customerName,
		CustomerEmail: ticketData["customer_email"],
		EventName:     ticketData["event_name"],
		EventID:       productID,
		ProductID:     productID,
		Quantity:      quantity,
		CheckedIn:     true,
		CheckedInAt:   now,
		TicketType:    ticketData["ticket_type"],
		TicketCode:    req.Token,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEventAttendees retrieves all attendees for a specific event
func GetEventAttendees(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		http.Error(w, "Missing event_id parameter", http.StatusBadRequest)
		return
	}

	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get all attendee tokens for this event
	attendeeTokens, err := rdb.SMembers(ctx, "event:"+eventID+":attendees").Result()
	if err != nil {
		http.Error(w, "Error retrieving attendees", http.StatusInternalServerError)
		return
	}

	// Get checked-in tokens
	checkedInTokens, _ := rdb.SMembers(ctx, "event:"+eventID+":checked_in").Result()
	checkedInMap := make(map[string]bool)
	for _, token := range checkedInTokens {
		checkedInMap[token] = true
	}

	// Collect attendee information
	attendees := []map[string]interface{}{}

	for _, token := range attendeeTokens {
		ticketKey := "ticket:" + token
		ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
		if err != nil {
			continue
		}

		quantity, _ := strconv.Atoi(ticketData["quantity"])

		attendee := map[string]interface{}{
			"name":      ticketData["customer_name"],
			"email":     ticketData["customer_email"],
			"quantity":  quantity,
			"token":     token,
			"checkedIn": ticketData["checked_in"] == "true",
		}

		if ticketData["checked_in"] == "true" {
			attendee["checkedInAt"] = ticketData["checked_in_at"]
		}

		attendees = append(attendees, attendee)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"attendees": attendees,
		"total":     len(attendees),
		"checkedIn": len(checkedInTokens),
	})
}

// Global map to track WebSocket connections for each event
var eventConnections = make(map[string]map[*websocket.Conn]bool)
var eventConnectionsMutex sync.RWMutex

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		return true
	},
}

// HandleEventUpdates handles WebSocket connections for real-time event updates
func HandleEventUpdates(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		http.Error(w, "Missing event_id parameter", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	defer conn.Close()

	// Register this connection for the event
	eventConnectionsMutex.Lock()
	if eventConnections[eventID] == nil {
		eventConnections[eventID] = make(map[*websocket.Conn]bool)
	}
	eventConnections[eventID][conn] = true
	eventConnectionsMutex.Unlock()

	// Remove connection when function returns
	defer func() {
		eventConnectionsMutex.Lock()
		delete(eventConnections[eventID], conn)
		eventConnectionsMutex.Unlock()
	}()

	// Simple ping/pong to keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// BroadcastEventUpdate sends an update to all connected clients for an event
func BroadcastEventUpdate(eventID string, message []byte) {
	eventConnectionsMutex.RLock()
	connections := eventConnections[eventID]
	eventConnectionsMutex.RUnlock()

	for conn := range connections {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			// Connection is broken, will be removed next time it's accessed
			log.Printf("Error sending message: %v", err)
		}
	}
}

// StartEventUpdatesListener starts a goroutine that listens for event updates
// and broadcasts them to connected clients
func StartEventUpdatesListener(eventID string) {
	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Subscribe to event updates channel
	pubsub := rdb.Subscribe(ctx, "event:"+eventID+":updates")

	// Start goroutine to process messages
	go func() {
		defer pubsub.Close()
		log.Printf("Starting event updates listener for event %s", eventID)

		ch := pubsub.Channel()
		for msg := range ch {
			log.Printf("Received update for event %s: %s", eventID, msg.Payload)
			BroadcastEventUpdate(eventID, []byte(msg.Payload))
		}
	}()
}

// GetEventBySlug retrieves an event by its slug with linked products
func GetEventBySlug(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")

	// Get slug from URL
	vars := mux.Vars(r)
	slug := vars["slug"]

	if slug == "" {
		http.Error(w, "Missing slug parameter", http.StatusBadRequest)
		return
	}

	// Search for products with type=event and matching slug
	params := &stripe.ProductListParams{}
	params.Filters.AddFilter("active", "", "true")
	params.AddExpand("data.default_price")

	var eventProduct *stripe.Product

	i := product.List(params)
	for i.Next() {
		p := i.Product()
		if p.Metadata["type"] == "event" && p.Metadata["slug"] == slug {
			eventProduct = p
			break
		}
	}

	if err := i.Err(); err != nil {
		log.Printf("Error listing products: %v", err)
		http.Error(w, "Error retrieving events", http.StatusInternalServerError)
		return
	}

	if eventProduct == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Format the response
	eventResponse := map[string]interface{}{
		"id":          eventProduct.ID,
		"name":        eventProduct.Name,
		"description": eventProduct.Description,
		"images":      eventProduct.Images,
		"metadata":    eventProduct.Metadata,
	}

	// Add price information if available
	if eventProduct.DefaultPrice != nil {
		dp := eventProduct.DefaultPrice
		eventResponse["price"] = map[string]interface{}{
			"id":          dp.ID,
			"unit_amount": dp.UnitAmount,
			"currency":    dp.Currency,
		}

		// Check if default price requires vehicle submission
		if dp.Metadata != nil && dp.Metadata["requires_vehicle_submission"] == "true" {
			eventResponse["requiresVehicleSubmission"] = true
		} else {
			eventResponse["requiresVehicleSubmission"] = false
		}
	}

	// Parse event date for convenience
	if eventDateStr, ok := eventProduct.Metadata["event_date"]; ok {
		eventResponse["event_date"] = eventDateStr
	}

	// Parse location
	if location, ok := eventProduct.Metadata["location"]; ok {
		eventResponse["location"] = location
	}

	// Check if event is sold out
	if availableSpots, ok := eventProduct.Metadata["available_spots"]; ok {
		if availableSpots == "0" || strings.ToLower(eventProduct.Metadata["status"]) == "soldout" {
			eventResponse["sold_out"] = true
		} else {
			eventResponse["sold_out"] = false
		}
	}

	// Fetch linked products if any
	linkedProducts := []map[string]interface{}{}
	if linkedProductIDs, ok := eventProduct.Metadata["linked_products"]; ok && linkedProductIDs != "" {
		// Parse linked product IDs (comma-separated)
		productIDs := strings.Split(linkedProductIDs, ",")
		for _, pid := range productIDs {
			pid = strings.TrimSpace(pid)
			if pid == "" {
				continue
			}

			// Fetch each linked product
			linkedParams := &stripe.ProductParams{}
			linkedParams.AddExpand("default_price")

			linkedProduct, err := product.Get(pid, linkedParams)
			if err != nil {
				log.Printf("Error fetching linked product %s: %v", pid, err)
				continue
			}

			if !linkedProduct.Active {
				continue
			}

			// Build linked product info
			linkedInfo := map[string]interface{}{
				"id":          linkedProduct.ID,
				"name":        linkedProduct.Name,
				"description": linkedProduct.Description,
				"images":      linkedProduct.Images,
				"type":        linkedProduct.Metadata["type"], // e.g., "merchandise", "addon"
			}

			// Add price info if available
			if linkedProduct.DefaultPrice != nil {
				linkedInfo["price"] = map[string]interface{}{
					"id":          linkedProduct.DefaultPrice.ID,
					"unit_amount": linkedProduct.DefaultPrice.UnitAmount,
					"currency":    linkedProduct.DefaultPrice.Currency,
				}
			}

			linkedProducts = append(linkedProducts, linkedInfo)
		}
	}

	if len(linkedProducts) > 0 {
		eventResponse["linkedProducts"] = linkedProducts
	}

	// Fetch event tiers if needed
	if eventProduct.Metadata["has_tiers"] == "true" {
		// Fetch tier prices for this product
		priceParams := &stripe.PriceListParams{
			Product: stripe.String(eventProduct.ID),
			Active:  stripe.Bool(true),
		}

		var tiers []map[string]interface{}
		priceIter := price.List(priceParams)

		for priceIter.Next() {
			p := priceIter.Price()

			// Skip prices without nicknames (non-tier prices)
			if p.Nickname == "" {
				continue
			}

			// Extract features from metadata
			var features []string
			if featuresJSON, ok := p.Metadata["features"]; ok && featuresJSON != "" {
				if err := json.Unmarshal([]byte(featuresJSON), &features); err != nil {
					log.Printf("Error parsing features: %v", err)
				}
			}

			// Check for vehicle submission requirement
			requiresVehicleSubmission := false
			if p.Metadata["requires_vehicle_submission"] == "true" {
				requiresVehicleSubmission = true
			}

			// Check for most popular flag
			isMostPopular := false
			if p.Metadata["is_most_popular"] == "true" {
				isMostPopular = true
			}

			// Parse included products for this tier
			var includedProducts []map[string]interface{}
			if includedProductsJSON, ok := p.Metadata["included_products"]; ok && includedProductsJSON != "" {
				// Format: [{"id":"prod_xxx","quantity":1,"name":"Event T-Shirt"}]
				if err := json.Unmarshal([]byte(includedProductsJSON), &includedProducts); err != nil {
					log.Printf("Error parsing included products: %v", err)
				} else {
					// Fetch full product details for each included product
					for i, incProduct := range includedProducts {
						if productID, ok := incProduct["id"].(string); ok {
							incParams := &stripe.ProductParams{}
							incParams.AddExpand("default_price")

							if incProd, err := product.Get(productID, incParams); err == nil {
								includedProducts[i]["name"] = incProd.Name
								includedProducts[i]["images"] = incProd.Images
								if incProd.DefaultPrice != nil {
									includedProducts[i]["value"] = incProd.DefaultPrice.UnitAmount
								}
							}
						}
					}
				}
			}

			tier := map[string]interface{}{
				"id":                        p.ID,
				"priceId":                   p.ID,
				"name":                      p.Nickname,
				"amount":                    float64(p.UnitAmount) / 100,
				"currency":                  string(p.Currency),
				"description":               p.Metadata["description"],
				"features":                  features,
				"requiresVehicleSubmission": requiresVehicleSubmission,
				"isMostPopular":             isMostPopular,
			}

			// Add included products if any
			if len(includedProducts) > 0 {
				tier["includedProducts"] = includedProducts
			}

			// Add max quantity if available
			if maxQty, ok := p.Metadata["max_quantity"]; ok && maxQty != "" {
				if maxQtyInt, err := strconv.Atoi(maxQty); err == nil {
					tier["maxQuantity"] = maxQtyInt
				}
			}

			tiers = append(tiers, tier)
		}

		if err := priceIter.Err(); err != nil {
			log.Printf("Error fetching prices: %v", err)
		}

		if len(tiers) > 0 {
			eventResponse["priceTiers"] = tiers
			eventResponse["hasTiers"] = true
		}
	}

	json.NewEncoder(w).Encode(eventResponse)
}

// GetEventTickets retrieves all tickets for a specific event
func GetEventTickets(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")

	// Get eventId from URL
	vars := mux.Vars(r)
	eventID := vars["eventId"]

	if eventID == "" {
		http.Error(w, "Missing eventId parameter", http.StatusBadRequest)
		return
	}

	// Get Redis client from service
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get all attendee tokens for this event
	attendeeTokens, err := rdb.SMembers(ctx, "event:"+eventID+":attendees").Result()
	if err != nil {
		http.Error(w, "Error retrieving tickets", http.StatusInternalServerError)
		return
	}

	// Collect ticket information
	tickets := []map[string]interface{}{}

	for _, token := range attendeeTokens {
		ticketKey := "ticket:" + token
		ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
		if err != nil {
			continue
		}

		// Format ticket data for response
		ticket := map[string]interface{}{
			"id":            token,
			"eventId":       eventID,
			"attendeeEmail": ticketData["customer_email"],
			"attendeeName":  ticketData["customer_name"],
			"ticketType":    ticketData["ticket_type"], // Add this field to your Redis data if needed
			"ticketCode":    token,
			"checkedIn":     ticketData["checked_in"] == "true",
			"purchasedAt":   ticketData["purchased_at"], // Add this field to your Redis data if needed
		}

		if ticketData["checked_in"] == "true" {
			ticket["checkedInAt"] = ticketData["checked_in_at"]
		}

		tickets = append(tickets, ticket)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tickets": tickets,
	})
}

// CleanupEventTickets removes invalid and duplicate tickets for an event
func CleanupEventTickets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get eventId from URL
	vars := mux.Vars(r)
	eventID := vars["eventId"]

	if eventID == "" {
		http.Error(w, "Missing eventId parameter", http.StatusBadRequest)
		return
	}

	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Track statistics
	type CleanupStats struct {
		TotalTickets     int      `json:"totalTickets"`
		InvalidTickets   []string `json:"invalidTickets"`
		DuplicateTickets []string `json:"duplicateTickets"`
		RefundedTickets  []string `json:"refundedTickets"`
		CleanedTickets   int      `json:"cleanedTickets"`
		ValidTickets     int      `json:"validTickets"`
	}

	stats := CleanupStats{}

	// Get all attendee tokens for this event
	attendeeTokens, err := rdb.SMembers(ctx, "event:"+eventID+":attendees").Result()
	if err != nil {
		http.Error(w, "Error retrieving tickets", http.StatusInternalServerError)
		return
	}

	stats.TotalTickets = len(attendeeTokens)

	// Track unique tickets by customer email + payment intent
	uniqueTickets := make(map[string]string) // key: email+paymentintent, value: token
	tokensToRemove := []string{}

	for _, token := range attendeeTokens {
		ticketKey := "ticket:" + token
		ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
		if err != nil {
			continue
		}

		// Check if ticket data is valid
		if ticketData["customer_email"] == "" || ticketData["stripe_product_id"] == "" {
			stats.InvalidTickets = append(stats.InvalidTickets, token)
			tokensToRemove = append(tokensToRemove, token)
			log.Printf("Invalid ticket found (missing data): %s", token)
			continue
		}

		// Check if it's been refunded
		if ticketData["status"] == "refunded" || ticketData["status"] == "cancelled" {
			stats.RefundedTickets = append(stats.RefundedTickets, token)
			tokensToRemove = append(tokensToRemove, token)
			log.Printf("Refunded/cancelled ticket found: %s", token)
			continue
		}

		// Check for duplicates based on customer email and payment intent
		paymentIntentID := ticketData["stripe_payment_intent_id"]
		if paymentIntentID == "" {
			// For checkout sessions without payment intent, use session ID
			paymentIntentID = ticketData["stripe_checkout_session_id"]
		}

		uniqueKey := ticketData["customer_email"] + ":" + paymentIntentID
		if existingToken, exists := uniqueTickets[uniqueKey]; exists {
			// Keep the older ticket (first occurrence)
			stats.DuplicateTickets = append(stats.DuplicateTickets, token)
			tokensToRemove = append(tokensToRemove, token)

			log.Printf("Duplicate ticket found: %s (keeping %s)", token, existingToken)
		} else {
			uniqueTickets[uniqueKey] = token
		}
	}

	// Remove invalid and duplicate tickets
	for _, token := range tokensToRemove {
		// Remove from attendees set
		if err := rdb.SRem(ctx, "event:"+eventID+":attendees", token).Err(); err != nil {
			log.Printf("Error removing ticket %s from attendees set: %v", token, err)
			continue
		}

		// Delete ticket data
		ticketKey := "ticket:" + token
		if err := rdb.Del(ctx, ticketKey).Err(); err != nil {
			log.Printf("Error deleting ticket %s: %v", token, err)
			continue
		}

		stats.CleanedTickets++
		log.Printf("Removed invalid/duplicate ticket: %s", token)
	}

	stats.ValidTickets = stats.TotalTickets - stats.CleanedTickets

	// Log cleanup summary
	log.Printf("Event %s ticket cleanup complete: %d total, %d cleaned, %d valid remaining",
		eventID, stats.TotalTickets, stats.CleanedTickets, stats.ValidTickets)

	// Return cleanup statistics
	json.NewEncoder(w).Encode(stats)
}

// InvalidateTicket marks a ticket as invalid/cancelled (helper function for refunds used in webhook)
func InvalidateTicket(ticketToken string, reason string) error {
	rdb := services.GetRedisClient()
	ctx := context.Background()

	ticketKey := "ticket:" + ticketToken

	// Get ticket data
	ticketData, err := rdb.HGetAll(ctx, ticketKey).Result()
	if err != nil {
		return fmt.Errorf("error retrieving ticket: %v", err)
	}

	if len(ticketData) == 0 {
		return fmt.Errorf("ticket not found: %s", ticketToken)
	}

	// Update ticket status
	updates := map[string]interface{}{
		"status":              "cancelled",
		"cancelled_at":        time.Now().Format(time.RFC3339),
		"cancellation_reason": reason,
	}

	if err := rdb.HSet(ctx, ticketKey, updates).Err(); err != nil {
		return fmt.Errorf("error updating ticket status: %v", err)
	}

	// Remove from attendees set
	eventID := ticketData["stripe_product_id"]
	if eventID != "" {
		if err := rdb.SRem(ctx, "event:"+eventID+":attendees", ticketToken).Err(); err != nil {
			log.Printf("Error removing ticket from attendees set: %v", err)
		}
	}

	log.Printf("Ticket %s invalidated: %s", ticketToken, reason)
	return nil
}
