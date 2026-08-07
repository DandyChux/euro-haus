package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ------------------------------------------------------------
// Fulfillment – replaces fulfillment:<id> and fulfillments:* sets
// ------------------------------------------------------------
type Fulfillment struct {
	ID              string `gorm:"primaryKey"`
	SessionID       string `gorm:"not null;index:idx_fulfillments_session"`
	ProductID       string
	ProductName     string
	CustomerEmail   string `gorm:"not null;index:idx_fulfillments_email"`
	CustomerName    string
	ShippingAddress string
	Quantity        int    `gorm:"not null;default:1"`
	Status          string `gorm:"not null;default:pending;index:idx_fulfillments_status"`
	Type            string `gorm:"not null;default:purchased"`
	TrackingNumber  string
	TrackingCarrier string
	Notes           string
	ShippedAt       *time.Time
	DeliveredAt     *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (Fulfillment) TableName() string { return "fulfillments" }

// ------------------------------------------------------------
// PriceInfo
// ------------------------------------------------------------
type PriceInfo struct {
	// Stripe Price ID.
	ID string `gorm:"primaryKey;column:id" json:"id"`
	StripeProductID string `gorm:"not null;index" json:"stripe_product_id"`

	UnitAmount int64  `gorm:"not null" json:"unit_amount"`
	Currency   string `gorm:"not null" json:"currency"`
	Nickname   string `gorm:"column:nickname" json:"nickname,omitempty"`
	Description string `gorm:"column:description" json:"description,omitempty"`

	Active bool `gorm:"not null;default:true;index" json:"active"`

	Features datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb" json:"features"`

	IsDefault bool `gorm:"column:is_default;not null;default:false" json:"default"`
	IsMostPopular bool `gorm:"column:is_most_popular;not null;default:false" json:"most_popular"`
	RequiresApproval bool `gorm:"column:requires_approval;not null;default:false" json:"requires_approval"`
	RequiresSubmission bool `gorm:"column:requires_submission;not null;default:false" json:"requires_submission"`

	Quantity int `gorm:"column:quantity" json:"quantity"`
	StockQuantity *int `gorm:"column:stock_quantity" json:"stock_quantity"`
	Size string `gorm:"column:size" json:"size,omitempty"`
	Color string `gorm:"column:color" json:"color,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	IncludedProductLinks []PriceIncludedProduct `gorm:"foreignKey:PriceID;references:ID" json:"-"`
}

func (PriceInfo) TableName() string {
	return "prices"
}

type PriceIncludedProduct struct {
	PriceID string `gorm:"type:varchar(255);primaryKey;index"`
	ProductID string `gorm:"type:varchar(255);primaryKey;index"`

	Quantity  int `gorm:"not null;default:1"`
	SortOrder int `gorm:"not null;default:0"`
}

func (PriceIncludedProduct) TableName() string {
	return "price_included_products"
}

type PriceMetadata map[string]string

func (PriceMetadata) GormDataType() string {
	return "json"
}

func (PriceMetadata) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value PriceMetadata) Value() (driver.Value, error) {
	if value == nil {
		value = PriceMetadata{}
	}

	return json.Marshal(value)
}

func (value *PriceMetadata) Scan(input interface{}) error {
	if input == nil {
		*value = PriceMetadata{}
		return nil
	}

	return scanEventJSON(input, value)
}
