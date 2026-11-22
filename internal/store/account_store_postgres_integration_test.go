//go:build integration

package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

func openTestPostgres(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	config, err := pq.NewConfig(url)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	adminConnector, err := pq.NewConnectorConfig(config)
	if err != nil {
		t.Fatalf("create test admin connector: %v", err)
	}
	admin := sql.OpenDB(adminConnector)
	t.Cleanup(func() { admin.Close() })

	schema := "bank_test_" + strings.ToLower(rand.Text())
	identifier := pq.QuoteIdentifier(schema)
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})

	testConfig := config.Clone()
	testConfig.Runtime["search_path"] = schema
	testConnector, err := pq.NewConnectorConfig(testConfig)
	if err != nil {
		t.Fatalf("create test connector: %v", err)
	}
	db := sql.OpenDB(testConnector)
	t.Cleanup(func() { db.Close() })
	migration, err := os.ReadFile("../../migrations/000001_create_accounts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.ExecContext(t.Context(), string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	return db
}

func addTestAccount(t *testing.T, accounts *PostgresAccounts, balanceCents int64) Account {
	t.Helper()
	created, err := accounts.AddAccount(t.Context(), balanceCents)
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	return created
}

func TestPostgresAddAccount(t *testing.T) {
	db := openTestPostgres(t)
	accounts := NewPostgresAccounts(db)
	var ids [16]int64
	runConcurrently(len(ids), func(i int) {
		created, err := accounts.AddAccount(t.Context(), 2500)
		if err != nil {
			t.Errorf("add account %d: %v", i, err)
			return
		}
		ids[i] = created.ID
		if created.BalanceCents != 2500 {
			t.Errorf("account %d balance = %d, want 2500", created.ID, created.BalanceCents)
		}
	})

	slices.Sort(ids[:])
	for i, id := range ids {
		if want := int64(i + 1); id != want {
			t.Fatalf("concurrent account IDs = %v, want consecutive IDs from 1 to %d", ids, len(ids))
		}
	}
}

func TestPostgresGetAccount(t *testing.T) {
	accounts := NewPostgresAccounts(openTestPostgres(t))
	created := addTestAccount(t, accounts, 900)
	got, err := accounts.GetAccount(t.Context(), created.ID)
	if err != nil || got != created {
		t.Fatalf("GetAccount(%d) = %+v, %v; want %+v", created.ID, got, err, created)
	}
	if _, err := accounts.GetAccount(t.Context(), created.ID+1); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("get missing account: %v, want %v", err, ErrAccountNotFound)
	}
}

func TestPostgresDeposit(t *testing.T) {
	db := openTestPostgres(t)
	accounts := NewPostgresAccounts(db)

	t.Run("rejects a missing account", func(t *testing.T) {
		if _, err := accounts.Deposit(t.Context(), 999, 100); !errors.Is(err, ErrAccountNotFound) {
			t.Errorf("deposit error = %v, want %v", err, ErrAccountNotFound)
		}
	})

	t.Run("rejects non-positive amounts", func(t *testing.T) {
		account := addTestAccount(t, accounts, 2500)
		for _, amountCents := range []int64{0, -1, math.MinInt64} {
			if _, err := accounts.Deposit(t.Context(), account.ID, amountCents); !errors.Is(err, ErrInvalidDepositAmount) {
				t.Errorf("deposit %d cents error = %v, want %v", amountCents, err, ErrInvalidDepositAmount)
			}
			assertPostgresBalance(t, db, account.ID, 2500)
		}
	})

	t.Run("rejects overflow", func(t *testing.T) {
		account := addTestAccount(t, accounts, 0)
		updated, err := accounts.Deposit(t.Context(), account.ID, math.MaxInt64)
		if err != nil || updated.ID != account.ID || updated.BalanceCents != math.MaxInt64 {
			t.Fatalf("deposit to maximum balance = %+v, %v", updated, err)
		}
		if _, err := accounts.Deposit(t.Context(), account.ID, 1); !errors.Is(err, ErrBalanceOverflow) {
			t.Errorf("overflow deposit error = %v, want %v", err, ErrBalanceOverflow)
		}
		assertPostgresBalance(t, db, account.ID, math.MaxInt64)
	})

	t.Run("concurrent deposits update the same account", func(t *testing.T) {
		const deposits = 16
		const amountCents = 100
		account := addTestAccount(t, accounts, 2500)
		runConcurrently(deposits, func(i int) {
			if _, err := accounts.Deposit(t.Context(), account.ID, amountCents); err != nil {
				t.Errorf("deposit %d: %v", i, err)
			}
		})
		assertPostgresBalance(t, db, account.ID, 2500+deposits*amountCents)
	})
}

func TestPostgresTransfer(t *testing.T) {
	t.Run("rejected transfers keep both balances", func(t *testing.T) {
		db := openTestPostgres(t)
		accounts := NewPostgresAccounts(db)
		from := addTestAccount(t, accounts, 1000)
		to := addTestAccount(t, accounts, math.MaxInt64)
		tests := []struct {
			name        string
			toID        int64
			amountCents int64
			expectedErr error
		}{
			{name: "same account", toID: from.ID, amountCents: 100, expectedErr: ErrInvalidTransferAccounts},
			{name: "zero amount", toID: to.ID, amountCents: 0, expectedErr: ErrInvalidTransferAmount},
			{name: "missing destination", toID: 999, amountCents: 100, expectedErr: ErrAccountNotFound},
			{name: "destination overflow", toID: to.ID, amountCents: 100, expectedErr: ErrBalanceOverflow},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, _, err := accounts.Transfer(t.Context(), from.ID, tt.toID, tt.amountCents); !errors.Is(err, tt.expectedErr) {
					t.Errorf("transfer error = %v, want %v", err, tt.expectedErr)
				}
				assertPostgresBalance(t, db, from.ID, from.BalanceCents)
				assertPostgresBalance(t, db, to.ID, to.BalanceCents)
			})
		}
	})

	t.Run("concurrent transfers in opposite directions", func(t *testing.T) {
		db := openTestPostgres(t)
		accounts := NewPostgresAccounts(db)
		a := addTestAccount(t, accounts, 1000)
		b := addTestAccount(t, accounts, 1000)
		transfers := []struct {
			fromID, toID, amountCents int64
		}{
			{fromID: a.ID, toID: b.ID, amountCents: 300},
			{fromID: b.ID, toID: a.ID, amountCents: 200},
		}
		runConcurrently(len(transfers), func(i int) {
			transfer := transfers[i]
			from, to, err := accounts.Transfer(t.Context(), transfer.fromID, transfer.toID, transfer.amountCents)
			if err != nil || from.ID != transfer.fromID || to.ID != transfer.toID {
				t.Errorf("transfer %d to %d returned %+v, %+v, %v", transfer.fromID, transfer.toID, from, to, err)
			}
		})
		assertPostgresBalance(t, db, a.ID, 900)  // 1000 - 300 + 200 = 900
		assertPostgresBalance(t, db, b.ID, 1100) // 1000 + 300 - 200 = 1100
	})

	t.Run("concurrent transfers from one account to different accounts", func(t *testing.T) {
		db := openTestPostgres(t)
		accounts := NewPostgresAccounts(db)
		const transferAmountCents = 800
		tests := []struct {
			name                 string
			initialBalance       int64
			expectedSuccesses    int
			expectedFinalBalance int64
		}{
			{name: "enough funds for both", initialBalance: 2000, expectedSuccesses: 2, expectedFinalBalance: 400}, // 2000 - 2*800 = 400
			{name: "enough funds for one", initialBalance: 1000, expectedSuccesses: 1, expectedFinalBalance: 200},  // 1000 - 800 = 200
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := addTestAccount(t, accounts, tt.initialBalance)
				b := addTestAccount(t, accounts, 0)
				c := addTestAccount(t, accounts, 0)
				destinations := [...]Account{b, c}
				var results [len(destinations)]error
				runConcurrently(len(results), func(i int) {
					_, _, results[i] = accounts.Transfer(t.Context(), a.ID, destinations[i].ID, transferAmountCents)
				})

				succeeded := 0
				for i, err := range results {
					var expectedBalance int64
					switch {
					case err == nil:
						succeeded++
						expectedBalance = transferAmountCents
					case errors.Is(err, ErrInsufficientFunds):
					default:
						t.Fatalf("transfer to account %d: %v", destinations[i].ID, err)
					}
					assertPostgresBalance(t, db, destinations[i].ID, expectedBalance)
				}
				if succeeded != tt.expectedSuccesses {
					t.Errorf("successful transfers = %d, want %d", succeeded, tt.expectedSuccesses)
				}
				assertPostgresBalance(t, db, a.ID, tt.expectedFinalBalance)
			})
		}
	})

	t.Run("rolls back the debit when the credit fails", func(t *testing.T) {
		db := openTestPostgres(t)
		accounts := NewPostgresAccounts(db)
		from := addTestAccount(t, accounts, 1000)
		to := addTestAccount(t, accounts, 0)

		if _, err := db.ExecContext(t.Context(), `ALTER TABLE accounts ADD CHECK (id <> 2 OR balance_cents = 0)`); err != nil {
			t.Fatalf("add test constraint: %v", err)
		}
		if _, _, err := accounts.Transfer(t.Context(), from.ID, to.ID, 100); pq.As(err, pqerror.CheckViolation) == nil {
			t.Fatalf("transfer error = %v, want a check constraint violation", err)
		}
		assertPostgresBalance(t, db, from.ID, 1000)
		assertPostgresBalance(t, db, to.ID, 0)
	})
}

func assertPostgresBalance(t *testing.T, db *sql.DB, id, want int64) {
	t.Helper()
	var balance int64
	if err := db.QueryRowContext(t.Context(), `SELECT balance_cents FROM accounts WHERE id = $1`, id).Scan(&balance); err != nil {
		t.Fatalf("read account %d balance: %v", id, err)
	}
	if balance != want {
		t.Errorf("account %d balance = %d, want %d", id, balance, want)
	}
}

func runConcurrently(count int, run func(i int)) {
	start := make(chan struct{})
	var ready, running sync.WaitGroup
	ready.Add(count)
	for i := range count {
		running.Go(func() {
			ready.Done()
			<-start
			run(i)
		})
	}
	ready.Wait()
	close(start)
	running.Wait()
}
