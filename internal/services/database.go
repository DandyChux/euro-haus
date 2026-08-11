package services

import (
	"fmt"
	"log"
	"os"

	"github.com/dandychux/euro-haus/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var DatabaseDSN string

// InitDB initializes the shared PostgreSQL connection pool.
func InitDB() {
	DatabaseDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("DB_PORT"),
	)

	config := &gorm.Config{}
	if os.Getenv("ENV") == "development" {
		config.Logger = logger.Default.LogMode(logger.Info)
	}

	var err error
	DB, err = gorm.Open(postgres.Open(DatabaseDSN), config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = DB.AutoMigrate(
		&models.ContentPlacement{},
		&models.Event{},
		&models.Product{},
		&models.EventProductLink{},
		&models.Fulfillment{},
		&models.BundleItem{},
		&models.PriceInfo{},
		&models.PriceIncludedProduct{},
		&models.Ticket{},
		&models.TokenData{},
		&models.User{},
		&models.PriceRequirement{},
		&models.SubmissionRequirementAnswer{},
		&models.VehicleSubmission{},
		&models.EmailJob{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database connected and migrated successfully")
}

func GetDB() *gorm.DB {
	return DB
}

func GetDatabaseDSN() string {
	return DatabaseDSN
}
