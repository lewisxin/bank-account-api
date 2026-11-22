GO ?= go
COMPOSE ?= docker compose
MIGRATE ?= migrate
POSTMAN ?= postman
export MIGRATION_NAME

.PHONY: build-local build-docker \
	run-local run-docker run-db run-stack \
	test-unit test-integration test-e2e \
	migrate-create migrate-up migrate-down

build-local:
	$(GO) build -o bank-account-api .

build-docker:
	$(COMPOSE) build bank-account-api

run-local:
	$(GO) run .

run-docker:
	$(COMPOSE) up -d bank-account-api

run-db:
	$(COMPOSE) up -d --wait db

run-stack:
	$(MAKE) build-docker
	$(MAKE) run-db
	$(MAKE) migrate-up
	$(MAKE) run-docker

test-unit:
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
