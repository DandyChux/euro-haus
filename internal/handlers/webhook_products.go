package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/product"
)

// ProcessBundledProducts processes any products included with a tier purchase
func ProcessBundledProducts(sessionID string, customerEmail string, customerName string) error {
	db := services.GetDB()
	ctx := context.Background()

	var metadataJSON []byte
	err := db.WithContext(ctx).Raw(`
		SELECT metadata
		FROM checkout_sessions
		WHERE session_id = ?
	`, sessionID).Row().Scan(&metadataJSON)
	if err != nil {
		log.Printf("No checkout session metadata found for session %s: %v", sessionID, err)
		return nil
	}

	var sessionMetadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &sessionMetadata); err != nil {
		log.Printf("Error parsing checkout session metadata: %v", err)
		return err
	}

	tierPriceID, _ := sessionMetadata["price_id"].(string)
	if tierPriceID == "" {
		log.Printf("No tier price found for session %s", sessionID)
		return nil
	}

	var includedProducts []models.PriceIncludedProduct

	err = db.WithContext(ctx).
		Where("price_id = ?", tierPriceID).
		Order("sort_order ASC, product_id ASC").
		Find(&includedProducts).
		Error

	if err != nil {
		return fmt.Errorf(
			"load included products for price %s: %w",
			tierPriceID,
			err,
		)
	}

	processedCount := 0

	for _, included := range includedProducts {
		productParams := &stripe.ProductParams{}
		product, err := product.Get(included.ProductID, productParams)
		if err != nil {
			log.Printf(
				"Unable to retrieve included product %s: %v",
				included.ProductID,
				err,
			)
			continue
		}

		fulfillmentID := fmt.Sprintf(
			"bundled:%s:%s",
			sessionID,
			included.ProductID,
		)

		err = db.WithContext(ctx).Exec(`
			INSERT INTO fulfillments (
				id,
				session_id,
				product_id,
				product_name,
				quantity,
				customer_email,
				customer_name,
				status,
				type
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, 'pending', 'bundled'
			)
			ON CONFLICT (id) DO NOTHING
		`,
			fulfillmentID,
			sessionID,
			included.ProductID,
			product.Name,
			included.Quantity,
			customerEmail,
			customerName,
		).Error

		if err != nil {
			log.Printf(
				"Error creating fulfillment for %s: %v",
				included.ProductID,
				err,
			)
			continue
		}

		processedCount++
	}

	// if processedCount > 0 {
	// 	sendBundledProductsEmail(customerEmail, customerName, includedProducts)
	// }

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
