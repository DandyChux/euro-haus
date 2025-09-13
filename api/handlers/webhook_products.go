package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"euro-haus-api/services"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
)

// ProcessBundledProducts processes any products included with a tier purchase
func ProcessBundledProducts(sessionID string, customerEmail string, customerName string) error {
	// Get Redis client
	rdb := services.GetRedisClient()
	ctx := context.Background()

	// Get session details from Redis or Stripe
	// This should be called after a successful checkout

	// First, check if this was a tier with included products
	sessionKey := fmt.Sprintf("checkout_session:%s", sessionID)
	tierPriceID, err := rdb.HGet(ctx, sessionKey, "price_id").Result()
	if err != nil {
		log.Printf("No tier price found for session %s", sessionID)
		return nil
	}

	// Get the price details to check for included products
	priceParams := &stripe.PriceParams{}
	tierPrice, err := price.Get(tierPriceID, priceParams)
	if err != nil {
		log.Printf("Error fetching price %s: %v", tierPriceID, err)
		return err
	}

	// Check if this tier has included products
	includedProductsJSON, ok := tierPrice.Metadata["included_products"]
	if !ok || includedProductsJSON == "" {
		log.Printf("No included products for tier %s", tierPriceID)
		return nil
	}

	// Parse included products
	var includedProducts []map[string]interface{}
	if err := json.Unmarshal([]byte(includedProductsJSON), &includedProducts); err != nil {
		log.Printf("Error parsing included products: %v", err)
		return err
	}

	// Process each included product
	for _, product := range includedProducts {
		productID, _ := product["id"].(string)
		quantity, _ := product["quantity"].(float64)
		productName, _ := product["name"].(string)

		if productID == "" {
			continue
		}

		// Create fulfillment record for each included product
		fulfillmentKey := fmt.Sprintf("fulfillment:%s:%s", sessionID, productID)
		fulfillmentData := map[string]interface{}{
			"session_id":     sessionID,
			"product_id":     productID,
			"product_name":   productName,
			"quantity":       int(quantity),
			"customer_email": customerEmail,
			"customer_name":  customerName,
			"status":         "pending",
			"type":           "bundled",
			"created_at":     time.Now().Format(time.RFC3339),
		}

		if err := rdb.HSet(ctx, fulfillmentKey, fulfillmentData).Err(); err != nil {
			log.Printf("Error creating fulfillment record: %v", err)
			continue
		}

		// Add to fulfillment queue
		rdb.SAdd(ctx, "fulfillments:pending", fulfillmentKey)

		log.Printf("Created fulfillment for bundled product %s (qty: %d) for customer %s",
			productName, int(quantity), customerEmail)
	}

	// Send notification email about included products if any were processed
	if len(includedProducts) > 0 {
		sendBundledProductsEmail(customerEmail, customerName, includedProducts)
	}

	return nil
}

// sendBundledProductsEmail sends an email about bundled products
func sendBundledProductsEmail(email, name string, products []map[string]interface{}) {
	productsList := ""
	for _, p := range products {
		productName, _ := p["name"].(string)
		quantity, _ := p["quantity"].(float64)
		productsList += fmt.Sprintf("• %s (Quantity: %d)\n", productName, int(quantity))
	}

	emailHTML := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2>Your Order Includes Additional Items!</h2>

			<p>Hi %s,</p>

			<p>Great news! Your ticket purchase includes the following items:</p>

			<div style="background-color: #f5f5f5; padding: 15px; border-radius: 5px; margin: 20px 0;">
				<pre style="font-family: Arial, sans-serif; margin: 0;">%s</pre>
			</div>

			<h3>What's Next?</h3>
			<ul>
				<li>Physical items will be available for pickup at the event check-in</li>
				<li>Digital items will be sent to your email shortly</li>
				<li>Please bring your ticket confirmation to claim physical items</li>
			</ul>

			<p>If you have any questions about your included items, please contact us at info@theeurohaus.com</p>

			<p>We look forward to seeing you at the event!</p>

			<hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">

			<p style="font-size: 12px; color: #666;">
				This email was sent to confirm the additional items included with your ticket purchase.
			</p>
		</body>
		</html>
	`, name, productsList)

	msg := &services.EmailMessage{
		To:       []string{email},
		Subject:  "Your Order Includes Additional Items - Euro Haus",
		BodyHTML: emailHTML,
	}

	if err := services.SendEmail(msg); err != nil {
		log.Printf("Error sending bundled products email: %v", err)
	}
}

// GetEventAddons retrieves available add-on products for an event
func GetEventAddons(eventID string) ([]map[string]interface{}, error) {
	// This would fetch the linked products for display during checkout
	// Implementation depends on how you want to structure the checkout flow

	return []map[string]interface{}{}, nil
}
