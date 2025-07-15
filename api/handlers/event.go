package handlers

import (
	"context"
	"encoding/json"
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
	ProductID     string `json:"productId"`
	Quantity      int    `json:"quantity"`
	CheckedIn     bool   `json:"checkedIn"`
	CheckedInAt   string `json:"checkedInAt,omitempty"`
}

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

	// Format response
	response := map[string]interface{}{
		"valid":         true,
		"customerName":  ticketData["customer_name"],
		"customerEmail": ticketData["customer_email"],
		"eventName":     ticketData["event_name"],
		"productId":     ticketData["stripe_product_id"],
		"quantity":      quantity,
		"checkedIn":     checkedIn,
	}

	if checkedIn {
		response["checkedInAt"] = ticketData["checked_in_at"]
	}

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
		http.Error(w, "Ticket not found", http.StatusBadRequest)
		return
	}

	// Get current check-in status
	checkedIn, err := rdb.HGet(ctx, ticketKey, "checked_in").Result()
	if err != nil {
		http.Error(w, "Error retrieving ticket data", http.StatusInternalServerError)
		return
	}

	if checkedIn == "true" {
		http.Error(w, "Ticket already checked in", http.StatusBadRequest)
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
	productID, _ := rdb.HGet(ctx, ticketKey, "stripe_product_id").Result()
	customerName, _ := rdb.HGet(ctx, ticketKey, "customer_name").Result()

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
	}

	// Return updated ticket info
	ValidateTicket(w, r)
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

// GetEventBySlug retrieves an event by its slug
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
