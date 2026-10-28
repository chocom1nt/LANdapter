# LANdapter development commands
# Quick start: make setup -> make up -> make install-frontend
# The setup scripts preserve existing secrets and fill in missing values.

GOCMD ?= go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
BINARY_DIR = bin
WEB_DIR = web
MIGRATIONS_DIR = migrations

# Detect the host OS first. PowerShell commands work when Make is invoked from
# PowerShell, cmd.exe, Git Bash, or MSYS, and avoid POSIX flags reaching cmd.exe.
WINDOWS_HOST := $(filter Windows_NT,$(OS))

ifneq ($(WINDOWS_HOST),)
BINARY_EXT = .exe
CREATE_BIN = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(BINARY_DIR)' | Out-Null"
REMOVE_DIR = powershell -NoProfile -Command "if (Test-Path '$(BINARY_DIR)') { Remove-Item -LiteralPath '$(BINARY_DIR)' -Recurse -Force }"
REMOVE_COVERAGE = powershell -NoProfile -Command "Remove-Item -Force -ErrorAction SilentlyContinue 'coverage.out','coverage.html'"
REMOVE_WEB_DIST = powershell -NoProfile -Command "if (Test-Path '$(WEB_DIR)/dist') { Remove-Item -LiteralPath '$(WEB_DIR)/dist' -Recurse -Force }"
SETUP_CMD = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/Setup.ps1
COPY_TOKEN_CMD = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/CopyAdminToken.ps1
RESET_CMD = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/factory-reset.ps1
CLEAR_CLIENTS_CMD = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/clear-all-clients.ps1
else
CREATE_BIN = mkdir -p "$(BINARY_DIR)"
REMOVE_DIR = rm -rf "$(BINARY_DIR)"
REMOVE_COVERAGE = rm -f coverage.out coverage.html
REMOVE_WEB_DIST = rm -rf "$(WEB_DIR)/dist"
SETUP_CMD = bash scripts/setup.sh
COPY_TOKEN_CMD = bash scripts/copy-admin-token.sh
RESET_CMD = bash scripts/factory-reset.sh
CLEAR_CLIENTS_CMD = bash scripts/clear-all-clients.sh
endif

MASTER_BINARY = $(BINARY_DIR)/master$(BINARY_EXT)
AGENT_BINARY = $(BINARY_DIR)/agent$(BINARY_EXT)
TUI_BINARY = $(BINARY_DIR)/lndp$(BINARY_EXT)

PG_HOST ?= localhost
PG_PORT ?= 5432
PG_USER ?= $(if $(DB_USER),$(DB_USER),landapter)
PG_PASSWORD ?= $(DB_PASSWORD)
PGPASSWORD ?= $(PG_PASSWORD)
export PGPASSWORD
PG_DB ?= landapter

.DEFAULT_GOAL := help

.PHONY: setup init quickstart copy-token reset clear-clients up down logs build build-cli build-tui run-master run-agent run-tui run-frontend build-frontend migrate-up migrate-down migrate-apply test test-unit test-integration test-cover seed-demo-agents clear-demo-agents clean install-frontend lint help

setup init:
	$(SETUP_CMD)

# Idempotent first-run path: setup secrets/config, install UI deps, start Master+Postgres.
quickstart:
	$(SETUP_CMD)
	npm ci --prefix "$(WEB_DIR)"
	docker compose up -d --build

copy-token:
	$(COPY_TOKEN_CMD)

reset:
	$(RESET_CMD)

# Destructive: deletes every client record after an explicit typed confirmation.
clear-clients:
	$(CLEAR_CLIENTS_CMD)

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

build:
	@$(CREATE_BIN)
	$(GOBUILD) -o "$(MASTER_BINARY)" ./cmd/master
	$(GOBUILD) -o "$(AGENT_BINARY)" ./cmd/agent
	@echo "Master and Agent binaries built in $(BINARY_DIR)/"

# The former master-cli command has been consolidated into the interactive lndp TUI.
build-cli: build-tui

build-tui:
	@$(CREATE_BIN)
	$(GOBUILD) -o "$(TUI_BINARY)" ./cmd/lndp
	@echo "TUI binary built: $(TUI_BINARY)"

run-master:
	$(GOCMD) run ./cmd/master

run-agent:
	$(GOCMD) run ./cmd/agent

run-tui:
	$(GOCMD) run ./cmd/lndp

run-frontend:
	cd "$(WEB_DIR)" && npm run dev

build-frontend:
	cd "$(WEB_DIR)" && npm run build

migrate-up:
	@echo "Applying database migrations..."
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/001_init.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/002_add_mac.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/003_add_devices.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/004_add_snapshots.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/005_agent_credentials.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/006_fix_clients_schema.up.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/007_policies_schedules.up.sql
	@echo "Migrations applied."

migrate-down:
	@echo "Rolling back all migrations..."
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/007_policies_schedules.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/006_fix_clients_schema.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/005_agent_credentials.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/004_add_snapshots.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/003_add_devices.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/002_add_mac.down.sql
	psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_DB) -f $(MIGRATIONS_DIR)/001_init.down.sql
	@echo "Migrations rolled back."

migrate-apply:
ifndef NAME
	@echo "Error: NAME is not set. Example: make migrate-apply NAME=007_policies_schedules"
	@exit 1
endif
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/apply-migration.ps1 -Name "$(NAME)"
else
	@test -f "$(MIGRATIONS_DIR)/$(NAME).up.sql" || (echo "Migration file not found: $(MIGRATIONS_DIR)/$(NAME).up.sql" && exit 1)
	docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < "$(MIGRATIONS_DIR)/$(NAME).up.sql"
endif

test:
	$(GOTEST) ./...
	cd "$(WEB_DIR)" && npm test

# Feed SQL through PowerShell on Windows; native cmd.exe does not treat single
# quotes as quoting characters, and shell redirection here is brittle.
ifeq ($(OS),Windows_NT)
seed-demo-agents:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/seed-demo-agents.ps1

clear-demo-agents:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/clear-demo-agents.ps1
else
seed-demo-agents:
	@test -f scripts/dev-seed-agents.sql || (echo "Missing scripts/dev-seed-agents.sql; run make from the repository root." && exit 1)
	docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < scripts/dev-seed-agents.sql

clear-demo-agents:
	@test -f scripts/dev-clear-demo-agents.sql || (echo "Missing scripts/dev-clear-demo-agents.sql; run make from the repository root." && exit 1)
	docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < scripts/dev-clear-demo-agents.sql
endif

test-unit:
	$(GOTEST) ./internal/... ./storage/... ./cmd/lndp/...
	cd "$(WEB_DIR)" && npm test

test-integration:
	$(GOTEST) -tags=integration ./tests/integration/...

test-cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean:
	$(REMOVE_DIR)
	$(REMOVE_COVERAGE)
	$(REMOVE_WEB_DIST)
	@echo "Cleaned build outputs."

install-frontend:
	cd "$(WEB_DIR)" && npm ci

lint:
	golangci-lint run ./...

help:
	@echo LANdapter development commands:
	@echo   setup / init       Generate missing secrets and local config files
	@echo   quickstart         Setup, install frontend dependencies, start Docker services
	@echo   copy-token         Copy admin token for the first web Settings login
	@echo   reset              Destructive factory reset with explicit confirmation
	@echo   clear-clients      Delete all client records with explicit confirmation
	@echo   up / down          Start or stop Docker Compose services
	@echo   logs               Follow Docker Compose logs
	@echo   build              Build Master and Agent
	@echo   build-tui          Build the experimental lndp terminal UI
	@echo   build-cli          Compatibility alias for build-tui
	@echo   run-master         Run Master locally, loading .env automatically
	@echo   run-agent          Run the Agent locally
	@echo   run-tui            Run lndp locally, loading .env automatically
	@echo   run-frontend       Start the Vite development server
	@echo   build-frontend     Build the frontend
	@echo   migrate-up/down    Apply or roll back SQL migrations
	@echo   migrate-apply      Apply one migration with NAME variable
	@echo   seed-demo-agents   Insert deterministic demo clients for UI/TUI smoke tests
	@echo   clear-demo-agents  Remove only the demo clients
	@echo   test / test-unit   Run Go and frontend translation tests
	@echo   test-integration   Run integration tests
	@echo   test-cover         Generate Go coverage report
	@echo   install-frontend   Install frontend dependencies with npm ci
	@echo   lint               Run golangci-lint
	@echo   clean              Remove build and coverage outputs
