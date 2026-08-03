package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/dandychux/euro-haus/internal/services"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

func main() {
	_ = godotenv.Load(".env")

	email := flag.String("email", os.Getenv("INITIAL_ADMIN_EMAIL"), "admin email")
	name := flag.String("name", os.Getenv("INITIAL_ADMIN_NAME"), "admin name")
	passwordStdin := flag.Bool(
		"password-stdin",
		false,
		"read the password from stdin instead of prompting",
	)

	flag.Parse()

	password := os.Getenv("INITIAL_ADMIN_PASSWORD")

	if password == "" {
		if *passwordStdin {
			password = readPasswordFromStdin()
		} else {
			password = readPasswordInteractively()
		}
	}

	*email = strings.TrimSpace(*email)
	*name = strings.TrimSpace(*name)

	if *email == "" || password == "" || *name == "" {
		log.Fatal("email, password, and name are required")
	}

	services.InitDB()

	authService := services.NewAuthService()
	user, err := authService.CreateAdminUser(
		context.Background(),
		&services.RegisterRequest{
			Email:    *email,
			Password: password,
			Name:     *name,
			Country:  "US",
		},
	)
	if err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	fmt.Printf("created admin account %s (%s)\n", user.Email, user.ID)
}

func readPasswordInteractively() string {
	if !term.IsTerminal(int(syscall.Stdin)) {
		log.Fatal("no interactive terminal; use -password-stdin")
	}

	fmt.Fprint(os.Stderr, "Password: ")

	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)

	if err != nil {
		log.Fatalf("failed to read password: %v", err)
	}

	return strings.TrimSpace(string(password))
}

func readPasswordFromStdin() string {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		log.Fatal("failed to read password from stdin")
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read password from stdin: %v", err)
	}

	return strings.TrimSpace(scanner.Text())
}
