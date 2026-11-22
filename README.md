# Bank Account API

## Overview
This is a demo project for a small bank account API written in Go.

### Folder Structure

```text
bank-account-api/
├── docs/
│   └── openapi.yml                      # HTTP API specification
├── e2e/
│   └── account_e2e.postman_collection.json
├── internal/
│   ├── handler/                         # Account HTTP handlers and unit tests
│   ├── middleware/                      # Structured request logging and tests
│   └── store/                           # Account types, PostgreSQL access, and store tests
├── migrations/                          # Versioned SQL migrations and rollbacks
├── scripts/                             # E2E runner and shared test project setup
├── main.go                              # Server startup and route registration
├── Dockerfile                           # API container image
├── docker-compose.yml                   # Local API, PostgreSQL, and migration services
├── docker-compose.test.yml              # Shared integration and E2E test overrides
└── Makefile                             # Build, run, test, and migration commands
```

## Quick Start

With Docker, Docker Compose, and `make` installed, run this from the project root.

```sh
make run-stack
```

This builds the API image, starts PostgreSQL and waits for it to accept connections, applies pending migrations, and starts the server at `http://127.0.0.1:8080`.

The [OpenAPI specification](docs/openapi.yml) documents the account, deposit, and transfer endpoints and their responses.

## Testing

### Unit Tests

Run the unit tests with the following command.

```sh
make test-unit
```

### Integration Test with PostgreSQL

> Requires Docker Compose

Run this from the project root. Integration test verifies actual postgres connection to a local Postgres running in Docker.

```sh
make test-integration
```


### End-to-End Tests

> Requires Docker Compose, `curl`, and the Postman CLI installed. 

Run this from the project root.

```sh
make test-e2e
```

Each run builds the API image, starts a fresh PostgreSQL container, applies migrations, and starts the server on a temporary local port. It runs the [account workflow collection](e2e/account_e2e.postman_collection.json), then removes the E2E containers and their database data, even if a test fails. Run the same command again for a clean database.

The collection uses `GET /account/{id}` to verify stored balances after account creation, deposits, and transfers. It also checks that rejected deposits and transfers leave the balances intact.


## Server Logs

Use `docker ps` to find the running `bank-account-api` container. Replace `<container_id_or_name>` with its ID or name to follow the server logs.

```sh
docker ps
docker logs -f <container_id_or_name>
```

## Build the Image

```sh
make build-docker
```

## Database Migrations

### Start PostgreSQL and Apply Migrations

For database work, run these commands in order. The API and migrations default to the Compose database. Set `DATABASE_URL` to use a different PostgreSQL server reachable from the containers.

```sh
make run-db
make migrate-up
```

### Add a Migration

Install the `golang-migrate` CLI to create migration files.

```sh
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

From the project root, replace `name_of_the_migration` with a descriptive name to create the next numbered pair of SQL files.

```sh
make migrate-create MIGRATION_NAME=name_of_the_migration
```

Write the schema change in the `.up.sql` file and its rollback in the matching `.down.sql` file.

### Roll Back a Migration

To roll back the latest migration, run the following command.

```sh
make migrate-down
```

Rolling back the initial migration drops the `accounts` table and its rows.


## Functional Requirements

- **R1**: Create an account with an optional initial balance.
- **R2**: Deposit money into an account.
- **R3**: Transfer money between accounts atomically.
- **R4**: Retrieve an account and its current balance by ID.

## Technical Design Decisions

- **Integer cents for monetary values**: Used integers in cents for account balances and transfer amounts to avoid floating-point rounding errors.
- **PostgreSQL for transactional consistency**: Used database transactions to guarantee atomic transfers and row-level locking to protect account balances during concurrent operations.
- **Consistent lock ordering**: Used consistent lock order by account ID to prevent deadlocks caused by concurrent transfers in opposite directions.
- **Interface-based dependency injection**: Defined an account store interface to decouple HTTP handlers from PostgreSQL, enabling unit tests with mock implementations.

## Out of Scope

The demo project left out certain implementation that are important for making it production-ready, such as the following:

- **Authentication**: Require login for `GET /account/{id}`, `POST /account/{id}/deposit`, and `POST /account/transfer`. Return `401` for missing or invalid credentials.
- **Authorization**: Restrict account reads to the owner. Allow a deposit into an account or a transfer from an account only for its owner or an authenticated caller with permission for that operation. Return `403` when an authenticated caller lacks permission.
- **Idempotency**: Write operations like deposit or transfer may be called multiple times due to form re-submission or timeout and retry. Accept idempotency keys may mitigate this issue.
- **CI/CD**: Automate builds, unit tests, integration tests, and E2E tests, then deploy the tested application image to a server or cloud environment.
- **Caching**: Consider caching `GET /account/{id}` responses to reduce PostgreSQL load. Define freshness requirements and invalidate affected entries after committed write operations deposits and transfers.
- **Observability**: Basic logging has been added to the project, but consider more comprehensive instrumentation such as OpenTelemetry or Datadog agent to monitor and alert server health such as request throughput, latency, and error rates.
- **Other Features**: Transaction history, multi-currency transfers.

