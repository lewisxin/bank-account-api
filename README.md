# Bank Account API

## Overview
This is an demo project for a small bank account API developed in Golang.

## Functional Requirements
### Core Requirements
- R1: Create an account with or without an intial balance
- R2: Deposit money to account
- R3: Transfer money between accounts
- R4: Retrieve account details


### Additional Requirements

### Out of Scope


## Folder Structure

## Build and Run

### Prerequisite

- Go 1.27.4 or newer

### Building

```sh
go build -o bank-account-api .
```

### Starting the Server

Run the compiled binary with `./bank-account-api`, or run the package directly.

```sh
go run .
```

The server listens on `http://127.0.0.1:8080`.

## Testing

### Unit Tests

```sh
go test ./...
```