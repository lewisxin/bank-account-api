# Bank Account API

## Overview
This is an demo project for a small bank account API developed in Golang.

## Functional Requirements
### Core Requirements
- R1: Create an account with or without an intial balance
- R2: Deposit money to account
- R3: Transfer money between accounts

### Additional Requirements

### Out of Scope


## Folder Structure

## Build and Run

### Prerequisites

- Go 1.27.4 or newer
- Docker with Docker Compose
- `golang-migrate` CLI (`migrate`) with PostgreSQL support

When installing `migrate` with Go, include the PostgreSQL build tag.

```sh
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Run `migrate -help` to confirm `postgres` appears under the database drivers.

### Building

```sh
go build -o bank-account-api .
```

### Starting the Server

Start PostgreSQL and apply the [database migrations](#database-migrations), then run the API from the project root.

```sh
go run .
```

You can also start the API with `./bank-account-api` after building it. The server listens on `http://127.0.0.1:8080`.

## Database Migrations

### Start the Local Database

```sh
docker compose up -d --wait db
```

### Add a Migration

Run this from the project root to create the next numbered pair of SQL files.

```sh
migrate create -ext sql -dir migrations -seq add_account_status
```

Write the schema change in the `.up.sql` file and its rollback in the matching `.down.sql` file.

### Apply or Roll Back Migrations

Set `DATABASE_URL` to the database you want to migrate. Use the same URL when starting the API.

```sh
export DATABASE_URL='postgres://bank:bank@127.0.0.1:5432/bank?sslmode=disable'
migrate -path migrations -database "$DATABASE_URL" up
```

To roll back the latest migration, run the following command.

```sh
migrate -path migrations -database "$DATABASE_URL" down 1
```

Rolling back the initial migration drops the `accounts` table and its rows.

## Testing

### Unit Tests

```sh
go test ./...
```

### PostgreSQL Integration Test

With the Compose database running, run the store tests against it.

```sh
TEST_DATABASE_URL='postgres://bank:bank@127.0.0.1:5432/bank?sslmode=disable' go test -count=1 ./internal/store
```
