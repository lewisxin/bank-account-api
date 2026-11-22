package store

import (
	"errors"
	"math"
)

type Account struct {
	ID           int64 `json:"id"`
	BalanceCents int64 `json:"balance_cents"`
}

var (
	ErrAccountNotFound         = errors.New("account not found")
	ErrBalanceOverflow         = errors.New("balance would exceed the maximum")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrInvalidDepositAmount    = errors.New("amount_cents must be a positive integer")
	ErrInvalidTransferAccounts = errors.New("from_account_id and to_account_id must be distinct positive integers")
	ErrInvalidTransferAmount   = errors.New("amount_cents must be a positive integer")
)

func validateTransfer(fromID, toID, amountCents int64) error {
	if fromID <= 0 || toID <= 0 || fromID == toID {
		return ErrInvalidTransferAccounts
	}
	if amountCents <= 0 {
		return ErrInvalidTransferAmount
	}
	return nil
}

func applyTransfer(from, to *Account, amountCents int64) error {
	if from.BalanceCents < amountCents {
		return ErrInsufficientFunds
	}
	if amountCents > math.MaxInt64-to.BalanceCents {
		return ErrBalanceOverflow
	}
	from.BalanceCents -= amountCents
	to.BalanceCents += amountCents
	return nil
}
