package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/lewisxin/bank-account-api/internal/handler"
	"github.com/lewisxin/bank-account-api/internal/middleware"
	"github.com/lewisxin/bank-account-api/internal/store"
)

const localDatabaseURL = "postgres://bank:bank@127.0.0.1:5432/bank?sslmode=disable"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = localDatabaseURL
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	addr := os.Getenv("APP_HOST")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	slog.Info("starting HTTP server", "addr", addr)
	routes := newMux(store.NewPostgresAccounts(db))
	return http.ListenAndServe(addr, middleware.RequestLogger(slog.Default(), routes))
}

func newMux(accountStore handler.AccountStore) *http.ServeMux {
	mux := http.NewServeMux()
	accounts := handler.NewAccounts(accountStore)
	mux.HandleFunc("POST /account", accounts.Create)
	mux.HandleFunc("GET /account/{id}", accounts.Get)
	mux.HandleFunc("POST /account/{id}/deposit", accounts.Deposit)
	mux.HandleFunc("POST /account/transfer", accounts.Transfer)
	return mux
}
