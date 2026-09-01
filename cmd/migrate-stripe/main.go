package main

import (
	"context"
	"encoding/json"
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

	productID := strings.TrimSpace(*stripeProductID)
	if productID == "" {
		log.Fatal("the -product-id argument is required")
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		log.Fatal("STRIPE_SECRET_KEY is required")
	}

	services.InitDB()

	ctx := context.Background()

	stripeProduct, err := getProduct(productID)
	if err != nil {
		log.Fatalf("unable to retrieve Stripe product: %v", err)
	}

	productType := defaultString(
		stripeProduct.Metadata["type"],
		"product",
	)

	switch productType {
	case "event":
		if err := migrateEvent(ctx, stripeProduct); err != nil {
			log.Fatalf("event migration failed: %v", err)
		}

	case "product", "bundle":
		if err := migrateProduct(ctx, stripeProduct, productType); err != nil {
			log.Fatalf("product migration failed: %v", err)
		}

	default:
		log.Fatalf(
			"unsupported Stripe product type %q; expected event, product, or bundle",
			productType,
		)
	}
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

func getProduct(productID string) (*stripe.Product, error) {
	params := &stripe.ProductParams{}
	params.AddExpand("default_price")

	stripeProduct, err := product.Get(productID, params)
	if err != nil {
		return nil, err
	}

	return stripeProduct, nil
}

func migrateEvent(
	ctx context.Context,
	stripeProduct *stripe.Product,
) error {
	event, err := buildEvent(stripeProduct)
	if err != nil {
		return err
	}

	db := services.GetDB()

	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}

		if err := migrateSubmissions(
			tx,
			event,
			stripeProduct.ID,
		); err != nil {
			return err
		}

		return services.SyncStripeProductPrices(
			ctx,
			stripeProduct.ID,
		)
	}); err != nil {
		return err
	}

	log.Printf(
		"created event %s (%s)",
		event.ID,
		event.Slug,
	)

	log.Println("Existing tickets were intentionally not migrated.")

	return nil
}

func migrateProduct(
	ctx context.Context,
	stripeProduct *stripe.Product,
	productType string,
) error {
	localProduct := buildProduct(stripeProduct, productType)
	db := services.GetDB()

	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&localProduct).Error; err != nil {
			return err
		}

		return services.SyncStripeProductPrices(
			ctx,
			stripeProduct.ID,
		)
	}); err != nil {
		return err
	}

	log.Printf(
		"created product %s (%s)",
		localProduct.ID,
		localProduct.Title,
	)

	return nil
}

func buildProduct(
	stripeProduct *stripe.Product,
	productType string,
) models.Product {
	metadata := stripeProduct.Metadata

	price := int64(0)
	currency := "usd"

	if stripeProduct.DefaultPrice != nil {
		price = stripeProduct.DefaultPrice.UnitAmount
		currency = string(stripeProduct.DefaultPrice.Currency)
	}

	return models.Product{
		ID:          stripeProduct.ID,
		Title:       stripeProduct.Name,
		Description: stripeProduct.Description,
		Type:        productType,
		Images:      models.ProductStringList(stripeProduct.Images),

		Price:          price,
		Currency:       currency,
		CompareAtPrice: optionalInt64(metadata["compare_at_price"]),

		IsNew:    metadata["is_new"] == "true",
		InStock:  stripeProduct.Active && metadata["in_stock"] != "false",
		Featured: metadata["featured"] == "true",

		Category:    defaultString(metadata["category"], "merchandise"),
		Subcategory: defaultString(metadata["subcategory"], "general"),

		Tags:        metadataProductStringList(metadata["tags"]),
		MaxQuantity: optionalInt(metadata["max_quantity"]),
	}
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

func optionalInt64(value string) *int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil || result < 0 {
		return nil
	}

	return &result
}

func optionalInt(value string) *int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	result, err := strconv.Atoi(value)
	if err != nil || result < 0 {
		return nil
	}

	return &result
}

func metadataProductStringList(value string) models.ProductStringList {
	if strings.TrimSpace(value) == "" {
		return models.ProductStringList{}
	}

	var result models.ProductStringList
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return models.ProductStringList{}
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
