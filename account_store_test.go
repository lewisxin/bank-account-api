package main

import (
	"math"
	"sync"
	"testing"
)

func TestAccountStoreAddAccountIncrementsID(t *testing.T) {
	store := newAccountStore()
	first := store.addAccount(2500)
	second := store.addAccount(0)

	if first.ID != 1 || second.ID != 2 {
		t.Errorf("account IDs = %d and %d, want 1 and 2", first.ID, second.ID)
	}
}

func TestAccountStoreDepositUpdatesBalance(t *testing.T) {
	store := newAccountStore()
	created := store.addAccount(9500)
	if _, err := store.deposit(created.ID, 1500); err != nil {
		t.Fatalf("deposit: %v", err)
	}
	if balance := store.accounts[created.ID].BalanceCents; balance != 11000 {
		t.Errorf("stored balance = %d, want 11000", balance)
	}
}

func TestAccountStoreDepositErrorsKeepBalance(t *testing.T) {
	tests := []struct {
		name        string
		id          int64
		amountCents int64
		expectedErr error
	}{
		{name: "missing account", id: 2, amountCents: 100, expectedErr: errAccountNotFound},
		{name: "balance overflow", id: 1, amountCents: math.MaxInt64, expectedErr: errBalanceOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAccountStore()
			created := store.addAccount(2500)
			if _, err := store.deposit(tt.id, tt.amountCents); err != tt.expectedErr {
				t.Fatalf("deposit error = %v, want %v", err, tt.expectedErr)
			}
			if balance := store.accounts[created.ID].BalanceCents; balance != created.BalanceCents {
				t.Errorf("stored balance = %d, want %d", balance, created.BalanceCents)
			}
		})
	}
}

func TestAccountStoreTransferUpdatesBothBalances(t *testing.T) {
	store := newAccountStore()
	from := store.addAccount(2500)
	to := store.addAccount(1000)
	if _, _, err := store.transfer(from.ID, to.ID, 500); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if debit, credit := store.accounts[from.ID].BalanceCents, store.accounts[to.ID].BalanceCents; debit != 2000 || credit != 1500 {
		t.Errorf("stored balances = %d and %d, want 2000 and 1500", debit, credit)
	}
}

func TestAccountStoreTransferErrorsKeepBalances(t *testing.T) {
	tests := []struct {
		name               string
		toID               int64
		amountCents        int64
		destinationBalance int64
		expectedErr        error
	}{
		{name: "same account", toID: 1, amountCents: 100, expectedErr: errInvalidTransferAccounts},
		{name: "non-positive amount", toID: 2, amountCents: 0, expectedErr: errInvalidTransferAmount},
		{name: "missing destination", toID: 3, amountCents: 100, expectedErr: errAccountNotFound},
		{name: "insufficient funds", toID: 2, amountCents: 1001, expectedErr: errInsufficientFunds},
		{name: "destination overflow", toID: 2, amountCents: 100, destinationBalance: math.MaxInt64, expectedErr: errBalanceOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAccountStore()
			from := store.addAccount(1000)
			to := store.addAccount(tt.destinationBalance)
			if _, _, err := store.transfer(from.ID, tt.toID, tt.amountCents); err != tt.expectedErr {
				t.Fatalf("transfer error = %v, want %v", err, tt.expectedErr)
			}
			if debit, credit := store.accounts[from.ID].BalanceCents, store.accounts[to.ID].BalanceCents; debit != from.BalanceCents || credit != to.BalanceCents {
				t.Errorf("stored balances = %d and %d, want %d and %d", debit, credit, from.BalanceCents, to.BalanceCents)
			}
		})
	}
}

func TestConcurrentTransfersRespectAvailableBalance(t *testing.T) {
	store := newAccountStore()
	from := store.addAccount(100)
	to := store.addAccount(0)
	results := make(chan error, 2)
	var transfers sync.WaitGroup
	for range 2 {
		transfers.Go(func() {
			_, _, err := store.transfer(from.ID, to.ID, 100)
			results <- err
		})
	}
	transfers.Wait()
	close(results)

	succeeded := 0
	for err := range results {
		switch err {
		case nil:
			succeeded++
		case errInsufficientFunds:
		default:
			t.Errorf("unexpected transfer error: %v", err)
		}
	}
	if succeeded != 1 {
		t.Errorf("successful transfers = %d, want 1", succeeded)
	}
	if debit, credit := store.accounts[from.ID].BalanceCents, store.accounts[to.ID].BalanceCents; debit != 0 || credit != 100 {
		t.Errorf("stored balances = %d and %d, want 0 and 100", debit, credit)
	}
}
