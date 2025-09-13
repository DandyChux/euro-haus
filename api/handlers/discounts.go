package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/coupon"
	"github.com/stripe/stripe-go/v82/promotioncode"
)

// CreateCoupon creates a new coupon for discounts
func CreateCoupon(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		Name             string            `json:"name"`
		PercentOff       *float64          `json:"percent_off"`
		AmountOff        *int64            `json:"amount_off"`
		Currency         string            `json:"currency"`
		Duration         string            `json:"duration"` // "once", "repeating", or "forever"
		DurationInMonths *int64            `json:"duration_in_months"`
		MaxRedemptions   *int64            `json:"max_redemptions"`
		RedeemBy         *int64            `json:"redeem_by"`           // Unix timestamp
		AppliesTo        []string          `json:"applies_to_products"` // Product IDs
		Metadata         map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create coupon params
	params := &stripe.CouponParams{
		Name:     stripe.String(req.Name),
		Duration: stripe.String(req.Duration),
		Metadata: req.Metadata,
	}

	// Set discount type (either percent or amount, not both)
	if req.PercentOff != nil {
		params.PercentOff = stripe.Float64(*req.PercentOff)
	} else if req.AmountOff != nil {
		params.AmountOff = stripe.Int64(*req.AmountOff)
		params.Currency = stripe.String(req.Currency)
	} else {
		http.Error(w, "Either percent_off or amount_off must be specified", http.StatusBadRequest)
		return
	}

	// Set duration options
	if req.Duration == "repeating" && req.DurationInMonths != nil {
		params.DurationInMonths = stripe.Int64(*req.DurationInMonths)
	}

	// Set redemption limits
	if req.MaxRedemptions != nil {
		params.MaxRedemptions = stripe.Int64(*req.MaxRedemptions)
	}

	// Set expiration date
	if req.RedeemBy != nil {
		params.RedeemBy = stripe.Int64(*req.RedeemBy)
	}

	// Apply to specific products if specified
	if len(req.AppliesTo) > 0 {
		params.AppliesTo = &stripe.CouponAppliesToParams{
			Products: stripe.StringSlice(req.AppliesTo),
		}
	}

	// Create the coupon
	c, err := coupon.New(params)
	if err != nil {
		log.Printf("Error creating coupon: %v", err)
		http.Error(w, "Failed to create coupon", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"coupon": c,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreatePromotionCode creates a customer-facing code for a coupon
func CreatePromotionCode(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		CouponID        string            `json:"coupon_id"`
		Code            string            `json:"code"`
		MaxRedemptions  *int64            `json:"max_redemptions"`
		ExpiresAt       *int64            `json:"expires_at"` // Unix timestamp
		FirstTimeOnly   bool              `json:"first_time_only"`
		MinimumAmount   *int64            `json:"minimum_amount"`
		MinimumCurrency string            `json:"minimum_currency"`
		Metadata        map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create promotion code params
	params := &stripe.PromotionCodeParams{
		Coupon:   stripe.String(req.CouponID),
		Code:     stripe.String(req.Code),
		Metadata: req.Metadata,
	}

	// Set optional parameters
	if req.MaxRedemptions != nil {
		params.MaxRedemptions = stripe.Int64(*req.MaxRedemptions)
	}

	if req.ExpiresAt != nil {
		params.ExpiresAt = stripe.Int64(*req.ExpiresAt)
	}

	// Set restrictions
	if req.FirstTimeOnly || req.MinimumAmount != nil {
		params.Restrictions = &stripe.PromotionCodeRestrictionsParams{}

		if req.FirstTimeOnly {
			params.Restrictions.FirstTimeTransaction = stripe.Bool(true)
		}

		if req.MinimumAmount != nil {
			params.Restrictions.MinimumAmount = stripe.Int64(*req.MinimumAmount)
			params.Restrictions.MinimumAmountCurrency = stripe.String(req.MinimumCurrency)
		}
	}

	// Create the promotion code
	pc, err := promotioncode.New(params)
	if err != nil {
		log.Printf("Error creating promotion code: %v", err)
		http.Error(w, "Failed to create promotion code", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"promotion_code": pc,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListCoupons returns all coupons
func ListCoupons(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// List all coupons
	params := &stripe.CouponListParams{}
	params.Filters.AddFilter("limit", "", "100")

	iter := coupon.List(params)
	var coupons []*stripe.Coupon

	for iter.Next() {
		c := iter.Coupon()
		// Include all coupons (both valid and expired) so the admin can see everything
		coupons = append(coupons, c)
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error listing coupons: %v", err)
		http.Error(w, "Failed to list coupons", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array
	if coupons == nil {
		coupons = []*stripe.Coupon{}
	}

	response := map[string]interface{}{
		"coupons": coupons,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DeleteCoupon marks a coupon as deleted
func DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get coupon ID from URL
	vars := mux.Vars(r)
	couponID := vars["id"]

	if couponID == "" {
		http.Error(w, "Coupon ID is required", http.StatusBadRequest)
		return
	}

	// Delete the coupon
	_, err := coupon.Del(couponID, nil)
	if err != nil {
		log.Printf("Error deleting coupon %s: %v", couponID, err)
		http.Error(w, "Failed to delete coupon", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Coupon deleted successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ValidatePromotionCode validates a promotion code (public endpoint)
func ValidatePromotionCode(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// List promotion codes with the given code
	params := &stripe.PromotionCodeListParams{
		Code: stripe.String(req.Code),
	}
	params.Filters.AddFilter("limit", "", "1")

	iter := promotioncode.List(params)

	if !iter.Next() {
		response := map[string]interface{}{
			"valid": false,
			"error": "Invalid promotion code",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	pc := iter.PromotionCode()

	// Check if the promotion code is active
	if !pc.Active {
		response := map[string]interface{}{
			"valid": false,
			"error": "Promotion code is not active",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if it has expired
	if pc.ExpiresAt > 0 && pc.ExpiresAt < time.Now().Unix() {
		response := map[string]interface{}{
			"valid": false,
			"error": "Promotion code has expired",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check redemption limits
	if pc.MaxRedemptions > 0 && pc.TimesRedeemed >= pc.MaxRedemptions {
		response := map[string]interface{}{
			"valid": false,
			"error": "Promotion code has reached its redemption limit",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the associated coupon
	c, err := coupon.Get(pc.Coupon.ID, nil)
	if err != nil {
		log.Printf("Error fetching coupon: %v", err)
		http.Error(w, "Failed to validate promotion code", http.StatusInternalServerError)
		return
	}

	// Build response with discount details
	response := map[string]interface{}{
		"valid": true,
		"promotion_code": map[string]interface{}{
			"id":          pc.ID,
			"code":        pc.Code,
			"coupon_id":   pc.Coupon.ID,
			"coupon_name": c.Name,
		},
		"discount": map[string]interface{}{
			"percent_off": c.PercentOff,
			"amount_off":  c.AmountOff,
			"currency":    c.Currency,
		},
		"restrictions": pc.Restrictions,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
