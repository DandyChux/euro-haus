package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

func main() {
	days := flag.Int("days", 35, "number of days to scan backwards")
	dryRun := flag.Bool("dry-run", false, "report imports without writing them")
	flag.Parse()

	if *days < 1 {
		log.Fatal("days must be greater than zero")
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		log.Fatal("STRIPE_SECRET_KEY environment variable not set")
	}

	services.InitDB()
	defer closeDatabase()

	ctx := context.Background()
	cutoff := time.Now().Add(-time.Duration(*days) * 24 * time.Hour)
	params := &stripe.CheckoutSessionListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)},
		CreatedRange: &stripe.RangeQueryParams{
			GreaterThanOrEqual: cutoff.Unix(),
		},
	}
	params.AddExpand("data.line_items")
	params.AddExpand("data.customer_details")
	params.AddExpand("data.shipping_cost.shipping_rate")

	var scanned, imported, skipped, existing, failed int
	iter := session.List(params)
	for iter.Next() {
		scanned++
		sess := iter.CheckoutSession()
		if sess == nil || sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			skipped++
			continue
		}

		if eventID := strings.TrimSpace(sess.Metadata["event_id"]); eventID != "" {
			log.Printf("skip event session=%s event_id=%s", sess.ID, eventID)
			skipped++
			continue
		}

		items := sess.LineItems
		if items == nil || len(items.Data) == 0 {
			log.Printf("skip session=%s: no line items returned", sess.ID)
			failed++
			continue
		}

		allEventProducts := true
		for _, item := range items.Data {
			if item == nil || item.Price == nil || item.Price.Product == nil {
				allEventProducts = false
				break
			}
			var eventCount int64
			if err := services.GetDB().WithContext(ctx).Model(&models.Event{}).
				Where("stripe_product_id = ?", item.Price.Product.ID).
				Count(&eventCount).Error; err != nil {
				log.Printf("check event product session=%s product=%s: %v", sess.ID, item.Price.Product.ID, err)
				failed++
				allEventProducts = false
				break
			}
			if eventCount == 0 {
				allEventProducts = false
			}
		}
		if allEventProducts {
			log.Printf("skip event session=%s: all line items belong to events", sess.ID)
			skipped++
			continue
		}

		for _, item := range items.Data {
			if item == nil || item.Price == nil || item.Price.Product == nil {
				log.Printf("skip session=%s: line item is missing product information", sess.ID)
				failed++
				continue
			}

			productID := item.Price.Product.ID
			productName := item.Description
			if productName == "" {
				productName = item.Price.Product.Name
			}
			quantity := int(item.Quantity)
			if quantity < 1 {
				quantity = 1
			}

			var count int64
			if err := services.GetDB().WithContext(ctx).Model(&models.Fulfillment{}).
				Where("session_id = ? AND product_id = ?", sess.ID, productID).
				Count(&count).Error; err != nil {
				log.Printf("check existing session=%s product=%s: %v", sess.ID, productID, err)
				failed++
				continue
			}
			if count > 0 {
				existing++
				continue
			}

			fulfillment := models.Fulfillment{
				ID:              fmt.Sprintf("stripe_%s_%s", sess.ID, productID),
				SessionID:       sess.ID,
				ProductID:       productID,
				ProductName:     productName,
				PriceNickname:   item.Price.Nickname,
				CustomerEmail:   customerEmail(sess),
				CustomerName:    customerName(sess),
				ShippingAddress: shippingAddress(sess),
				Quantity:        quantity,
				Status:          "pending",
				Type:            "purchased",
				Notes:           "Imported from Stripe Checkout Session",
			}

			if *dryRun {
				log.Printf("would import session=%s product=%s quantity=%d email=%s", sess.ID, productID, quantity, fulfillment.CustomerEmail)
				imported++
				continue
			}

			if err := services.GetDB().WithContext(ctx).Create(&fulfillment).Error; err != nil {
				log.Printf("insert session=%s product=%s: %v", sess.ID, productID, err)
				failed++
				continue
			}
			imported++
			log.Printf("imported session=%s product=%s quantity=%d", sess.ID, productID, quantity)
		}
	}
	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	log.Printf("complete: scanned=%d imported=%d existing=%d skipped=%d failed=%d dry_run=%t", scanned, imported, existing, skipped, failed, *dryRun)
}

func customerEmail(sess *stripe.CheckoutSession) string {
	if sess.CustomerDetails != nil && sess.CustomerDetails.Email != "" {
		return sess.CustomerDetails.Email
	}
	return sess.CustomerEmail
}

func customerName(sess *stripe.CheckoutSession) string {
	if sess.CustomerDetails != nil {
		return sess.CustomerDetails.Name
	}
	return ""
}

func shippingAddress(sess *stripe.CheckoutSession) string {
	if sess.CustomerDetails == nil || sess.CustomerDetails.Address == nil {
		return ""
	}
	address := sess.CustomerDetails.Address
	parts := []string{address.Line1, address.Line2, address.City, address.State, address.PostalCode, address.Country}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			result = append(result, strings.TrimSpace(part))
		}
	}
	return strings.Join(result, ", ")
}

func closeDatabase() {
	db, err := services.GetDB().DB()
	if err != nil {
		return
	}
	if err := db.Close(); err != nil {
		log.Printf("close database: %v", err)
	}
}
