.PHONY: help db-up db-down db-reset db-status db-psql db-migrate-up db-migrate-down docker-up docker-down docker-logs clean dev-server test-coverage docker-build goimports

# Load environment variables
include .env
export

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Docker commands
docker-up: ## Start all services (PostgreSQL)
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-logs: ## Show logs from all services
	docker compose logs -f

docker-clean: ## Remove all containers and volumes
	docker compose down -v

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t rekko-api:latest .
	@echo "Image built successfully!"
	@docker images rekko-api:latest --format "Image size: {{.Size}}"

# Database commands
db-status: ## Show database status
	@./scripts/db.sh status

db-psql: ## Connect to database with psql
	@./scripts/db.sh psql

db-seed: ## Seed database with test data
	@./scripts/db.sh seed

db-dump: ## Create database backup
	@./scripts/db.sh dump

# Migration commands
db-migrate-up: ## Run all pending migrations
	@./scripts/db.sh up

db-migrate-down: ## Rollback last migration
	@./scripts/db.sh down

db-migrate-reset: ## Reset database (drop + recreate)
	@./scripts/db.sh reset

db-migrate-version: ## Show current migration version
	@./scripts/db.sh version

# Development workflow
dev: docker-up db-migrate-up ## Start development environment
	@echo "Development environment ready!"
	@echo "Database: postgres://rekko:rekko@localhost:5433/rekko_dev"

dev-server: ## Start development server with hot reload
	@echo "Starting development server with hot reload..."
	@air

dev-clean: docker-down ## Clean development environment
	@echo "Development environment cleaned!"

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ tmp/ coverage.out coverage.html build-errors.log
	@echo "Clean complete!"

# Testing
test: ## Run all tests
	go test -v -race ./...

test-unit: ## Run unit tests only
	go test -v -short ./...

test-integration: ## Run integration tests
	go test -v -tags=integration ./...

test-short: ## Run short tests only
	go test -v -short ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Code quality
lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

goimports: ## Run goimports
	@goimports -w .

# Build
build: ## Build application
	go build -o bin/rekko ./cmd/api

run: ## Run application
	go run ./cmd/api

# Dependencies
deps: ## Download dependencies
	go mod download

deps-tidy: ## Tidy dependencies
	go mod tidy

deps-verify: ## Verify dependencies
	go mod verify

# Install tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Tools installed successfully!"

setup: docker-up db-migrate-up ## Full setup (docker + migrations)
	@echo "Setup complete!"
	@echo "Run 'make dev-server' to start development with hot reload"
