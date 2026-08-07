package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dandychux/euro-haus/internal/services"

	"github.com/gorilla/mux"
)

type FulfillmentStatus string

const (
	FulfillmentPending    FulfillmentStatus = "pending"
	FulfillmentProcessing FulfillmentStatus = "processing"
	FulfillmentShipped    FulfillmentStatus = "shipped"
	FulfillmentDelivered  FulfillmentStatus = "delivered"
	FulfillmentCancelled  FulfillmentStatus = "cancelled"
)

type Fulfillment struct {
	ID              string            `json:"id"`
	OrderID         string            `json:"order_id"`
	ProductID       string            `json:"product_id"`
	ProductName     string            `json:"product_name"`
	Quantity        int               `json:"quantity"`
	CustomerEmail   string            `json:"customer_email"`
	CustomerName    string            `json:"customer_name"`
	ShippingAddress string            `json:"shipping_address"`
	Status          FulfillmentStatus `json:"status"`
	Type            string            `json:"type"`
	TrackingNumber  string            `json:"tracking_number,omitempty"`
	TrackingCarrier string            `json:"tracking_carrier,omitempty"`
	Notes           string            `json:"notes,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	ShippedAt       *time.Time        `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time        `json:"delivered_at,omitempty"`
}

func GetPendingFulfillments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	db := services.GetDB()
	rows, err := db.WithContext(r.Context()).Raw(`
		SELECT id, session_id, COALESCE(product_id, ''), COALESCE(product_name, ''),
		       quantity, customer_email, COALESCE(customer_name, ''),
		       COALESCE(shipping_address, ''), status, type,
		       COALESCE(tracking_number, ''), COALESCE(tracking_carrier, ''),
		       COALESCE(notes, ''), created_at, updated_at, shipped_at, delivered_at
		FROM fulfillments
		WHERE status = 'pending'
		ORDER BY created_at ASC
	`).Rows()
	if err != nil {
		http.Error(w, "Failed to retrieve fulfillments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fulfillments := []Fulfillment{}
	for rows.Next() {
		var f Fulfillment
		var status string
		if err := rows.Scan(
			&f.ID, &f.OrderID, &f.ProductID, &f.ProductName,
			&f.Quantity, &f.CustomerEmail, &f.CustomerName,
			&f.ShippingAddress, &status, &f.Type,
			&f.TrackingNumber, &f.TrackingCarrier,
			&f.Notes, &f.CreatedAt, &f.UpdatedAt, &f.ShippedAt, &f.DeliveredAt,
		); err != nil {
			continue
		}
		f.Status = FulfillmentStatus(status)
		fulfillments = append(fulfillments, f)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"fulfillments": fulfillments,
		"total":        len(fulfillments),
	})
}

func UpdateFulfillmentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := mux.Vars(r)["id"]

	var req struct {
		Status          string `json:"status"`
		TrackingNumber  string `json:"trackingNumber,omitempty"`
		TrackingCarrier string `json:"trackingCarrier,omitempty"`
		Notes           string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db := services.GetDB()
	ctx := r.Context()

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var existing Fulfillment
	var status string
	err := tx.Raw(`
		SELECT id, session_id, COALESCE(product_id, ''), COALESCE(product_name, ''),
		       quantity, customer_email, COALESCE(customer_name, ''), status, type
		FROM fulfillments WHERE id = ? FOR UPDATE
	`, id).Row().Scan(
		&existing.ID, &existing.OrderID, &existing.ProductID, &existing.ProductName,
		&existing.Quantity, &existing.CustomerEmail, &existing.CustomerName, &status, &existing.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Fulfillment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to load fulfillment", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	var shippedAt, deliveredAt *time.Time
	switch FulfillmentStatus(req.Status) {
	case FulfillmentShipped:
		shippedAt = &now
	case FulfillmentDelivered:
		deliveredAt = &now
	}

	err = tx.Exec(`
		UPDATE fulfillments SET
			status           = ?,
			tracking_number  = COALESCE(NULLIF(?, ''), tracking_number),
			tracking_carrier = COALESCE(NULLIF(?, ''), tracking_carrier),
			notes            = COALESCE(NULLIF(?, ''), notes),
			shipped_at       = COALESCE(?, shipped_at),
			delivered_at     = COALESCE(?, delivered_at)
		WHERE id = ?
	`, req.Status, req.TrackingNumber, req.TrackingCarrier, req.Notes, shippedAt, deliveredAt, id).Error
	if err != nil {
		http.Error(w, "Failed to update fulfillment", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit", http.StatusInternalServerError)
		return
	}

	if FulfillmentStatus(req.Status) == FulfillmentShipped {
		sendShippingNotification(map[string]string{
			"customer_name":  existing.CustomerName,
			"customer_email": existing.CustomerEmail,
			"product_name":   existing.ProductName,
			"quantity":       fmt.Sprintf("%d", existing.Quantity),
			"session_id":     existing.OrderID,
			"order_id":       existing.OrderID,
		}, req.TrackingNumber, req.TrackingCarrier)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Fulfillment %s updated to %s", id, req.Status),
	})
}

// Helper function to send shipping notification
func sendShippingNotification(fulfillmentData map[string]string, trackingNumber, carrier string) {
	emailData := map[string]interface{}{
		"CustomerName":    fulfillmentData["customer_name"],
		"ProductName":     fulfillmentData["product_name"],
		"Quantity":        fulfillmentData["quantity"],
		"TrackingNumber":  trackingNumber,
		"TrackingCarrier": carrier,
		"OrderID":         fulfillmentData["session_id"],
	}

	emailHTML := fmt.Sprintf(`
        <html>
        <body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
            <h2>Your Order Has Shipped!</h2>

            <p>Hi %s,</p>

            <p>Great news! Your order has been shipped and is on its way to you.</p>

            <div style="background-color: #f5f5f5; padding: 15px; border-radius: 5px; margin: 20px 0;">
                <h3>Shipment Details</h3>
                <p><strong>Item:</strong> %s (Quantity: %s)</p>
                <p><strong>Tracking Number:</strong> %s</p>
                <p><strong>Carrier:</strong> %s</p>
            </div>

            <p>You can track your package using the tracking number above.</p>

            <p>If you have any questions about your order, please contact us at info@theeurohaus.com</p>

            <p>Thank you for your order!</p>

            <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">

            <p style="font-size: 12px; color: #666;">
                Order ID: %s<br>
                Euro Haus Events - Premium Automotive Experiences
            </p>
        </body>
        </html>
    `, emailData["customer_name"], emailData["product_name"],
		emailData["quantity"], trackingNumber, carrier, fulfillmentData["order_id"])

	msg := &services.EmailMessage{
		To:       []string{fulfillmentData["customer_email"]},
		Subject:  "Your Order Has Shipped - Euro Haus",
		BodyHTML: emailHTML,
	}

	if err := services.QueueEmail(
		context.Background(),
		"",
		msg,
	); err != nil {
		log.Printf(
			"Failed to queue shipping notification for order %s: %v",
			fulfillmentData["order_id"],
			err,
		)
	}

}
