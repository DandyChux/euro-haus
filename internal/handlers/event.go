package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"

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

// LinkProductsRequest represents a request to link products to an event
type LinkProductsRequest struct {
	EventID    string   `json:"event_id"`
	ProductIDs []string `json:"product_ids"`
}

// TierProductsRequest represents products to include in a tier
type TierProductsRequest struct {
	PriceID  string                   `json:"price_id"`
	Products []IncludedProductRequest `json:"products"`
}

type EventWriteRequest struct {
	ID              string `json:"id"`
	StripeProductID string `json:"stripe_product_id"`

	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	LongDescription string `json:"long_description"`
	Images          []string `json:"images"`

	EventDate string `json:"date"`
	Location  string `json:"location"`
	Venue     string `json:"venue"`
	Organizer string `json:"organizer"`

	Capacity       int `json:"capacity"`
	AvailableSpots int `json:"available_spots"`

	Status   string `json:"status"`
	Active   bool   `json:"active"`
	Featured bool   `json:"featured"`

	Tags     models.EventStringList `json:"tags"`
	Agenda   models.EventAgenda     `json:"agenda"`
	Includes models.EventStringList `json:"includes"`
	Sponsors models.EventSponsors    `json:"sponsors"`

	Prices []EventPriceWriteRequest `json:"prices"`
}

type EventPriceWriteRequest struct {
	ID              string `json:"id"`
	StripeProductID string `json:"stripe_product_id"`

	UnitAmount  int64  `json:"unit_amount"`
	Currency    string `json:"currency"`
	Nickname    string `json:"nickname"`
	Description string `json:"description"`

	Active bool `json:"active"`

	Features          []string `json:"features"`
	Default           bool     `json:"default"`
	MostPopular       bool     `json:"most_popular"`
	RequiresApproval  bool     `json:"requires_approval"`
	RequiresSubmission bool    `json:"requires_submission"`

	Requirements []RequirementInput `json:"requirements"`

	Quantity  int    `json:"quantity"`
	Size      string `json:"size"`
	Color     string `json:"color"`
	SoldOut   bool   `json:"sold_out"`
	StockQuantity *int `json:"stock_quantity"`

	IncludedProducts []IncludedProductWriteRequest `json:"included_products"`
}

type IncludedProductWriteRequest struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// IncludedProductRequest represents a product included in a tier
type IncludedProductRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type IncludedProductResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	Quantity    int      `json:"quantity"`
	SortOrder   int      `json:"sortOrder,omitempty"`
	DefaultPrice *models.PriceInfo `json:"default_price,omitempty"`
}

type EventResponse struct {
	ID                   string `json:"id"`
	StripeProductID      string `json:"stripe_product_id,omitempty"`

	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	LongDescription string `json:"long_description"`
	Images          []string `json:"images"`

	EventDate string `json:"date"`
	Location  string `json:"location"`
	Venue     string `json:"venue"`
	Organizer string `json:"organizer,omitempty"`

	Capacity       int `json:"capacity"`
	AvailableSpots int `json:"available_spots"`

	Status   string `json:"status"`
	Active   bool   `json:"active"`
	Featured bool   `json:"featured"`

	Tags     models.EventStringList `json:"tags"`
	Agenda   models.EventAgenda     `json:"agenda"`
	Includes models.EventStringList `json:"includes"`
	Sponsors models.EventSponsors    `json:"sponsors"`

	Prices  []models.PriceInfo `json:"prices,omitempty"`
}

func withEventPrices(db *gorm.DB) *gorm.DB {
	return db.Preload("Prices", func(tx *gorm.DB) *gorm.DB {
		return tx.
			Where("active = ?", true).
			Order("unit_amount ASC, id ASC")
	})
}

func normalizedEventPrices(
	prices []models.PriceInfo,
) []models.PriceInfo {
	if prices == nil {
		prices = []models.PriceInfo{}
	}

	result := append([]models.PriceInfo(nil), prices...)

	if len(result) == 1 {
		result[0].IsDefault = true
	}

	return result
}

func eventToResponse(event models.Event) EventResponse {
	images := []string{}

	if event.Images != nil {
		images = event.Images
	}

	return EventResponse{
		ID:              event.ID,
		StripeProductID: event.StripeProductID,
		Prices:          normalizedEventPrices(event.Prices),

		Slug:            event.Slug,
		Name:            event.Name,
		Description:     event.Description,
		LongDescription: event.LongDescription,
		Images:          images,

		EventDate: event.EventDate,
		Location:  event.Location,
		Venue:     event.Venue,
		Organizer: event.Organizer,

		Capacity:       event.Capacity,
		AvailableSpots: event.AvailableSpots,

		Status:   event.Status,
		Active:   event.Active,
		Featured: event.Featured,

		Tags:     event.Tags,
		Agenda:   event.Agenda,
		Includes: event.Includes,
		Sponsors: event.Sponsors,
	}
}

func loadEventPrices(
	ctx context.Context,
	event *models.Event,
) error {
	var prices []models.PriceInfo

	err := services.GetDB().
		WithContext(ctx).
		Preload("IncludedProductLinks").
		Preload("Requirements", func(tx *gorm.DB) *gorm.DB {
			return tx.
				Where("active = ?", true).
				Order("sort_order ASC, id ASC")
		}).
		Where(
			"stripe_product_id = ? AND active = TRUE",
			event.StripeProductID,
		).
		Order("unit_amount ASC, id ASC").
		Find(&prices).
		Error

	if err != nil {
		return err
	}

	if prices == nil {
		prices = []models.PriceInfo{}
	}

	if len(prices) == 1 {
		prices[0].IsDefault = true
	}

	event.Prices = prices
	return nil
}

func findEventByID(
	ctx context.Context,
	id string,
) (models.Event, error) {
	var event models.Event

	err := services.GetDB().
		WithContext(ctx).
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return event, err
	}

	if err := loadEventPrices(ctx, &event); err != nil {
		return event, err
	}

	return event, nil
}

func findActiveEventByID(ctx context.Context, id string) (models.Event, error) {
	event, err := findEventByID(ctx, id)
	if err != nil {
		return event, err
	}

	if !event.Active {
		return event, gorm.ErrRecordNotFound
	}

	return event, nil
}

func findEventByStripeProductID(
	ctx context.Context,
	stripeProductID string,
) (models.Event, error) {
	var event models.Event

	err := services.GetDB().
		WithContext(ctx).
		Where("stripe_product_id = ?", stripeProductID).
		First(&event).
		Error

	if err != nil {
		return event, err
	}

	if err := loadEventPrices(ctx, &event); err != nil {
		return event, err
	}

	return event, nil
}

func replaceEventProductLinks(
	ctx context.Context,
	eventID string,
	productIDs []string,
) error {
	db := services.GetDB().WithContext(ctx)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("event_id = ?", eventID).
			Delete(&models.EventProductLink{}).
			Error; err != nil {
			return err
		}

		links := make([]models.EventProductLink, 0, len(productIDs))

		for index, productID := range productIDs {
			links = append(links, models.EventProductLink{
				EventID:   eventID,
				ProductID: productID,
				SortOrder: index,
			})
		}

		if len(links) == 0 {
			return nil
		}

		return tx.Create(&links).Error
	})
}

func removeEventProductLink(
	ctx context.Context,
	eventID string,
	productID string,
) error {
	result := services.GetDB().
		WithContext(ctx).
		Where("event_id = ? AND product_id = ?", eventID, productID).
		Delete(&models.EventProductLink{})

	return result.Error
}

func includedProductID(
	product IncludedProductWriteRequest,
) string {
	if product.ProductID != "" {
		return product.ProductID
	}

	return product.ID
}

func marshalPriceFeatures(
	features []string,
) ([]byte, error) {
	if features == nil {
		features = []string{}
	}

	return json.Marshal(features)
}

func replacePriceIncludedProductLinks(
	tx *gorm.DB,
	priceID string,
	products []IncludedProductWriteRequest,
) error {
	if err := tx.
		Where("price_id = ?", priceID).
		Delete(&models.PriceIncludedProduct{}).
		Error; err != nil {
		return err
	}

	links := make([]models.PriceIncludedProduct, 0, len(products))

	for index, included := range products {
		productID := includedProductID(included)

		if productID == "" {
			continue
		}

		quantity := included.Quantity

		if quantity < 1 {
			quantity = 1
		}

		links = append(links, models.PriceIncludedProduct{
			PriceID:   priceID,
			ProductID: productID,
			Quantity:  quantity,
			SortOrder: index,
		})
	}

	if len(links) == 0 {
		return nil
	}

	return tx.Create(&links).Error
}

func replacePriceRequirements(
	tx *gorm.DB,
	priceID string,
	inputs []RequirementInput,
) error {
	seenKeys := make(map[string]struct{}, len(inputs))

	for index := range inputs {
		input := inputs[index]

		input.Key = strings.TrimSpace(input.Key)
		input.Label = strings.TrimSpace(input.Label)
		input.Type = strings.TrimSpace(input.Type)
		input.SortOrder = index

		if input.Active == false {
			// New frontend rows may omit active, so treat them as active.
			input.Active = true
		}

		if err := validateRequirementInput(input); err != nil {
			return err
		}

		if _, exists := seenKeys[input.Key]; exists {
			return fmt.Errorf(
				"duplicate requirement key %q for price %s",
				input.Key,
				priceID,
			)
		}

		seenKeys[input.Key] = struct{}{}

		inputs[index] = input
	}

	// Preserve old definitions for historical submission answers.
	if err := tx.
		Model(&models.PriceRequirement{}).
		Where("price_id = ?", priceID).
		Update("active", false).
		Error; err != nil {
		return err
	}

	for _, input := range inputs {
		requirement := requirementFromInput(priceID, input)

		if requirement.ID == "" {
			if err := tx.Create(&requirement).Error; err != nil {
				return err
			}

			continue
		}

		var existing models.PriceRequirement

		err := tx.
			Where("id = ? AND price_id = ?", requirement.ID, priceID).
			First(&existing).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&requirement).Error; err != nil {
				return err
			}

			continue
		}

		if err != nil {
			return err
		}

		if err := tx.
			Model(&existing).
			Updates(map[string]any{
				"key":        requirement.Key,
				"label":      requirement.Label,
				"type":       requirement.Type,
				"required":   requirement.Required,
				"options":    requirement.Options,
				"sort_order": requirement.SortOrder,
				"active":     true,
			}).
			Error; err != nil {
			return err
		}
	}

	return nil
}

func upsertEventPrice(
	tx *gorm.DB,
	eventProductID string,
	request EventPriceWriteRequest,
) (string, error) {
	featuresJSON, err := marshalPriceFeatures(request.Features)
	if err != nil {
		return "", err
	}

	priceID := request.ID

	if priceID == "" {
		// New local prices must first be created in Stripe because the
		// Stripe Price ID is also the primary key of the local prices row.
		stripePrice, err := price.New(&stripe.PriceParams{
			Product:    stripe.String(eventProductID),
			UnitAmount: stripe.Int64(request.UnitAmount),
			Currency:   stripe.String(request.Currency),
			Nickname:   stripe.String(request.Nickname),
		})
		if err != nil {
			return "", fmt.Errorf("create Stripe price: %w", err)
		}

		priceID = stripePrice.ID
	} else {
		var existing models.PriceInfo

		if err := tx.
			Where("id = ?", priceID).
			First(&existing).
			Error; err != nil {
			return "", fmt.Errorf("load local price %s: %w", priceID, err)
		}

		if existing.UnitAmount != request.UnitAmount ||
			existing.Currency != request.Currency {
			stripePrice, err := price.New(&stripe.PriceParams{
				Product:    stripe.String(eventProductID),
				UnitAmount: stripe.Int64(request.UnitAmount),
				Currency:   stripe.String(request.Currency),
				Nickname:   stripe.String(request.Nickname),
			})
			if err != nil {
				return "", fmt.Errorf("create replacement Stripe price: %w", err)
			}

			if _, err := price.Update(
				existing.ID,
				&stripe.PriceParams{
					Active: stripe.Bool(false),
				},
			); err != nil {
				return "", fmt.Errorf(
					"archive replaced Stripe price %s: %w",
					existing.ID,
					err,
				)
			}

			priceID = stripePrice.ID
		} else {
			if _, err := price.Update(
				priceID,
				&stripe.PriceParams{
					Nickname: stripe.String(request.Nickname),
					Active:   stripe.Bool(request.Active),
				},
			); err != nil {
				return "", fmt.Errorf("update Stripe price %s: %w", priceID, err)
			}
		}
	}

	priceInfo := models.PriceInfo{
		ID:                 priceID,
		StripeProductID:    eventProductID,
		UnitAmount:         request.UnitAmount,
		Currency:           request.Currency,
		Nickname:           request.Nickname,
		Description:        request.Description,
		Active:             request.Active,
		Features:           featuresJSON,
		IsDefault:          request.Default,
		IsMostPopular:      request.MostPopular,
		RequiresApproval:   request.RequiresApproval,
		RequiresSubmission: request.RequiresSubmission,
		Quantity:           request.Quantity,
		StockQuantity:      request.StockQuantity,
		Size:               request.Size,
		Color:              request.Color,
	}

	if err := tx.Save(&priceInfo).Error; err != nil {
		return "", fmt.Errorf("save local price %s: %w", priceID, err)
	}

	if err := replacePriceIncludedProductLinks(
		tx,
		priceID,
		request.IncludedProducts,
	); err != nil {
		return "", fmt.Errorf(
			"save included products for price %s: %w",
			priceID,
			err,
		)
	}

	return priceID, nil
}

// ValidateTicket checks if a ticket token is valid
func ValidateTicket(w http.ResponseWriter, r *http.Request) {
	var req ValidateTicketRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "Ticket token is required", http.StatusBadRequest)
		return
	}

	db := services.GetDB()

	var (
		info        models.TicketInfo
		checkedInAt sql.NullTime
		invalidated bool
	)

	err := db.WithContext(r.Context()).Raw(`
		SELECT
			customer_name,
			customer_email,
			event_id,
			quantity,
			checked_in,
			checked_in_at,
			ticket_type,
			invalidated
		FROM tickets
		WHERE token = ?
	`, req.Token).Row().Scan(
		&info.CustomerName,
		&info.CustomerEmail,
		&info.EventID,
		&info.Quantity,
		&info.CheckedIn,
		&checkedInAt,
		&info.TicketType,
		&invalidated,
	)

	w.Header().Set("Content-Type", "application/json")

	if errors.Is(err, sql.ErrNoRows) || invalidated {
		w.WriteHeader(http.StatusNotFound)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "Ticket not found",
		})
		return
	}

	if err != nil {
		log.Printf("Error retrieving ticket %s: %v", req.Token, err)
		http.Error(w, "Error retrieving ticket data", http.StatusInternalServerError)
		return
	}

	if info.TicketType == "" {
		info.TicketType = "General"
	}

	if checkedInAt.Valid {
		info.CheckedInAt = checkedInAt.Time.UTC().Format(time.RFC3339)
	}

	info.Valid = true
	info.TicketCode = req.Token

	_ = json.NewEncoder(w).Encode(info)
}

// CheckInTicket marks a ticket as checked in
func CheckInTicket(w http.ResponseWriter, r *http.Request) {
	var req ValidateTicketRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "Ticket token is required", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	ctx := r.Context()

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var (
		info        models.TicketInfo
		checkedIn   bool
		checkedInAt sql.NullTime
	)

	err := tx.Raw(`
		SELECT
			t.customer_name,
			t.customer_email,
			COALESCE(NULLIF(t.event_id, ''), t.stripe_product_id),
			t.quantity,
			t.checked_in,
			t.checked_in_at,
			t.ticket_type
		FROM tickets t
		LEFT JOIN events e
			ON e.stripe_product_id =
			   COALESCE(NULLIF(t.event_id, ''), t.stripe_product_id)
		WHERE t.token = ?
		  AND t.invalidated = FALSE
		FOR UPDATE
	`, req.Token).Row().Scan(
		&info.CustomerName,
		&info.CustomerEmail,
		&info.EventID,
		&info.Quantity,
		&checkedIn,
		&checkedInAt,
		&info.TicketType,
	)

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error retrieving ticket %s: %v", req.Token, err)
		http.Error(w, "Error retrieving ticket data", http.StatusInternalServerError)
		return
	}

	info.Valid = true
	info.TicketCode = req.Token

	if info.TicketType == "" {
		info.TicketType = "General"
	}

	if checkedIn {
		info.CheckedIn = true

		if checkedInAt.Valid {
			info.CheckedInAt = checkedInAt.Time.UTC().Format(time.RFC3339)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
		return
	}

	now := time.Now().UTC()

	if err := tx.Exec(`
		UPDATE tickets
		SET checked_in = TRUE,
		    checked_in_at = ?,
		    updated_at = NOW()
		WHERE token = ?
	`, now, req.Token).Error; err != nil {
		http.Error(w, "Error updating ticket", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Error committing check-in", http.StatusInternalServerError)
		return
	}

	info.CheckedIn = true
	info.CheckedInAt = now.Format(time.RFC3339)

	BroadcastEventUpdate(info.EventID, map[string]interface{}{
		"action":    "check_in",
		"eventId":   info.EventID,
		"ticket":    req.Token,
		"customer":  info.CustomerName,
		"timestamp": info.CheckedInAt,
	})

	log.Printf(
		"Ticket %s checked in for event %s by %s",
		req.Token,
		info.EventID,
		info.CustomerName,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// GetEventAttendees retrieves all attendees for a specific event
func GetEventAttendees(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Missing event_id parameter", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	rows, err := db.WithContext(r.Context()).Raw(`
		SELECT token, ticket_type, customer_name, customer_email, quantity, checked_in, checked_in_at
		FROM tickets
		WHERE event_id = ? AND invalidated = FALSE
		ORDER BY customer_name
	`, eventID).Rows()
	if err != nil {
		http.Error(w, "Error retrieving attendees", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	attendees := []map[string]interface{}{}
	checkedInCount := 0
	for rows.Next() {
		var (
			token, ticketType, name, email string
			quantity           int
			checkedIn          bool
			checkedInAt        sql.NullTime
		)
		if err := rows.Scan(&token, &ticketType, &name, &email, &quantity, &checkedIn, &checkedInAt); err != nil {
			continue
		}
		a := map[string]interface{}{
			"name":      name,
			"email":     email,
			"quantity":  quantity,
			"ticket_type": ticketType,
			"token":     token,
			"checked_in": checkedIn,
		}
		if checkedIn {
			checkedInCount++
			if checkedInAt.Valid {
				a["checked_in_at"] = checkedInAt.Time.Format(time.RFC3339)
			}
		}
		attendees = append(attendees, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"attendees": attendees,
		"stats": map[string]interface{}{
			"total":     len(attendees),
			"checked_in": checkedInCount,
		},
	})
}

func LinkProductsToEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventID := mux.Vars(r)["id"]

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	var req struct {
		ProductIDs []string `json:"productIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validProductIDs := make([]string, 0, len(req.ProductIDs))
	seen := make(map[string]struct{})

	for _, productID := range req.ProductIDs {
		productID = strings.TrimSpace(productID)

		if productID == "" {
			continue
		}

		if _, exists := seen[productID]; exists {
			continue
		}

		linkedProduct, err := product.Get(productID, nil)
		if err != nil {
			log.Printf("Product %s not found: %v", productID, err)
			continue
		}

		if !linkedProduct.Active {
			continue
		}

		seen[productID] = struct{}{}
		validProductIDs = append(validProductIDs, productID)
	}

	if err := replaceEventProductLinks(
		r.Context(),
		event.ID,
		validProductIDs,
	); err != nil {
		log.Printf("Unable to replace event product links: %v", err)
		http.Error(w, "Failed to link products", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"eventId":        event.ID,
		"linkedProducts": validProductIDs,
	})
}

// Global map to track WebSocket connections for each event
var eventConnections = make(map[*websocket.Conn]string)
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
	eventConnections[conn] = eventID
	eventConnectionsMutex.Unlock()

	// Remove connection when function returns
	defer func() {
		eventConnectionsMutex.Lock()
		delete(eventConnections, conn)
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

// BroadcastEventUpdate triggers notification broadcast.
func BroadcastEventUpdate(eventID string, payload map[string]interface{}) {
	db := services.GetDB()
	if db == nil {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("BroadcastEventUpdate marshal: %v", err)
		return
	}

	// pg_notify avoids needing to interpolate channel names into NOTIFY
	body := eventID + "|" + string(data)
	if err := db.Exec(`SELECT pg_notify('event_updates', ?)`, body).Error; err != nil {
		log.Printf("pg_notify failed: %v", err)
	}
}

// StartEventUpdatesListener listens for changes on the Postgres NOTIFY channel for a specific event
func StartEventUpdatesListener(eventID string) {
	dsn := services.GetDatabaseDSN()
	if dsn == "" {
		log.Printf("Database DSN not available for event updates listener")
		return
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Printf("Error connecting for event %s: %v", eventID, err)
		return
	}

	channelName := fmt.Sprintf("event_updates_%s", eventID)

	go func() {
		defer conn.Close(ctx)

		_, err := conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channelName))
		if err != nil {
			log.Printf("Error listening to channel %s: %v", channelName, err)
			return
		}
		log.Printf("Starting event updates listener for event %s on Postgres channel %s", eventID, channelName)

		for {
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				log.Printf("Notification loop error for event %s: %v", eventID, err)
				return
			}

			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
				log.Printf("JSON unmarshal error on notify channel: %v", err)
				continue
			}

			BroadcastEventUpdate(eventID, payload)
		}
	}()
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	query := services.GetDB().
		WithContext(r.Context()).
		Order("event_date DESC")

	if !includeInactive {
		query = query.Where("active = TRUE")
	}

	var events []models.Event
	if err := query.Find(&events).Error; err != nil {
		log.Printf("Error retrieving events: %v", err)
		http.Error(w, "Error retrieving events", http.StatusInternalServerError)
		return
	}

	response := make([]EventResponse, 0, len(events))

	for _, event := range events {
		if err := loadEventPrices(r.Context(), &event); err != nil {
			log.Printf("Error retrieving prices for event %s: %v", event.ID, err)

			http.Error(w, "Error retrieving event prices", http.StatusInternalServerError)
			return
		}
		response = append(response, eventToResponse(event))
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding event list: %v", err)
		http.Error(w, "Error encoding event list", http.StatusInternalServerError)
	}
}

func GetEventByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Missing event ID", http.StatusBadRequest)
		return
	}

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error retrieving event %s: %v", eventID, err)
		http.Error(w, "Error retrieving event", http.StatusInternalServerError)
		return
	}

	response := eventToResponse(event)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding event %s: %v", eventID, err)
	}
}

func GetAdminEventByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Missing event ID", http.StatusBadRequest)
		return
	}

	event, err := findEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error retrieving event %s: %v", eventID, err)
		http.Error(w, "Error retrieving event", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(eventToResponse(event)); err != nil {
		log.Printf("Error encoding event %s: %v", eventID, err)
	}
}

func getEventStripeProduct(event models.Event) (*stripe.Product, error) {
	params := &stripe.ProductParams{}
	params.AddExpand("default_price")

	return product.Get(event.StripeProductID, params)
}

// GetEventTickets returns list of event tickets
func GetEventTickets(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]

	db := services.GetDB()
	rows, err := db.WithContext(r.Context()).Raw(`
		SELECT token, customer_name, customer_email, ticket_type, quantity,
			checked_in, checked_in_at, created_at
		FROM tickets
		WHERE event_id = ? AND invalidated = FALSE
		ORDER BY created_at DESC
	`, eventID).Rows()
	if err != nil {
		http.Error(w, "Failed to retrieve tickets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tickets := []map[string]interface{}{}
	for rows.Next() {
		var (
			token, name, email, ticketType string
			quantity                       int
			checkedIn                      bool
			checkedInAt                    sql.NullTime
			createdAt                      time.Time
		)
		if err := rows.Scan(&token, &name, &email, &ticketType, &quantity, &checkedIn, &checkedInAt, &createdAt); err != nil {
			log.Printf("Error scanning ticket: %v", err)
			continue
		}
		t := map[string]interface{}{
			"token":         token,
			"customerName":  name,
			"customerEmail": email,
			"ticketType":    ticketType,
			"quantity":      quantity,
			"checked_in":     checkedIn,
			"createdAt":     createdAt.Format(time.RFC3339),
		}
		if checkedInAt.Valid {
			t["checked_in_at"] = checkedInAt.Time.Format(time.RFC3339)
		}
		tickets = append(tickets, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

func GetEventPrices(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"prices": normalizedEventPrices(event.Prices),
	})
}


// CleanupEventTickets cleans up tickets
func CleanupEventTickets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	if eventID == "" {
		http.Error(w, "Missing eventId parameter", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	ctx := context.Background()

	type CleanupStats struct {
		TotalTickets int `json:"totalTickets"`
		UnusedDels   int `json:"unusedDels"`
		Remaining    int `json:"remaining"`
	}

	var stats CleanupStats

	// Get total count
	err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM tickets WHERE event_id = ?`, eventID).Row().Scan(&stats.TotalTickets)
	if err != nil {
		http.Error(w, "Error retrieving ticket total", http.StatusInternalServerError)
		return
	}

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Deletes unused/not checked in tickets
	query := `
		DELETE FROM tickets
		WHERE event_id = ? AND checked_in = FALSE
	`
	res := tx.Exec(query, eventID)
	if res.Error != nil {
		log.Printf("Cleanup query failed: %v", res.Error)
		http.Error(w, "Deletion failed", http.StatusInternalServerError)
		return
	}
	stats.UnusedDels = int(res.RowsAffected)

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Commit failed", http.StatusInternalServerError)
		return
	}

	stats.Remaining = stats.TotalTickets - stats.UnusedDels

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// InvalidateTicket marks a ticket as invalidated.
func InvalidateTicket(ticketToken string, reason string) error {
	db := services.GetDB()
	err := db.Exec(`
		UPDATE tickets
		SET invalidated = TRUE,
		    invalidated_reason = ?,
		    invalidated_at = NOW()
		WHERE token = ?
	`, reason, ticketToken).Error
	if err != nil {
		return fmt.Errorf("invalidate ticket: %w", err)
	}
	return nil
}

func AddProductsToTier(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventID := mux.Vars(r)["eventId"]
	priceId := mux.Vars(r)["priceId"]

	if strings.TrimSpace(eventID) == "" {
		http.Error(
			w,
			"Missing event ID",
			http.StatusBadRequest,
		)
		return
	}

	var req TierProductsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if eventID == "" || priceId == "" {
		http.Error(
			w,
			"Event ID and price ID are required",
			http.StatusBadRequest,
		)
		return
	}

	event, err := findEventByID(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		http.Error(
			w,
			"Failed to load event",
			http.StatusInternalServerError,
		)
		return
	}

	var eventPrice models.PriceInfo

	db := services.GetDB().WithContext(r.Context())

	err = db.
		Where(
			"id = ? AND stripe_product_id = ?",
			priceId,
			event.StripeProductID,
		).
		First(&eventPrice).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(
				w,
				"Price does not belong to this event",
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			"Failed to validate price",
			http.StatusInternalServerError,
		)
		return
	}

	if err := replacePriceIncludedProducts(
		r.Context(),
		priceId,
		req.Products,
	); err != nil {
		log.Printf(
			"Failed to replace included products for price %s: %v",
			priceId,
			err,
		)

		http.Error(
			w,
			"Failed to update included products",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"price_id": priceId,
	})
}

// GetEventLinkedProducts retrieves all products linked to an event
func GetEventLinkedProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["id"]

	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Unable to retrieve event", http.StatusInternalServerError)
		return
	}

	var eventLinks []models.EventProductLink

	err = services.GetDB().
		WithContext(r.Context()).
		Where("event_id = ?", event.ID).
		Order("sort_order ASC, product_id ASC").
		Find(&eventLinks).
		Error

	if err != nil {
		http.Error(
			w,
			"Unable to retrieve linked products",
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"event_id":   event.ID,
		"event_name": event.Name,
	}

	// Get directly linked products (add-ons)
	linkedProducts := []map[string]interface{}{}

	for _, link := range eventLinks {
		linkedParams := &stripe.ProductParams{}
		linkedParams.AddExpand("default_price")

		linkedProduct, err := product.Get(link.ProductID, linkedParams)
		if err != nil {
			log.Printf(
				"Unable to retrieve linked product %s: %v",
				link.ProductID,
				err,
			)
			continue
		}

		productInfo := map[string]interface{}{
			"id":          linkedProduct.ID,
			"name":        linkedProduct.Name,
			"description": linkedProduct.Description,
			"images":      linkedProduct.Images,
			"active":      linkedProduct.Active,
			"sort_order":  link.SortOrder,
		}

		if linkedProduct.DefaultPrice != nil {
			productInfo["default_price"] = linkedProduct.DefaultPrice
		}

		linkedProducts = append(linkedProducts, productInfo)
	}
	response["linked_products"] = linkedProducts

	// Get products included in tiers
	tierProducts := []map[string]interface{}{}

	for _, tier := range event.Prices {
		if tier.Nickname == "" {
			continue
		}

		tierInfo := map[string]interface{}{
			"tierId":   tier.ID,
			"tierName": tier.Nickname,
			"amount":   float64(tier.UnitAmount) / 100,
			"currency": tier.Currency,
		}

		var includedLinks []models.PriceIncludedProduct

		err := services.GetDB().
			WithContext(r.Context()).
			Where("price_id = ?", tier.ID).
			Order("sort_order ASC, product_id ASC").
			Find(&includedLinks).
			Error

		if err != nil {
			http.Error(
				w,
				"Unable to retrieve tier products",
				http.StatusInternalServerError,
			)
			return
		}

		includedProducts := []map[string]interface{}{}

		for _, link := range includedLinks {
			linkedParams := &stripe.ProductParams{}
			linkedParams.AddExpand("default_price")

			linkedProduct, err := product.Get(link.ProductID, linkedParams)
			if err != nil {
				continue
			}

			productInfo := map[string]interface{}{
				"id":         linkedProduct.ID,
				"name":       linkedProduct.Name,
				"quantity":   link.Quantity,
				"sort_order": link.SortOrder,
			}

			if linkedProduct.Description != "" {
				productInfo["description"] = linkedProduct.Description
			}

			if len(linkedProduct.Images) > 0 {
				productInfo["images"] = linkedProduct.Images
			}

			if linkedProduct.DefaultPrice != nil {
				productInfo["default_price"] = map[string]interface{}{
					"id":          linkedProduct.DefaultPrice.ID,
					"unit_amount": linkedProduct.DefaultPrice.UnitAmount,
					"currency":    linkedProduct.DefaultPrice.Currency,
				}
			}

			includedProducts = append(includedProducts, productInfo)
		}

		tierInfo["included_products"] = includedProducts
		tierProducts = append(tierProducts, tierInfo)
	}
	response["tier_products"] = tierProducts

	json.NewEncoder(w).Encode(response)
}

func RemoveProductFromEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventID := vars["id"]
	productID := vars["productId"]

	if eventID == "" || productID == "" {
		http.Error(w, "Missing event or product ID", http.StatusBadRequest)
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

	if err := removeEventProductLink(
		r.Context(),
		event.ID,
		productID,
	); err != nil {
		log.Printf("Unable to remove event product link: %v", err)
		http.Error(w, "Failed to remove product", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// GetEventMerchandiseRecommendations returns recommended products for an event.
func GetEventMerchandiseRecommendations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventID := mux.Vars(r)["id"]
	excludePriceID := r.URL.Query().Get("excludePrice")

	db := services.GetDB().WithContext(r.Context())

	// The event relationship is database-owned.
	event, err := findActiveEventByID(r.Context(), eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Unable to retrieve event %s: %v", eventID, err)
		http.Error(
			w,
			"Unable to retrieve event",
			http.StatusInternalServerError,
		)
		return
	}

	excludedProductIDs := make(map[string]bool)

	// Load products included in the selected tier from PostgreSQL.
	if excludePriceID != "" {
		var tier models.PriceInfo

		err := db.
			Where(
				"id = ? AND stripe_product_id = ? AND active = TRUE",
				excludePriceID,
				event.StripeProductID,
			).
			First(&tier).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price not found for event", http.StatusNotFound)
			return
		}

		if err != nil {
			log.Printf(
				"Unable to retrieve price %s: %v",
				excludePriceID,
				err,
			)
			http.Error(
				w,
				"Unable to retrieve price",
				http.StatusInternalServerError,
			)
			return
		}

		var includedProducts []models.PriceIncludedProduct

		err = db.
			Where("price_id = ?", tier.ID).
			Order("sort_order ASC, product_id ASC").
			Find(&includedProducts).
			Error

		if err != nil {
			log.Printf(
				"Unable to retrieve included products for price %s: %v",
				tier.ID,
				err,
			)
			http.Error(
				w,
				"Unable to retrieve included products",
				http.StatusInternalServerError,
			)
			return
		}

		for _, includedProduct := range includedProducts {
			excludedProductIDs[includedProduct.ProductID] = true
		}
	}

	// Load linked add-on product IDs from PostgreSQL.
	var eventLinks []models.EventProductLink

	err = db.
		Where("event_id = ?", event.ID).
		Order("sort_order ASC, product_id ASC").
		Find(&eventLinks).
		Error

	if err != nil {
		log.Printf(
			"Unable to retrieve linked products for event %s: %v",
			event.ID,
			err,
		)
		http.Error(
			w,
			"Unable to retrieve linked products",
			http.StatusInternalServerError,
		)
		return
	}

	recommendations := make([]map[string]interface{}, 0)

	for _, eventLink := range eventLinks {
		productID := strings.TrimSpace(eventLink.ProductID)

		if productID == "" || excludedProductIDs[productID] {
			continue
		}

		/*
			Product relationships and quantities come from PostgreSQL.

			Product catalog details still come from Stripe because there is
			currently no local products table.
		*/
		productParams := &stripe.ProductParams{}
		productParams.AddExpand("default_price")

		linkedProduct, err := product.Get(productID, productParams)
		if err != nil {
			log.Printf(
				"Unable to retrieve linked product %s: %v",
				productID,
				err,
			)
			continue
		}

		if !linkedProduct.Active {
			continue
		}

		var localProduct models.Product

		if err := db.
			Where("id = ?", productID).
			First(&localProduct).
			Error; err != nil {
				log.Printf(
					"Failed to load local product %s: %v",
					linkedProduct.ID,
					err,
				)
				continue
		}

		if err := loadProductPrices(
			r.Context(),
			&localProduct,
		); err != nil {
			log.Printf(
				"Failed to load product prices for %s: %v",
				productID,
				err,
			)
			continue
		}

		if localProduct.Type != "merchandise" &&
			localProduct.Type != "addon" {
			continue
		}

		productType := localProduct.Type

		recommendation := map[string]interface{}{
			"id":          linkedProduct.ID,
			"name":        linkedProduct.Name,
			"description": linkedProduct.Description,
			"type":        productType,
			"images":      linkedProduct.Images,
			"sort_order":  eventLink.SortOrder,
		}

		if linkedProduct.DefaultPrice != nil {
			recommendation["price"] = map[string]interface{}{
				"id":          linkedProduct.DefaultPrice.ID,
				"unit_amount": linkedProduct.DefaultPrice.UnitAmount,
				"currency":    linkedProduct.DefaultPrice.Currency,
			}
		}

		if productType == "merchandise" {
			recommendation["reason"] = "Official event merchandise"
		} else {
			recommendation["reason"] = "Enhance your event experience"
		}

		recommendations = append(recommendations, recommendation)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"eventId":         event.ID,
		"eventName":       event.Name,
	})
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var request EventWriteRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if strings.TrimSpace(request.Name) == "" {
		http.Error(
			w,
			"Event name is required",
			http.StatusBadRequest,
		)
		return
	}

	if strings.TrimSpace(request.Slug) == "" {
		http.Error(
			w,
			"Event slug is required",
			http.StatusBadRequest,
		)
		return
	}

	if request.Capacity < 1 {
		http.Error(
			w,
			"Capacity must be greater than zero",
			http.StatusBadRequest,
		)
		return
	}

	if request.AvailableSpots < 0 ||
		request.AvailableSpots > request.Capacity {
		http.Error(
			w,
			"Available spots must be between zero and capacity",
			http.StatusBadRequest,
		)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	var existingEvent models.Event

	if err := db.
		Where("slug = ?", request.Slug).
		First(&existingEvent).
		Error; err == nil {
		http.Error(
			w,
			"An event with this slug already exists",
			http.StatusConflict,
		)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(
			w,
			"Failed to validate event slug",
			http.StatusInternalServerError,
		)
		return
	}

	stripeParams := &stripe.ProductParams{
		Name:        stripe.String(request.Name),
		Active:      stripe.Bool(request.Active),
	}

	if len(request.Images) > 0 {
		stripeParams.Images = stripe.StringSlice(request.Images)
	}

	if request.Description != "" {
		stripeParams.Description = stripe.String(request.Description)
	}

	stripeProduct, err := product.New(stripeParams)
	if err != nil {
		log.Printf("Failed to create event Stripe product: %v", err)

		http.Error(
			w,
			"Failed to create event catalog product",
			http.StatusInternalServerError,
		)
		return
	}

	event := models.Event{
		StripeProductID: stripeProduct.ID,

		Slug:            request.Slug,
		Name:            request.Name,
		Description:     request.Description,
		LongDescription: request.LongDescription,
		Images:          request.Images,

		EventDate: request.EventDate,
		Location:  request.Location,
		Venue:     request.Venue,
		Organizer: request.Organizer,

		Capacity:       request.Capacity,
		AvailableSpots: request.AvailableSpots,

		Status:   request.Status,
		Active:   request.Active,
		Featured: request.Featured,

		Tags:     request.Tags,
		Agenda:   request.Agenda,
		Includes: request.Includes,
		Sponsors: request.Sponsors,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}

		for _, priceRequest := range request.Prices {
			priceID, err := upsertEventPrice(
				tx,
				stripeProduct.ID,
				priceRequest,
			)
			if err != nil {
				return err
			}

			if err := replacePriceRequirements(
				tx,
				priceID,
				priceRequest.Requirements,
			); err != nil {
				return fmt.Errorf(
					"replace requirements for price %s: %w",
					priceID,
					err,
				)
			}
		}

		return nil
	})

	if err != nil {
		// The database write failed after the Stripe product was created.
		// This is safe cleanup for a local/pre-production project.
		if _, deleteErr := product.Del(stripeProduct.ID, nil); deleteErr != nil {
			log.Printf(
				"Failed to clean up Stripe product %s: %v",
				stripeProduct.ID,
				deleteErr,
			)
		}

		log.Printf("Failed to create event: %v", err)

		http.Error(
			w,
			"Failed to create event",
			http.StatusInternalServerError,
		)
		return
	}

	created, err := findEventByID(r.Context(), event.ID)
	if err != nil {
		http.Error(
			w,
			"Event created but could not be loaded",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    created.ID,
		"event": eventToResponse(created),
	})
}

func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]

	if eventID == "" {
		http.Error(
			w,
			"Event ID is required",
			http.StatusBadRequest,
		)
		return
	}

	var request EventWriteRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	var event models.Event

	if err := db.
		Where("id = ?", eventID).
		First(&event).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(
				w,
				"Event not found",
				http.StatusNotFound,
			)
			return
		}

		http.Error(
			w,
			"Failed to load event",
			http.StatusInternalServerError,
		)
		return
	}

	if strings.TrimSpace(request.Name) == "" {
		http.Error(
			w,
			"Event name is required",
			http.StatusBadRequest,
		)
		return
	}

	if strings.TrimSpace(request.Slug) == "" {
		http.Error(
			w,
			"Event slug is required",
			http.StatusBadRequest,
		)
		return
	}

	var slugOwner models.Event

	if err := db.
		Where("slug = ? AND id <> ?", request.Slug, eventID).
		First(&slugOwner).
		Error; err == nil {
		http.Error(
			w,
			"An event with this slug already exists",
			http.StatusConflict,
		)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(
			w,
			"Failed to validate event slug",
			http.StatusInternalServerError,
		)
		return
	}

	stripeParams := &stripe.ProductParams{
		Name:        stripe.String(request.Name),
		Active:      stripe.Bool(request.Active),
	}

	if request.Description != "" {
		stripeParams.Description = stripe.String(request.Description)
	}

	if len(request.Images) > 0 {
		stripeParams.Images = stripe.StringSlice(request.Images)
	}

	_, err := product.Update(
		event.StripeProductID,
		stripeParams,
	)
	if err != nil {
		log.Printf(
			"Failed to update event Stripe product %s: %v",
			event.StripeProductID,
			err,
		)

		http.Error(
			w,
			"Failed to update event catalog product",
			http.StatusInternalServerError,
		)
		return
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		event.Slug = request.Slug
		event.Name = request.Name
		event.Description = request.Description
		event.LongDescription = request.LongDescription
		event.Images = request.Images

		event.EventDate = request.EventDate
		event.Location = request.Location
		event.Venue = request.Venue
		event.Organizer = request.Organizer

		event.Capacity = request.Capacity
		event.AvailableSpots = request.AvailableSpots

		event.Status = request.Status
		event.Active = request.Active
		event.Featured = request.Featured

		event.Tags = request.Tags
		event.Agenda = request.Agenda
		event.Includes = request.Includes
		event.Sponsors = request.Sponsors

		if err := tx.Save(&event).Error; err != nil {
			return err
		}

		incomingPriceIDs := make(map[string]struct{})

		for _, priceRequest := range request.Prices {
			priceID, err := upsertEventPrice(
				tx,
				event.StripeProductID,
				priceRequest,
			)
			if err != nil {
				return err
			}

			if err := replacePriceRequirements(
				tx,
				priceID,
				priceRequest.Requirements,
			); err != nil {
				return fmt.Errorf(
					"replace requirements for price %s: %w",
					priceID,
					err,
				)
			}

			incomingPriceIDs[priceID] = struct{}{}
		}

		// Archive local and Stripe prices removed from the form.
		var existingPrices []models.PriceInfo

		if err := tx.
			Where(
				"stripe_product_id = ? AND active = TRUE",
				event.StripeProductID,
			).
			Find(&existingPrices).
			Error; err != nil {
			return err
		}

		for _, existingPrice := range existingPrices {
			if _, exists := incomingPriceIDs[existingPrice.ID]; exists {
				continue
			}

			if err := tx.
				Model(&models.PriceInfo{}).
				Where("id = ?", existingPrice.ID).
				Update("active", false).
				Error; err != nil {
				return err
			}

			if _, err := price.Update(
				existingPrice.ID,
				&stripe.PriceParams{
					Active: stripe.Bool(false),
				},
			); err != nil {
				return fmt.Errorf(
					"archive Stripe price %s: %w",
					existingPrice.ID,
					err,
				)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Failed to update event %s: %v", eventID, err)

		http.Error(
			w,
			"Failed to update event",
			http.StatusInternalServerError,
		)
		return
	}

	updated, err := findEventByID(r.Context(), eventID)
	if err != nil {
		http.Error(
			w,
			"Event updated but could not be loaded",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    updated.ID,
		"event": eventToResponse(updated),
	})
}
