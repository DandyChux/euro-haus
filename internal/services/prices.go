package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProductPriceInput struct {
	ID                 string
	UnitAmount         int64
	Currency           string
	Nickname           string
	Description        string
	Active             bool
	Features           []string
	IsMostPopular      bool
	RequiresApproval   bool
	RequiresSubmission bool
	Quantity           int
	StockQuantity      *int
	Size               string
	Color              string
}

func SyncProductPrices(
	ctx context.Context,
	productID string,
	inputs []ProductPriceInput,
) error {
	if strings.TrimSpace(productID) == "" {
		return errors.New("product ID is empty")
	}

	db := GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}

	// Stripe prices are immutable for amount and currency. We can reuse a
	// price only when both values remain unchanged.
	iter := price.List(&stripe.PriceListParams{
		Product: stripe.String(productID),
	})

	existingStripePrices := make(map[string]*stripe.Price)

	for iter.Next() {
		stripePrice := iter.Price()
		if stripePrice == nil || stripePrice.ID == "" {
			continue
		}

		existingStripePrices[stripePrice.ID] = stripePrice
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf(
			"list Stripe prices for product %s: %w",
			productID,
			err,
		)
	}

	// Load existing local prices once. This lets us preserve local-only
	// fields when a Stripe price is reused.
	var existingLocalPrices []models.PriceInfo

	if err := db.WithContext(ctx).
		Where("stripe_product_id = ?", productID).
		Find(&existingLocalPrices).
		Error; err != nil {
		return fmt.Errorf(
			"load local prices for product %s: %w",
			productID,
			err,
		)
	}

	existingLocalByID := make(map[string]models.PriceInfo, len(existingLocalPrices))

	for _, localPrice := range existingLocalPrices {
		existingLocalByID[localPrice.ID] = localPrice
	}

	seenStripePriceIDs := make(map[string]struct{}, len(inputs))
	resolvedPrices := make([]models.PriceInfo, 0, len(inputs))

	defaultStripePriceID := ""

	for index, input := range inputs {
		input.Currency = strings.ToLower(strings.TrimSpace(input.Currency))
		input.Nickname = strings.TrimSpace(input.Nickname)
		input.Description = strings.TrimSpace(input.Description)
		input.Size = strings.TrimSpace(input.Size)
		input.Color = strings.TrimSpace(input.Color)

		if input.UnitAmount < 0 {
			return fmt.Errorf(
				"price %d has an invalid amount: %d",
				index,
				input.UnitAmount,
			)
		}

		if input.Currency == "" {
			input.Currency = "usd"
		}

		if len(input.Currency) != 3 {
			return fmt.Errorf(
				"price %d has an invalid currency: %q",
				index,
				input.Currency,
			)
		}

		priceID := strings.TrimSpace(input.ID)
		stripePrice := (*stripe.Price)(nil)

		// Reuse an existing Stripe price only when the immutable Stripe
		// fields still match.
		if priceID != "" {
			existing, ok := existingStripePrices[priceID]
			if !ok {
				return fmt.Errorf(
					"price %s does not belong to product %s",
					priceID,
					productID,
				)
			}

			if existing.UnitAmount == input.UnitAmount &&
				strings.EqualFold(
					string(existing.Currency),
					input.Currency,
				) {
				stripePrice = existing
			}
		}

		// If the amount/currency changed, or this is a new price, create
		// a new Stripe price.
		if stripePrice == nil {
			params := &stripe.PriceParams{
				Product:    stripe.String(productID),
				UnitAmount: stripe.Int64(input.UnitAmount),
				Currency:   stripe.String(input.Currency),
				Active:     stripe.Bool(input.Active),
			}

			if input.Nickname != "" {
				params.Nickname = stripe.String(input.Nickname)
			}

			created, err := price.New(params)
			if err != nil {
				return fmt.Errorf(
					"create Stripe price for product %s at index %d: %w",
					productID,
					index,
					err,
				)
			}

			stripePrice = created
			priceID = created.ID
		} else {
			// Nickname and active state are mutable, so keep Stripe's
			// representation aligned with the submitted form.
			updateParams := &stripe.PriceParams{
				Active: stripe.Bool(input.Active),
			}

			if input.Nickname != "" {
				updateParams.Nickname = stripe.String(input.Nickname)
			}

			if _, err := price.Update(priceID, updateParams); err != nil {
				return fmt.Errorf(
					"update Stripe price %s: %w",
					priceID,
					err,
				)
			}
		}

		seenStripePriceIDs[priceID] = struct{}{}

		if index == 0 {
			defaultStripePriceID = priceID
		}

		featuresJSON, err := json.Marshal(input.Features)
		if err != nil {
			return fmt.Errorf(
				"marshal features for price %s: %w",
				priceID,
				err,
			)
		}

		localPrice := models.PriceInfo{
			ID:                 priceID,
			StripeProductID:   productID,
			UnitAmount:        input.UnitAmount,
			Currency:          input.Currency,
			Nickname:          input.Nickname,
			Description:       input.Description,
			Active:             input.Active,
			Features:          datatypes.JSON(featuresJSON),
			IsDefault:         index == 0,
			IsMostPopular:     input.IsMostPopular,
			RequiresApproval:  input.RequiresApproval,
			RequiresSubmission: input.RequiresSubmission,
			Quantity:           input.Quantity,
			StockQuantity:     input.StockQuantity,
			Size:              input.Size,
			Color:             input.Color,
		}

		// Preserve fields that are not part of ProductPriceInput when
		// reusing an existing local price.
		if existing, ok := existingLocalByID[priceID]; ok {
			localPrice.CreatedAt = existing.CreatedAt
			localPrice.IncludedProductLinks = existing.IncludedProductLinks
			localPrice.Requirements = existing.Requirements
		}

		resolvedPrices = append(resolvedPrices, localPrice)
	}

	// Make the submitted first price the Stripe default. Stripe products
	// cannot have a default price when no prices were submitted, so only
	// update it when one exists.
	if defaultStripePriceID != "" {
		if _, err := product.Update(productID, &stripe.ProductParams{
			DefaultPrice: stripe.String(defaultStripePriceID),
		}); err != nil {
			return fmt.Errorf(
				"set default Stripe price %s for product %s: %w",
				defaultStripePriceID,
				productID,
				err,
			)
		}
	}

	// Archive Stripe prices omitted by the submitted collection.
	for existingID := range existingStripePrices {
		if _, seen := seenStripePriceIDs[existingID]; seen {
			continue
		}

		if _, err := price.Update(existingID, &stripe.PriceParams{
			Active: stripe.Bool(false),
		}); err != nil {
			return fmt.Errorf(
				"archive removed Stripe price %s: %w",
				existingID,
				err,
			)
		}
	}

	// Persist local prices atomically. Clear defaults first so only the
	// submitted first price is marked default.
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.PriceInfo{}).
			Where("stripe_product_id = ?", productID).
			Update("is_default", false).
			Error; err != nil {
			return fmt.Errorf("clear existing price defaults: %w", err)
		}

		for _, localPrice := range resolvedPrices {
			if err := tx.Save(&localPrice).Error; err != nil {
				return fmt.Errorf(
					"save local price %s: %w",
					localPrice.ID,
					err,
				)
			}
		}

		// Mark removed local prices inactive. Do not delete them because
		// they may be referenced by orders, tickets, or submissions.
		if len(seenStripePriceIDs) == 0 {
			if err := tx.
				Model(&models.PriceInfo{}).
				Where("stripe_product_id = ?", productID).
				Update("active", false).
				Error; err != nil {
				return fmt.Errorf(
					"archive all removed local prices: %w",
					err,
				)
			}
		} else {
			for existingID := range existingLocalByID {
				if _, seen := seenStripePriceIDs[existingID]; seen {
					continue
				}

				if err := tx.
					Model(&models.PriceInfo{}).
					Where(
						"id = ? AND stripe_product_id = ?",
						existingID,
						productID,
					).
					Updates(map[string]interface{}{
						"active":     false,
						"is_default": false,
					}).
					Error; err != nil {
					return fmt.Errorf(
						"archive removed local price %s: %w",
						existingID,
						err,
					)
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
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
