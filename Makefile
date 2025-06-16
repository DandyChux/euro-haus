# The .PHONY directive tells Make that these targets are not files
# This prevents conflicts if files with these names exist in the directory
# and ensures the targets always run when called
.PHONY: start-server start-db build clean test run migrate dev docker deps docs fmt lint

start-server:
	@echo "Starting server..."
	@cd api && air -c air.toml

start-db:
	@echo "Starting database..."
	@cd api && docker compose -f docker-compose.dev.yml up -d postgres

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/server ./main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf web/

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download
