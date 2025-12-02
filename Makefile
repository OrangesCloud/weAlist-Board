# Variables
APP_NAME=project-board-api
BINARY_NAME=main
DOCKER_IMAGE=$(APP_NAME):latest
DOCKER_IMAGE_DEV=$(APP_NAME):dev
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
COVERAGE_FILE=coverage.out

# Database variables (can be overridden by environment)
DB_HOST?=localhost
DB_PORT?=5432
DB_USER?=postgres
DB_PASSWORD?=password
DB_NAME?=project_board
DATABASE_URL?=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Build variables
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: help build run test lint clean migrate-up migrate-down docker-build docker-run

## help: Display this help screen
help:
	@echo "Available targets:"
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | sed 's/^## //' | awk 'BEGIN {FS = ":"}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## build: Build the application binary
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/api/main.go
	@echo "✓ Build complete: bin/$(BINARY_NAME)"

## build-linux: Build for Linux (useful for Docker)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux cmd/api/main.go
	@echo "✓ Linux build complete: bin/$(BINARY_NAME)-linux"

## run: Run the application
run:
	@echo "Running $(APP_NAME)..."
	@go run cmd/api/main.go

## dev: Run in development mode with hot reload (requires air)
dev:
	@if ! command -v air > /dev/null; then \
		echo "air is not installed. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@air

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	@echo "✓ Tests complete"

## test-short: Run tests without race detector (faster)
test-short:
	@echo "Running tests (short mode)..."
	@go test -v -short ./...

## test-coverage: Run tests and display coverage report
test-coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=$(COVERAGE_FILE)

## test-coverage-text: Display coverage in terminal
test-coverage-text: test
	@echo "Coverage summary:"
	@go tool cover -func=$(COVERAGE_FILE)

## lint: Run linter (golangci-lint)
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "golangci-lint is not installed. Please install it: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	@golangci-lint run ./...
	@echo "✓ Lint complete"

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

## check: Run fmt, vet, and lint
check: fmt vet lint
	@echo "✓ All checks passed"

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

## deps-upgrade: Upgrade all dependencies
deps-upgrade:
	@echo "Upgrading dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies upgraded"

## clean: Clean build artifacts and cache
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f $(COVERAGE_FILE)
	@go clean -cache -testcache -modcache
	@echo "✓ Clean complete"

## migrate-up: Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	@if [ ! -f migrations/001_init_schema.sql ]; then \
		echo "Error: Migration file not found"; \
		exit 1; \
	fi
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/001_init_schema.sql
	@if [ -f migrations/002_add_project_members_and_board_fields.sql ]; then \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/002_add_project_members_and_board_fields.sql; \
	fi
	@if [ -f migrations/003_migrate_existing_project_owners.sql ]; then \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/003_migrate_existing_project_owners.sql; \
	fi
	@echo "✓ Migrations applied"

## migrate-down: Run database migrations down
migrate-down:
	@echo "Running migrations down..."
	@if [ -f migrations/003_migrate_existing_project_owners_down.sql ]; then \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/003_migrate_existing_project_owners_down.sql; \
	fi
	@if [ -f migrations/002_add_project_members_and_board_fields_down.sql ]; then \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/002_add_project_members_and_board_fields_down.sql; \
	fi
	@if [ ! -f migrations/001_init_schema_down.sql ]; then \
		echo "Error: Migration down file not found"; \
		exit 1; \
	fi
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/001_init_schema_down.sql
	@echo "✓ Migrations rolled back"

## migrate-status: Check migration status
migrate-status:
	@echo "Checking database connection..."
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "\dt" || echo "Cannot connect to database"

## db-create: Create database
db-create:
	@echo "Creating database $(DB_NAME)..."
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "CREATE DATABASE $(DB_NAME);" || echo "Database may already exist"
	@echo "✓ Database created"

## db-drop: Drop database (WARNING: destructive)
db-drop:
	@echo "WARNING: This will drop the database $(DB_NAME)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);"; \
		echo "✓ Database dropped"; \
	else \
		echo "Cancelled"; \
	fi

## db-reset: Drop, create, and migrate database
db-reset: db-drop db-create migrate-up
	@echo "✓ Database reset complete"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .
	@echo "✓ Docker image built: $(DOCKER_IMAGE)"

## docker-build-dev: Build Docker image for development
docker-build-dev:
	@echo "Building development Docker image..."
	@docker build -t $(DOCKER_IMAGE_DEV) --target builder .
	@echo "✓ Development Docker image built: $(DOCKER_IMAGE_DEV)"

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		--env-file .env \
		$(DOCKER_IMAGE)
	@echo "✓ Container started: $(APP_NAME)"

## docker-run-interactive: Run Docker container interactively
docker-run-interactive:
	@echo "Running Docker container (interactive)..."
	@docker run -it --rm \
		-p 8080:8080 \
		--env-file .env \
		$(DOCKER_IMAGE)

## docker-stop: Stop Docker container
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true
	@echo "✓ Container stopped"

## docker-logs: View Docker container logs
docker-logs:
	@docker logs -f $(APP_NAME)

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d
	@echo "✓ Services started"

## docker-compose-down: Stop services with docker-compose
docker-compose-down:
	@echo "Stopping services with docker-compose..."
	@docker-compose down
	@echo "✓ Services stopped"

## docker-compose-logs: View docker-compose logs
docker-compose-logs:
	@docker-compose logs -f

## docker-clean: Remove Docker images and containers
docker-clean:
	@echo "Cleaning Docker resources..."
	@docker stop $(APP_NAME) 2>/dev/null || true
	@docker rm $(APP_NAME) 2>/dev/null || true
	@docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "✓ Docker resources cleaned"

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✓ Tools installed"

## swagger: Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@if ! command -v swag > /dev/null; then \
		echo "swag is not installed. Installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g cmd/api/main.go -o docs
	@echo "✓ Swagger documentation generated"

## all: Run fmt, vet, lint, test, and build
all: fmt vet lint test build
	@echo "✓ All tasks complete"

.DEFAULT_GOAL := help
