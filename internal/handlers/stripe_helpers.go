package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dandychux/euro-haus/internal/services"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

// paymentIntentIDFromSession supports both expanded and unexpanded Stripe
// Checkout Session payment_intent values.
func paymentIntentIDFromSession(sess *stripe.CheckoutSession) string {
	if sess == nil || sess.PaymentIntent == nil {
		return ""
	}

	return strings.TrimSpace(sess.PaymentIntent.ID)
}

// persistCheckoutState writes all Stripe identifiers received from Stripe.
// This function is intentionally idempotent because webhook delivery can be
// retried and events can arrive out of order.
func persistCheckoutState(
	ctx context.Context,
	submissionID string,
	checkoutSessionID string,
	paymentIntentID string,
	checkoutCompleted bool,
	paymentSucceeded bool,
	paymentSucceededAt bool,
) error {
	if strings.TrimSpace(submissionID) == "" {
		return fmt.Errorf("submission ID is required")
	}

	if strings.TrimSpace(checkoutSessionID) == "" &&
		strings.TrimSpace(paymentIntentID) == "" {
		return fmt.Errorf("checkout session ID or payment intent ID is required")
	}

	db := services.GetDB()
	if db == nil {
		return fmt.Errorf("database is unavailable")
	}

	query := `
		UPDATE vehicle_submissions
		SET
			checkout_session_id = COALESCE(NULLIF(?, ''), checkout_session_id),
			payment_intent_id = COALESCE(NULLIF(?, ''), payment_intent_id),
			checkout_completed = CASE
				WHEN ? THEN TRUE
				ELSE checkout_completed
			END,
			checkout_completed_at = CASE
				WHEN ? THEN COALESCE(checkout_completed_at, NOW())
				ELSE checkout_completed_at
			END,
			payment_succeeded_before_approval = CASE
				WHEN ? THEN TRUE
				ELSE payment_succeeded_before_approval
			END,
			payment_succeeded_at = CASE
				WHEN ? THEN COALESCE(payment_succeeded_at, NOW())
				ELSE payment_succeeded_at
			END
		WHERE id = ?
	`

	result := db.WithContext(ctx).Exec(
		query,
		strings.TrimSpace(checkoutSessionID),
		strings.TrimSpace(paymentIntentID),
		checkoutCompleted,
		checkoutCompleted,
		paymentSucceeded,
		paymentSucceededAt,
		submissionID,
	)

	if result.Error != nil {
		return fmt.Errorf("persist checkout state: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("submission %s was not found", submissionID)
	}

	return nil
}

// retrieveCheckoutSessionPaymentIntent retrieves the PaymentIntent associated
// with a Checkout Session, regardless of whether Stripe returned the intent
// expanded or only as an ID.
func retrieveCheckoutSessionPaymentIntent(
	ctx context.Context,
	checkoutSessionID string,
) (*stripe.PaymentIntent, error) {
	checkoutSessionID = strings.TrimSpace(checkoutSessionID)
	if checkoutSessionID == "" {
		return nil, fmt.Errorf("checkout session ID is required")
	}

	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("payment_intent")

	sess, err := session.Get(checkoutSessionID, params)
	if err != nil {
		return nil, fmt.Errorf("retrieve checkout session: %w", err)
	}

	paymentIntentID := paymentIntentIDFromSession(sess)
	if paymentIntentID == "" {
		return nil, fmt.Errorf(
			"checkout session %s has no payment intent",
			checkoutSessionID,
		)
	}

	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("retrieve payment intent %s: %w", paymentIntentID, err)
	}

	return pi, nil
}

// persistCheckoutSessionAfterCreation must be called after Stripe creates a
// session. A successful Stripe response without a successful DB update is not
// a valid checkout creation result.
func persistCheckoutSessionAfterCreation(
	ctx context.Context,
	submissionID string,
	checkoutSessionID string,
	priceID string,
	promotionCode string,
	requiresApproval bool,
) error {
	db := services.GetDB()
	if db == nil {
		return fmt.Errorf("database is unavailable")
	}

	result := db.WithContext(ctx).Exec(`
		UPDATE vehicle_submissions
		SET
			checkout_session_id = ?,
			price_id = NULLIF(?, ''),
			requires_approval = ?,
			checkout_created_at = NOW(),
			promotion_code = NULLIF(?, '')
		WHERE id = ?
	`,
		strings.TrimSpace(checkoutSessionID),
		strings.TrimSpace(priceID),
		requiresApproval,
		strings.TrimSpace(promotionCode),
		submissionID,
	)

	if result.Error != nil {
		return fmt.Errorf("persist checkout session: %w", result.Error)
	}

	if result.RowsAffected != 1 {
		return fmt.Errorf("submission %s was not found", submissionID)
	}

	return nil
}

// persistPaymentIntentID stores the PaymentIntent as soon as it becomes known.
// This is useful for both webhook and approval recovery paths.
func persistPaymentIntentID(
	ctx context.Context,
	submissionID string,
	paymentIntentID string,
) error {
	paymentIntentID = strings.TrimSpace(paymentIntentID)
	if paymentIntentID == "" {
		return fmt.Errorf("payment intent ID is required")
	}

	db := services.GetDB()
	if db == nil {
		return fmt.Errorf("database is unavailable")
	}

	result := db.WithContext(ctx).Exec(`
		UPDATE vehicle_submissions
		SET payment_intent_id = ?
		WHERE id = ?
	`, paymentIntentID, submissionID)

	if result.Error != nil {
		return fmt.Errorf("persist payment intent ID: %w", result.Error)
	}

	if result.RowsAffected != 1 {
		return fmt.Errorf("submission %s was not found", submissionID)
	}

	return nil
}

func logPersistenceFailure(operation string, submissionID string, err error) {
	if err != nil {
		log.Printf(
			"%s failed for submission %s: %v",
			operation,
			submissionID,
			err,
		)
	}
}
