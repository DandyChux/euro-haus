package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID string `gorm:"type:uuid;primaryKey"`

	StripeProductID string `gorm:"uniqueIndex;not null"`

	Slug            string `gorm:"uniqueIndex;not null"`
	Name            string `gorm:"not null"`
	Description     string
	LongDescription string

	Images EventImages `gorm:"type:jsonb;not null;default:'[]'::jsonb"`

	EventDate string
	Location  string
	Venue     string
	Organizer string

	Capacity       int `gorm:"not null;default:0"`
	AvailableSpots int `gorm:"not null;default:0"`

	Status   string `gorm:"not null;default:upcoming;index"`
	Active   bool   `gorm:"not null;default:true;index"`
	Featured bool   `gorm:"not null;default:false"`

	Tags     EventStringList `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	Agenda   EventAgenda     `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	Includes EventStringList `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	Sponsors EventSponsors   `gorm:"type:jsonb;not null;default:'[]'::jsonb"`

	Prices []PriceInfo `gorm:"-" json:"prices"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Event) TableName() string {
	return "events"
}

func (event *Event) BeforeCreate(tx *gorm.DB) error {
	if event.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		event.ID = id.String()
	}

	return nil
}

type EventImages []string
type EventStringList []string

type EventAgenda []AgendaItem

type AgendaItem struct {
	Time     string `json:"time"`
	Activity string `json:"activity"`
}

type Sponsor struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Logo        string `json:"logo,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

type EventSponsors []Sponsor

type EventProductLink struct {
	EventID   string `gorm:"type:uuid;primaryKey"`
	ProductID string `gorm:"primaryKey"`
	SortOrder int    `gorm:"not null;default:0"`
}

func (EventProductLink) TableName() string {
	return "event_product_links"
}

func (EventImages) GormDataType() string {
	return "json"
}

func (EventImages) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value EventImages) Value() (driver.Value, error) {
	return json.Marshal(value)
}

func (value *EventImages) Scan(input interface{}) error {
	return scanEventJSON(input, value)
}

func (EventStringList) GormDataType() string {
	return "json"
}

func (EventStringList) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value EventStringList) Value() (driver.Value, error) {
	return json.Marshal(value)
}

func (value *EventStringList) Scan(input interface{}) error {
	return scanEventJSON(input, value)
}

func (EventAgenda) GormDataType() string {
	return "json"
}

func (EventAgenda) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value EventAgenda) Value() (driver.Value, error) {
	return json.Marshal(value)
}

func (value *EventAgenda) Scan(input interface{}) error {
	return scanEventJSON(input, value)
}

func (EventSponsors) GormDataType() string {
	return "json"
}

func (EventSponsors) GormDBDataType(db *gorm.DB, fieldName string) string {
	return "JSONB"
}

func (value EventSponsors) Value() (driver.Value, error) {
	return json.Marshal(value)
}

func (value *EventSponsors) Scan(input interface{}) error {
	return scanEventJSON(input, value)
}

func scanEventJSON(input interface{}, destination interface{}) error {
	if input == nil {
		return nil
	}

	var raw []byte

	switch value := input.(type) {
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return fmt.Errorf("unsupported JSON value type %T", input)
	}

	return json.Unmarshal(raw, destination)
}
