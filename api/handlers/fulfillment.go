package handlers

import (
	"context"
	"encoding/json"
	"euro-haus-api/services"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	OrderID         string            `json:"orderId"`
	ProductID       string            `json:"productId"`
	ProductName     string            `json:"productName"`
	Quantity        int               `json:"quantity"`
	CustomerEmail   string            `json:"customerEmail"`
	CustomerName    string            `json:"customerName"`
	ShippingAddress string            `json:"shippingAddress"`
	Status          FulfillmentStatus `json:"status"`
	Type            string            `json:"type"` // "included" or "purchased"
	TrackingNumber  string            `json:"trackingNumber,omitempty"`
	TrackingCarrier string            `json:"trackingCarrier,omitempty"`
	Notes           string            `json:"notes,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	ShippedAt       *time.Time        `json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time        `json:"deliveredAt,omitempty"`
}

// GetPendingFulfillments retrieves all pending fulfillments
func GetPendingFulfillments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get all pending fulfillment keys
	pendingKeys, err := rdb.SMembers(ctx, "fulfillments:pending").Result()
	if err != nil {
		http.Error(w, "Failed to retrieve fulfillments", http.StatusInternalServerError)
		return
	}

	fulfillments := []Fulfillment{}

	for _, key := range pendingKeys {
		data, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}

		fulfillment := Fulfillment{
			ID:              data["id"],
			OrderID:         data["session_id"],
			ProductID:       data["product_id"],
			ProductName:     data["product_name"],
			CustomerEmail:   data["customer_email"],
			CustomerName:    data["customer_name"],
			ShippingAddress: data["shipping_address"],
			Status:          FulfillmentStatus(data["status"]),
			Type:            data["type"],
			Notes:           data["notes"],
		}

		// Parse quantity
		if qty, err := strconv.Atoi(data["quantity"]); err == nil {
			fulfillment.Quantity = qty
		}

		// Parse timestamps
		if createdAt, err := time.Parse(time.RFC3339, data["created_at"]); err == nil {
			fulfillment.CreatedAt = createdAt
		}

		fulfillments = append(fulfillments, fulfillment)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"fulfillments": fulfillments,
		"total":        len(fulfillments),
	})
}

// UpdateFulfillmentStatus updates the status of a fulfillment
func UpdateFulfillmentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	fulfillmentID := vars["id"]

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

	rdb := services.GetRedisClient()
	ctx := context.Background()

	fulfillmentKey := fmt.Sprintf("fulfillment:%s", fulfillmentID)

	// Get existing fulfillment
	exists, err := rdb.Exists(ctx, fulfillmentKey).Result()
	if err != nil || exists == 0 {
		http.Error(w, "Fulfillment not found", http.StatusNotFound)
		return
	}

	// Update fulfillment
	updates := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	if req.TrackingNumber != "" {
		updates["tracking_number"] = req.TrackingNumber
	}

	if req.TrackingCarrier != "" {
		updates["tracking_carrier"] = req.TrackingCarrier
	}

	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	// Update status-specific timestamps
	switch FulfillmentStatus(req.Status) {
	case FulfillmentShipped:
		updates["shipped_at"] = time.Now().Format(time.RFC3339)

		// Send shipping notification email
		fulfillmentData, _ := rdb.HGetAll(ctx, fulfillmentKey).Result()
		sendShippingNotification(fulfillmentData, req.TrackingNumber, req.TrackingCarrier)

	case FulfillmentDelivered:
		updates["delivered_at"] = time.Now().Format(time.RFC3339)
	}

	// Update in Redis
	if err := rdb.HSet(ctx, fulfillmentKey, updates).Err(); err != nil {
		http.Error(w, "Failed to update fulfillment", http.StatusInternalServerError)
		return
	}

	// Update fulfillment sets
	oldStatus, _ := rdb.HGet(ctx, fulfillmentKey, "status").Result()

	if oldStatus == "pending" {
		rdb.SRem(ctx, "fulfillments:pending", fulfillmentKey)
	}

	if req.Status == "shipped" {
		rdb.SAdd(ctx, "fulfillments:shipped", fulfillmentKey)
	} else if req.Status == "delivered" {
		rdb.SRem(ctx, "fulfillments:shipped", fulfillmentKey)
		rdb.SAdd(ctx, "fulfillments:delivered", fulfillmentKey)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Fulfillment %s updated to %s", fulfillmentID, req.Status),
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

	services.SendEmail(msg)
}
