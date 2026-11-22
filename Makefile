GO ?= go
COMPOSE ?= docker compose
MIGRATE ?= migrate
POSTMAN ?= postman
export MIGRATION_NAME

.PHONY: build run docker-build docker-run up db-up test test-integration test-e2e migrate-create migrate-up migrate-down

build:
	$(GO) build -o bank-account-api .

run:
	$(GO) run .

docker-build:
	$(COMPOSE) build bank-account-api

docker-run:
	$(COMPOSE) up -d bank-account-api

up:
	$(MAKE) docker-build
	$(MAKE) db-up
	$(MAKE) migrate-up
	$(MAKE) docker-run

db-up:
	$(COMPOSE) up -d --wait db

test:
	$(GO) test ./...

test-integration:
	@set -eu; \
	. ./scripts/test-project.sh; \
	prepare_test_project integration "$(COMPOSE)"; \
	$$test_compose run --rm -T test-integration

test-e2e:
	sh scripts/run-e2e.sh "$(COMPOSE)" "$(POSTMAN)"

migrate-create:
	@test -n "$$MIGRATION_NAME" || { printf '%s\n' 'Usage: make migrate-create MIGRATION_NAME=add_account_status' >&2; exit 1; }
	$(MIGRATE) create -ext sql -dir migrations -seq "$$MIGRATION_NAME"

migrate-up:
	$(COMPOSE) run --rm -T migrate up

migrate-down:
	$(COMPOSE) run --rm -T migrate down 1
