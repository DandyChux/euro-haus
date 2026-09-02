package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)


type ProductResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	Type        string   `json:"type"`

	Price    int64  `json:"price"`
	Currency string `json:"currency"`

	CompareAtPrice *int64 `json:"compare_at_price"`

	IsNew    bool `json:"is_new"`
	InStock bool `json:"in_stock"`
	Featured bool `json:"featured"`

	Category    string   `json:"category"`
	Subcategory string   `json:"subcategory"`
	Tags        []string `json:"tags"`
	MaxQuantity *int     `json:"max_quantity"`

	Active       bool         `json:"active"`
	DefaultPrice *models.PriceInfo `json:"default_price,omitempty"`
	Prices []models.PriceInfo `json:"prices"`

	BundleItems  []BundleItemResponse `json:"bundle_items,omitempty"`
	DiscountType string               `json:"discount_type,omitempty"`
	DiscountValue float64             `json:"discount_value,omitempty"`

	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

type BundleItemResponse struct {
	ProductID string  `json:"productId"`
	ProductName string `json:"productName"`
	Quantity  int     `json:"quantity"`
	Price     int64   `json:"price"`
}


func GetProducts(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	query := services.GetDB().
		WithContext(r.Context()).
		Preload("BundleItems")

	if !includeInactive {
		query = query.Where("active = ?", true)
	}

	var products []models.Product

	if err := query.
		Order("created_at DESC").
		Find(&products).
		Error; err != nil {
		log.Printf("Error loading products from database: %v", err)
		http.Error(
			w,
			"Failed to load products",
			http.StatusInternalServerError,
		)
		return
	}

	responses := make([]ProductResponse, 0, len(products))

	for _, product := range products {
		if err := loadProductPrices(r.Context(), &product); err != nil {
			log.Printf(
				"Error loading prices for product %s: %v",
				product.ID,
				err,
			)
			http.Error(
				w,
				"Failed to load product prices",
				http.StatusInternalServerError,
			)
			return
		}

		bundleItems, err := loadBundleItems(r.Context(), &product)
		if err != nil {
			log.Printf(
				"Error loading bundle items for product %s: %v",
				product.ID,
				err,
			)
			http.Error(
				w,
				"Failed to load bundle items",
				http.StatusInternalServerError,
			)
			return
		}

		responses = append(
			responses,
			productToResponse(&product, bundleItems),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"products": responses,
	}); err != nil {
		log.Printf("Error encoding products response: %v", err)
	}
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(r)["id"]

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	var localProduct models.Product

	err := services.GetDB().
		WithContext(r.Context()).
		Preload("BundleItems").
		Where("id = ?", productID).
		First(&localProduct).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf(
			"Error loading product %s from database: %v",
			productID,
			err,
		)
		http.Error(
			w,
			"Failed to load product",
			http.StatusInternalServerError,
		)
		return
	}

	if !localProduct.Active {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if err := loadProductPrices(r.Context(), &localProduct); err != nil {
		log.Printf(
			"Error loading prices for product %s: %v",
			productID,
			err,
		)
		http.Error(
			w,
			"Failed to load product prices",
			http.StatusInternalServerError,
		)
		return
	}

	bundleItems, err := loadBundleItems(r.Context(), &localProduct)
	if err != nil {
		log.Printf(
			"Error loading bundle items for product %s: %v",
			productID,
			err,
		)
		http.Error(
			w,
			"Failed to load bundle items",
			http.StatusInternalServerError,
		)
		return
	}

	response := productToResponse(&localProduct, bundleItems)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding product response: %v", err)
	}
}

func GetBundleProducts(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	var products []models.Product

	query := services.GetDB().
		WithContext(r.Context()).
		Preload("BundleItems").
		Where("type = ?", "bundle")

	if !includeInactive {
		query = query.Where("active = ?", true)
	}

	if err := query.Order("created_at DESC").Find(&products).Error; err != nil {
		http.Error(
			w,
			"Failed to load bundled products",
			http.StatusInternalServerError,
		)
		return
	}

	response := make([]ProductResponse, 0, len(products))

	for index := range products {
		product := &products[index]

		if err := loadProductPrices(r.Context(), product); err != nil {
			http.Error(
				w,
				"Failed to load bundled product prices",
				http.StatusInternalServerError,
			)
			return
		}

		bundleItems, err := loadBundleItems(r.Context(), product)
		if err != nil {
			http.Error(
				w,
				"Failed to load bundled product items",
				http.StatusInternalServerError,
			)
			return
		}

		response = append(response, productToResponse(
			product,
			bundleItems,
		))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"products": response,
	})
}

type UpdateVariantStockRequest struct {
	Variants []VariantStockUpdate `json:"variants"`
}

type VariantStockUpdate struct {
	ID            string `json:"id"`
	StockQuantity *int   `json:"stock_quantity"`
}

func UpdateVariantStock(w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(r)["productId"]

	if productID == "" {
		http.Error(
			w,
			"Product ID is required",
			http.StatusBadRequest,
		)
		return
	}

	var req UpdateVariantStockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"Invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	db := services.GetDB().WithContext(r.Context())

	for _, update := range req.Variants {
		if update.ID == "" {
			http.Error(
				w,
				"Variant price ID is required",
				http.StatusBadRequest,
			)
			return
		}

		var priceInfo models.PriceInfo

		err := db.
			Where(
				"id = ? AND stripe_product_id = ?",
				update.ID,
				productID,
			).
			First(&priceInfo).
			Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(
					w,
					"Variant does not belong to this product",
					http.StatusBadRequest,
				)
				return
			}

			http.Error(
				w,
				"Failed to load variant",
				http.StatusInternalServerError,
			)
			return
		}

		if update.StockQuantity != nil &&
			*update.StockQuantity < 0 {
			http.Error(
				w,
				"Stock quantity cannot be negative",
				http.StatusBadRequest,
			)
			return
		}

		if err := db.
			Model(&models.PriceInfo{}).
			Where("id = ?", update.ID).
			Updates(map[string]interface{}{
				"stock_quantity": update.StockQuantity,
				"active": update.StockQuantity == nil ||
					*update.StockQuantity > 0,
			}).
			Error; err != nil {
			http.Error(
				w,
				"Failed to update stock",
				http.StatusInternalServerError,
			)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

func GetProductVariants(w http.ResponseWriter, r *http.Request) {
	productID := mux.Vars(r)["productId"]

	var variants []models.PriceInfo

	err := services.GetDB().
		WithContext(r.Context()).
		Where(
			"stripe_product_id = ? AND active = TRUE",
			productID,
		).
		Order("unit_amount ASC, id ASC").
		Find(&variants).
		Error

	if err != nil {
		http.Error(
			w,
			"Failed to load product variants",
			http.StatusInternalServerError,
		)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"variants": variants,
	})
}

func productToResponse(
	product *models.Product,
	bundleItems []BundleItemResponse,
) ProductResponse {
	prices := product.Prices

	if prices == nil {
		prices = []models.PriceInfo{}
	}

	created := product.CreatedAt.Unix()
	updated := product.UpdatedAt.Unix()

	response := ProductResponse{
		ID:            product.ID,
		Name:          product.Title,
		Description:   product.Description,
		Images:        []string(product.Images),
		Type:          product.Type,
		Price:         product.Price,
		Currency:      product.Currency,
		CompareAtPrice: product.CompareAtPrice,
		IsNew:         product.IsNew,
		InStock:       product.InStock,
		Featured:      product.Featured,
		Category:      product.Category,
		Subcategory:   product.Subcategory,
		Tags:          []string(product.Tags),
		MaxQuantity:   product.MaxQuantity,
		Active:        product.Active,
		DefaultPrice:  defaultProductPrice(prices),
		Prices:        prices,
		BundleItems:   bundleItems,
		Created:       created,
		Updated:       updated,
	}

	return response
}

func defaultProductPrice(prices []models.PriceInfo) *models.PriceInfo {
	for index := range prices {
		if prices[index].IsDefault && prices[index].Active {
			return &prices[index]
		}
	}

	for index := range prices {
		if prices[index].Active {
			return &prices[index]
		}
	}

	return nil
}

func loadProductPrices(
	ctx context.Context,
	product *models.Product,
) error {
	var prices []models.PriceInfo

	err := services.GetDB().
		WithContext(ctx).
		Preload("IncludedProductLinks").
		Where(
			"stripe_product_id = ?",
			product.ID,
		).
		Order("active DESC, unit_amount ASC, id ASC").
		Find(&prices).
		Error

	if err != nil {
		return err
	}

	if prices == nil {
		prices = []models.PriceInfo{}
	}

	product.Prices = prices

	return nil
}

func loadBundleItems(
	ctx context.Context,
	product *models.Product,
) ([]BundleItemResponse, error) {
	if product.Type != "bundle" {
		return []BundleItemResponse{}, nil
	}

	result := make([]BundleItemResponse, 0, len(product.BundleItems))

	for _, item := range product.BundleItems {
		var bundledProduct models.Product

		if err := services.GetDB().
			WithContext(ctx).
			Where("id = ?", item.ProductID).
			First(&bundledProduct).
			Error; err != nil {
			return nil, fmt.Errorf(
				"load bundled product %s: %w",
				item.ProductID,
				err,
			)
		}

		result = append(result, BundleItemResponse{
			ProductID:   bundledProduct.ID,
			ProductName: bundledProduct.Title,
			Quantity:    item.Quantity,
			Price:       bundledProduct.Price,
		})
	}

	return result, nil
}
