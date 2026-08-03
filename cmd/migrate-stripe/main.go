package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/product"
	"gorm.io/gorm"
)

type legacySponsor struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Logo        string `json:"logo,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

type legacySponsorTier struct {
	TierName     string                 `json:"tierName"`
	DisplayOrder int                    `json:"displayOrder"`
	Sponsors     []legacySponsor       `json:"sponsors"`
}

type legacyAgendaItem struct {
	Time     string `json:"time"`
	Activity string `json:"activity"`
}

func main() {
	loadEnvironment()

	stripeProductID := flag.String(
		"product-id",
		"",
		"Stripe product ID to migrate, for example prod_123",
	)

	flag.Parse()

	if strings.TrimSpace(*stripeProductID) == "" {
		log.Fatal("the -product-id argument is required")
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		log.Fatal("STRIPE_SECRET_KEY is required")
	}

	services.InitDB()

	ctx := context.Background()

	stripeEvent, err := getStripeEvent(strings.TrimSpace(*stripeProductID))
	if err != nil {
		log.Fatalf("unable to retrieve Stripe product: %v", err)
	}

	event, err := buildEvent(stripeEvent)
	if err != nil {
		log.Fatalf("unable to build event: %v", err)
	}

	db := services.GetDB()

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}

		if err := migrateSubmissions(
			tx,
			event,
			stripeEvent.ID,
		); err != nil {
			return err
		}

		if err := services.SyncStripeProductPrices(
			ctx,
			stripeEvent.ID,
		); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatalf("event migration failed: %v", err)
	}

	log.Printf(
		"created event %s (%s)",
		event.ID,
		event.Slug,
	)

	log.Println("Existing tickets were intentionally not migrated.")
}

func loadEnvironment() {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env file found")
		}
	}
}

func getStripeEvent(productID string) (*stripe.Product, error) {
	params := &stripe.ProductParams{}
	params.AddExpand("default_price")

	event, err := product.Get(productID, params)
	if err != nil {
		return nil, err
	}

	if event.Metadata["type"] != "event" {
		return nil, errors.New("Stripe product is not marked as an event")
	}

	return event, nil
}

func buildEvent(stripeEvent *stripe.Product) (models.Event, error) {
	metadata := stripeEvent.Metadata

	capacity := parseInt(metadata["capacity"])
	availableSpots := parseInt(metadata["available_spots"])

	if availableSpots == 0 && capacity > 0 {
		availableSpots = capacity
	}

	images := models.EventImages{}
	if len(stripeEvent.Images) > 0 {
		images = append(images, stripeEvent.Images...)
	}

	event := models.Event{
		StripeProductID: stripeEvent.ID,
		Slug:            metadata["slug"],
		Name:            stripeEvent.Name,
		Description:     stripeEvent.Description,
		LongDescription: metadata["long_description"],
		Images:          images,

		EventDate: metadata["event_date"],
		Location:  metadata["location"],
		Venue:     metadata["venue"],
		Organizer: metadata["organizer"],

		Capacity:       capacity,
		AvailableSpots: availableSpots,

		Status:   defaultString(metadata["status"], "completed"),
		Active:   false,
		Featured: metadata["featured"] == "true",

		Tags:     metadataStringList(metadata["tags"]),
		Agenda:   metadataAgenda(metadata["agenda"]),
		Includes: metadataStringList(metadata["includes"]),
		Sponsors: migrateSponsors(
			metadata["sponsors"],
			metadata["sponsor_tiers"],
		),
	}

	return event, nil
}

func migrateSubmissions(
	tx *gorm.DB,
	event models.Event,
	legacyStripeProductID string,
) error {
	result := tx.Exec(`
		UPDATE vehicle_submissions
		SET event_id = ?,
		    event_slug = ?
		WHERE event_id = ?
	`, event.ID, event.Slug, legacyStripeProductID)

	if result.Error != nil {
		return result.Error
	}

	log.Printf(
		"updated %d submissions for event %s",
		result.RowsAffected,
		event.ID,
	)

	return nil
}

func migrateLinkedProducts(
	tx *gorm.DB,
	event models.Event,
	raw string,
) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	ids := uniqueCSVValues(raw)

	links := make([]models.EventProductLink, 0, len(ids))

	for index, productID := range ids {
		linkedProduct, err := product.Get(productID, nil)
		if err != nil {
			log.Printf(
				"skipping linked product %s: %v",
				productID,
				err,
			)
			continue
		}

		if !linkedProduct.Active {
			continue
		}

		links = append(links, models.EventProductLink{
			EventID:   event.ID,
			ProductID: productID,
			SortOrder: index,
		})
	}

	if len(links) == 0 {
		return nil
	}

	return tx.Create(&links).Error
}

func migrateSponsors(
	rawSponsors string,
	rawSponsorTiers string,
) models.EventSponsors {
	var sponsors models.EventSponsors

	if rawSponsors != "" {
		if err := json.Unmarshal(
			[]byte(rawSponsors),
			&sponsors,
		); err == nil {
			return sponsors
		}
	}

	var legacyTiers []struct {
		TierName string `json:"tierName"`
		Sponsors []struct {
			Name        string `json:"name"`
			Logo        string `json:"logo,omitempty"`
			URL         string `json:"url,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"sponsors"`
	}

	if rawSponsorTiers == "" {
		return sponsors
	}

	if err := json.Unmarshal(
		[]byte(rawSponsorTiers),
		&legacyTiers,
	); err != nil {
		log.Printf("unable to parse sponsor tiers: %v", err)
		return sponsors
	}

	for _, tier := range legacyTiers {
		for _, sponsor := range tier.Sponsors {
			sponsors = append(sponsors, models.Sponsor{
				Name:        sponsor.Name,
				Tier:        tier.TierName,
				Logo:        sponsor.Logo,
				URL:         sponsor.URL,
				Description: sponsor.Description,
			})
		}
	}

	return sponsors
}

func metadataStringList(value string) models.EventStringList {
	if value == "" {
		return models.EventStringList{}
	}

	var result models.EventStringList

	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return models.EventStringList{}
	}

	return result
}

func metadataAgenda(value string) models.EventAgenda {
	if value == "" {
		return models.EventAgenda{}
	}

	var result models.EventAgenda

	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return models.EventAgenda{}
	}

	return result
}

func parseInt(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil || result < 0 {
		return 0
	}

	return result
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func uniqueCSVValues(value string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, raw := range strings.Split(value, ",") {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
