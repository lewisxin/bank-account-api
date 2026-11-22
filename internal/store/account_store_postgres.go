package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type PostgresAccounts struct {
	db *sql.DB
}

func NewPostgresAccounts(db *sql.DB) *PostgresAccounts {
	return &PostgresAccounts{db: db}
}

func (s *PostgresAccounts) AddAccount(ctx context.Context, initialBalanceCents int64) (Account, error) {
	var created Account
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO accounts (balance_cents) VALUES ($1)
		RETURNING id, balance_cents`, initialBalanceCents).Scan(&created.ID, &created.BalanceCents)
	if err != nil {
		return Account{}, fmt.Errorf("insert account: %w", err)
	}
	return created, nil
}

func (s *PostgresAccounts) Deposit(ctx context.Context, id, amountCents int64) (Account, error) {
	if amountCents <= 0 {
		return Account{}, ErrInvalidDepositAmount
	}

	var updated Account
	err := s.db.QueryRowContext(ctx, `
		UPDATE accounts SET balance_cents = balance_cents + $2 WHERE id = $1
		RETURNING id, balance_cents`, id, amountCents).Scan(&updated.ID, &updated.BalanceCents)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if pq.As(err, pqerror.NumericValueOutOfRange) != nil {
		return Account{}, ErrBalanceOverflow
	}
	if err != nil {
		return Account{}, fmt.Errorf("update account balance: %w", err)
	}
	return updated, nil
}

func (s *PostgresAccounts) Transfer(ctx context.Context, fromID, toID, amountCents int64) (Account, Account, error) {
	if err := validateTransfer(fromID, toID, amountCents); err != nil {
		return Account{}, Account{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, Account{}, fmt.Errorf("begin transfer: %w", err)
	}
	defer tx.Rollback()

	// always lock in ID order to prevent deadlocks
	from, err := lockAccount(ctx, tx, min(fromID, toID))
	if err != nil {
		return Account{}, Account{}, err
	}
	to, err := lockAccount(ctx, tx, max(fromID, toID))
	if err != nil {
		return Account{}, Account{}, err
	}
	if from.ID != fromID {
		from, to = to, from
	}
	if err := applyTransfer(&from, &to, amountCents); err != nil {
		return Account{}, Account{}, err
	}

	for _, account := range []Account{from, to} {
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET balance_cents = $2 WHERE id = $1`, account.ID, account.BalanceCents); err != nil {
			return Account{}, Account{}, fmt.Errorf("update transfer balance: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Account{}, Account{}, fmt.Errorf("commit transfer: %w", err)
	}
	return from, to, nil
}

func lockAccount(ctx context.Context, tx *sql.Tx, id int64) (Account, error) {
	var current Account
	err := tx.QueryRowContext(ctx, `SELECT id, balance_cents FROM accounts WHERE id = $1 FOR UPDATE`, id).
		Scan(&current.ID, &current.BalanceCents)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("lock account: %w", err)
	}
	return current, nil
}
