package store

import (
	"math"
	"sync"
	"testing"
)

func TestAccountStoreAddAccountIncrementsID(t *testing.T) {
	store := NewAccounts()
	first := store.AddAccount(2500)
	second := store.AddAccount(0)

	if first.ID != 1 || second.ID != 2 {
		t.Errorf("account IDs = %d and %d, want 1 and 2", first.ID, second.ID)
	}
}

func TestAccountStoreDepositUpdatesBalance(t *testing.T) {
	store := NewAccounts()
	created := store.AddAccount(9500)
	if _, err := store.Deposit(created.ID, 1500); err != nil {
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
		{name: "missing account", id: 2, amountCents: 100, expectedErr: ErrAccountNotFound},
		{name: "balance overflow", id: 1, amountCents: math.MaxInt64, expectedErr: ErrBalanceOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewAccounts()
			created := store.AddAccount(2500)
			if _, err := store.Deposit(tt.id, tt.amountCents); err != tt.expectedErr {
				t.Fatalf("deposit error = %v, want %v", err, tt.expectedErr)
			}
			if balance := store.accounts[created.ID].BalanceCents; balance != created.BalanceCents {
				t.Errorf("stored balance = %d, want %d", balance, created.BalanceCents)
			}
		})
	}
}

func TestAccountStoreTransferUpdatesBothBalances(t *testing.T) {
	store := NewAccounts()
	from := store.AddAccount(2500)
	to := store.AddAccount(1000)
	if _, _, err := store.Transfer(from.ID, to.ID, 500); err != nil {
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
		{name: "same account", toID: 1, amountCents: 100, expectedErr: ErrInvalidTransferAccounts},
		{name: "non-positive amount", toID: 2, amountCents: 0, expectedErr: ErrInvalidTransferAmount},
		{name: "missing destination", toID: 3, amountCents: 100, expectedErr: ErrAccountNotFound},
		{name: "insufficient funds", toID: 2, amountCents: 1001, expectedErr: ErrInsufficientFunds},
		{name: "destination overflow", toID: 2, amountCents: 100, destinationBalance: math.MaxInt64, expectedErr: ErrBalanceOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewAccounts()
			from := store.AddAccount(1000)
			to := store.AddAccount(tt.destinationBalance)
			if _, _, err := store.Transfer(from.ID, tt.toID, tt.amountCents); err != tt.expectedErr {
				t.Fatalf("transfer error = %v, want %v", err, tt.expectedErr)
			}
			if debit, credit := store.accounts[from.ID].BalanceCents, store.accounts[to.ID].BalanceCents; debit != from.BalanceCents || credit != to.BalanceCents {
				t.Errorf("stored balances = %d and %d, want %d and %d", debit, credit, from.BalanceCents, to.BalanceCents)
			}
		})
	}
}

func TestConcurrentTransfersRespectAvailableBalance(t *testing.T) {
	store := NewAccounts()
	from := store.AddAccount(100)
	to := store.AddAccount(0)
	results := make(chan error, 2)
	var transfers sync.WaitGroup
	for range 2 {
		transfers.Go(func() {
			_, _, err := store.Transfer(from.ID, to.ID, 100)
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
		case ErrInsufficientFunds:
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
