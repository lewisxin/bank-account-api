# Bank Account API

## Overview
This is a demo project for a small bank account API written in Go.

## Quick Start

With Docker, Docker Compose, and `make` installed, run this from the project root.

```sh
make up
```

This builds the API image, starts PostgreSQL and waits for it to accept connections, applies pending migrations, and starts the server at `http://127.0.0.1:8080`.

The [OpenAPI specification](docs/openapi.yml) documents the account, deposit, and transfer endpoints and their responses.

## Build the Image

```sh
make docker-build
```

## Database Migrations

### Start PostgreSQL and Apply Migrations

For database work, run these commands in order. The API and migrations default to the Compose database. Set `DATABASE_URL` to use a different PostgreSQL server reachable from the containers.

```sh
make db-up
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

## Testing

### Unit Tests

Run the unit tests with Go 1.27.4 or newer. These tests run without PostgreSQL.

```sh
make test
```

### Full Suite With PostgreSQL

```sh
make test-integration
```

Compose starts the dedicated `test-db` service if needed and waits for it to be ready. The `integration` build tag includes the PostgreSQL tests alongside the unit tests. The PostgreSQL tests create temporary schemas in `bank_test`, apply their migrations, and drop the schemas during cleanup.

### End-to-End Tests

With Docker Compose 2.24.4 or newer, `curl`, and the Postman CLI installed, run this from the project root.

```sh
make test-e2e
```

Each run builds the API image, starts a fresh PostgreSQL container, applies migrations, and starts the server on a temporary local port. It runs the [account workflow collection](e2e/account_e2e.postman_collection.json), then removes the E2E containers and their database data, even if a test fails. Run the same command again for a clean database.

The collection uses `GET /account/{id}` to verify stored balances after account creation, deposits, and transfers. It also checks that rejected deposits and transfers leave the balances intact.

## Functional Requirements

- **R1** Create an account with an optional initial balance.
- **R2** Deposit money into an account.
- **R3** Transfer money between accounts atomically.
- **R4** Retrieve an account and its current balance by ID.
