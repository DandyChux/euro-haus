package models

import "time"

// ------------------------------------------------------------
// ContentPlacement – CMS-lite content blocks
// ------------------------------------------------------------
type ContentPlacement struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Page        string `gorm:"not null;index:idx_placements_page"`
	Type        string `gorm:"not null"`
	MediaURL    string
	MediaKey    string `gorm:"index:idx_placements_media_key"`
	TextContent string
	HTML        bool `gorm:"not null;default:false"`
	UpdatedBy   string
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ContentPlacement) TableName() string { return "content_placements" }
