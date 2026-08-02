package main

import (
	"context"
	"log"
	"os"

	"github.com/dandychux/euro-haus/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	email := os.Getenv("INITIAL_ADMIN_EMAIL")
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	name := os.Getenv("INITIAL_ADMIN_NAME")

	if email == "" || password == "" || name == "" {
		log.Fatal(
			"INITIAL_ADMIN_EMAIL, INITIAL_ADMIN_PASSWORD, and INITIAL_ADMIN_NAME are required",
		)
	}

	services.InitDB()

	authService := services.NewAuthService()

	user, err := authService.CreateAdminUser(
		context.Background(),
		&services.RegisterRequest{
			Email:    email,
			Password: password,
			Name:     name,
			Country:  "US",
		},
	)
	if err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	log.Printf("created admin account %s (%s)", user.Email, user.ID)
}
