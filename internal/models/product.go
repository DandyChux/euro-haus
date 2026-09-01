package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID string `gorm:"type:varchar(255);primaryKey" json:"id"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`
	Type        string `gorm:"not null;default:product;index" json:"type"`

	Images ProductStringList `gorm:"type:jsonb;not null;default:'[]'::jsonb" json:"images"`

	Price          int64  `gorm:"not null;default:0" json:"price"`
	Currency       string `gorm:"not null;default:usd" json:"currency"`
	CompareAtPrice *int64 `gorm:"column:compare_at_price" json:"compare_at_price,omitempty"`

	IsNew    bool `gorm:"column:is_new;not null;default:false" json:"is_new"`
	InStock bool `gorm:"column:in_stock;not null;default:true" json:"in_stock"`
	Active bool `gorm:"not null;default:true" json:"active"`
	Featured bool `gorm:"not null;default:false" json:"featured"`

	Category    string `json:"category,omitempty"`
	Subcategory string `json:"subcategory,omitempty"`

	Tags        ProductStringList `gorm:"type:jsonb;not null;default:'[]'::jsonb" json:"tags"`
	MaxQuantity *int              `gorm:"column:max_quantity" json:"max_quantity,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Prices []PriceInfo `gorm:"-" json:"prices,omitempty"`
	BundleItems []BundleItem `gorm:"foreignKey:BundleProductID;references:ID" json:"bundle_items,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

type BundleItem struct {
	BundleProductID string `gorm:"type:varchar(255);primaryKey"`
	ProductID       string `gorm:"type:varchar(255);primaryKey"`
	Quantity        int    `gorm:"not null;default:1"`
	SortOrder       int    `gorm:"not null;default:0"`
}

func (BundleItem) TableName() string {
	return "bundle_items"
}

type ProductStringList []string

func (ProductStringList) GormDataType() string {
	return "json"
}

func (ProductStringList) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value ProductStringList) Value() (driver.Value, error) {
	if value == nil {
		value = ProductStringList{}
	}

	return json.Marshal(value)
}

func (value *ProductStringList) Scan(input interface{}) error {
	if input == nil {
		*value = ProductStringList{}
		return nil
	}

	var raw []byte

	switch value := input.(type) {
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return fmt.Errorf(
			"cannot scan product string list from %T",
			input,
		)
	}

	if len(raw) == 0 {
		*value = ProductStringList{}
		return nil
	}

	return json.Unmarshal(raw, value)
}
