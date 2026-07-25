# LANdapter Makefile
# Targets:
#   build          - build master and agent binaries
#   run-master     - run master server
#   run-agent      - run agent client
#   run-frontend   - run frontend development server
#   build-frontend - build frontend for production
#   migrate-up     - apply all database migrations
#   migrate-down   - rollback all database migrations
#   migrate-apply  - apply a specific migration (usage: make migrate-apply NAME=004_add_snapshots)
#   test           - run all unit tests
#   test-unit      - run unit tests only
#   test-integration - run integration tests (requires database)
#   test-cover     - run tests with coverage report
#   clean          - remove build artifacts
#   help           - show this help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_DIR=bin
MASTER_BINARY=$(BINARY_DIR)/master
AGENT_BINARY=$(BINARY_DIR)/agent

# PostgreSQL connection parameters (adjust as needed)
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=postgres
PG_DB=landapter
PG_SSL=disable
PG_DSN=postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_DB)?sslmode=$(PG_SSL)

# Migration directory
MIGRATIONS_DIR=migrations

# Frontend directory
WEB_DIR=web

# Default target
.DEFAULT_GOAL := help

# Build binaries
.PHONY: build
build:
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(MASTER_BINARY) cmd/master/main.go
	$(GOBUILD) -o $(AGENT_BINARY) cmd/agent/main.go
	@echo "Binaries built: $(MASTER_BINARY), $(AGENT_BINARY)"

# Run master
.PHONY: run-master
run-master:
	$(GOCMD) run cmd/master/main.go

# Run agent
.PHONY: run-agent
run-agent:
	$(GOCMD) run cmd/agent/main.go

# Run frontend development server
.PHONY: run-frontend
run-frontend:
	cd $(WEB_DIR) && npm run dev

# Build frontend for production
.PHONY: build-frontend
build-frontend:
	cd $(WEB_DIR) && npm run build

# Apply all migrations (up)
.PHONY: migrate-up
migrate-up:
	@echo "Applying all migrations..."
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/001_init.up.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/002_add_mac_up.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/003_add_devices_up.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/004_add_snapshots.up.sql
	@echo "Migrations applied."

# Rollback all migrations (down)
.PHONY: migrate-down
migrate-down:
	@echo "Rolling back all migrations..."
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/004_add_snapshots.down.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/003_add_devices.down.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/002_add_mac_down.sql
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/001_init.down.sql
	@echo "Migrations rolled back."

# Apply a specific migration (up) by name, e.g.: make migrate-apply NAME=004_add_snapshots
.PHONY: migrate-apply
migrate-apply:
ifndef NAME
	@echo "Error: NAME is not set. Usage: make migrate-apply NAME=004_add_snapshots"
	exit 1
endif
	psql -h $(PG_HOST) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/$(NAME).up.sql
	@echo "Migration $(NAME) applied."

# Run all tests (unit)
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run unit tests only (excluding integration)
.PHONY: test-unit
test-unit:
	$(GOTEST) -v ./internal/... ./storage/...

# Run integration tests (requires PostgreSQL running and test database)
.PHONY: test-integration
test-integration:
	$(GOTEST) -v -tags=integration ./tests/integration/...

# Run tests with coverage report
.PHONY: test-cover
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	rm -rf $(WEB_DIR)/dist
	@echo "Cleaned."

# Install frontend dependencies
.PHONY: install-frontend
install-frontend:
	cd $(WEB_DIR) && npm install

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run ./...

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build          - build master and agent binaries"
	@echo "  make run-master     - run master server"
	@echo "  make migrate-up     - apply all database migrations"
	@echo "  make test           - run all tests"