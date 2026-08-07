package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SyncStripeProductPrices(
	ctx context.Context,
	stripeProductID string,
) error {
	params := &stripe.ProductParams{}
	params.AddExpand("default_price")

	db := GetDB()

	stripeProduct, err := product.Get(stripeProductID, params)
	if err != nil {
		return fmt.Errorf("retrieve Stripe product %s: %w", stripeProductID, err)
	}

	defaultPriceID := ""
	if stripeProduct.DefaultPrice != nil {
		defaultPriceID = stripeProduct.DefaultPrice.ID
	}

	iter := price.List(&stripe.PriceListParams{
		Product: stripe.String(stripeProductID),
	})

	prices := make([]models.PriceInfo, 0)

	for iter.Next() {
		stripePrice := iter.Price()

		var existing models.PriceInfo

		err := db.
			Where("id = ?", stripePrice.ID).
			First(&existing).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			existing = models.PriceInfo{
				ID:              stripePrice.ID,
				StripeProductID: stripeProductID,
				Features:        datatypes.JSON([]byte("[]")),
			}
		} else if err != nil {
			return fmt.Errorf(
				"load local price %s: %w",
				stripePrice.ID,
				err,
			)
		}

		existing.StripeProductID = stripeProductID
		existing.UnitAmount = stripePrice.UnitAmount
		existing.Currency = string(stripePrice.Currency)
		existing.Nickname = stripePrice.Nickname
		existing.Active = stripePrice.Active
		existing.IsDefault =
			defaultPriceID != "" &&
			stripePrice.ID == defaultPriceID

		if existing.Features == nil {
			existing.Features = datatypes.JSON([]byte("[]"))
		}

		prices = append(prices, existing)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("list Stripe prices for %s: %w", stripeProductID, err)
	}

	if len(prices) == 1 {
		prices[0].IsDefault = true
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear the previous default before setting the current one.
		if err := tx.
			Model(&models.PriceInfo{}).
			Where("stripe_product_id = ?", stripeProductID).
			Update("is_default", false).
			Error; err != nil {
			return err
		}

		for _, priceInfo := range prices {
			if err := tx.Save(&priceInfo).Error; err != nil {
				return fmt.Errorf(
					"save Stripe price %s: %w",
					priceInfo.ID,
					err,
				)
			}
		}

		return nil
	})
}

func GetSubmissionPayment(paymentIntentID string) (*stripe.PaymentIntent, error) {
	if paymentIntentID == "" {
		return nil, fmt.Errorf("payment intent ID is empty")
	}

	paymentIntentParams := &stripe.PaymentIntentParams{}

	paymentIntent, err := paymentintent.Get(paymentIntentID, paymentIntentParams)
	if err != nil {
		return nil, fmt.Errorf("get payment intent for Stripe price %s: %w", paymentIntentID, err)
	}

	return paymentIntent, nil
}
