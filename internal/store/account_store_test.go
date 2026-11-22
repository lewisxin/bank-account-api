package store

import (
	"errors"
	"math"
	"testing"
)

func TestValidateTransfer(t *testing.T) {
	tests := []struct {
		name        string
		fromID      int64
		toID        int64
		amountCents int64
		expectedErr error
	}{
		{name: "valid transfer", fromID: 1, toID: 2, amountCents: 1},
		{name: "zero source ID", fromID: 0, toID: 2, amountCents: 100, expectedErr: ErrInvalidTransferAccounts},
		{name: "negative destination ID", fromID: 1, toID: -1, amountCents: 100, expectedErr: ErrInvalidTransferAccounts},
		{name: "same account", fromID: 1, toID: 1, amountCents: 100, expectedErr: ErrInvalidTransferAccounts},
		{name: "zero amount", fromID: 1, toID: 2, amountCents: 0, expectedErr: ErrInvalidTransferAmount},
		{name: "negative amount", fromID: 1, toID: 2, amountCents: -1, expectedErr: ErrInvalidTransferAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateTransfer(tt.fromID, tt.toID, tt.amountCents); !errors.Is(err, tt.expectedErr) {
				t.Errorf("validateTransfer error = %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestApplyTransfer(t *testing.T) {
	tests := []struct {
		name                string
		fromBalance         int64
		toBalance           int64
		amountCents         int64
		expectedFromBalance int64
		expectedToBalance   int64
		expectedErr         error
	}{
		{
			name:                "debits the source and credits the destination",
			fromBalance:         1000,
			toBalance:           500,
			amountCents:         100,
			expectedFromBalance: 900,
			expectedToBalance:   600,
		},
		{
			name:                "empties the source and brings the destination to its maximum balance",
			fromBalance:         100,
			toBalance:           math.MaxInt64 - 100,
			amountCents:         100,
			expectedFromBalance: 0,
			expectedToBalance:   math.MaxInt64,
		},
		{
			name:                "rejects insufficient funds",
			fromBalance:         1000,
			toBalance:           500,
			amountCents:         1001,
			expectedFromBalance: 1000,
			expectedToBalance:   500,
			expectedErr:         ErrInsufficientFunds,
		},
		{
			name:                "rejects destination overflow",
			fromBalance:         1000,
			toBalance:           math.MaxInt64 - 99,
			amountCents:         100,
			expectedFromBalance: 1000,
			expectedToBalance:   math.MaxInt64 - 99,
			expectedErr:         ErrBalanceOverflow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := Account{ID: 1, BalanceCents: tt.fromBalance}
			to := Account{ID: 2, BalanceCents: tt.toBalance}
			if err := applyTransfer(&from, &to, tt.amountCents); !errors.Is(err, tt.expectedErr) {
				t.Errorf("applyTransfer error = %v, want %v", err, tt.expectedErr)
			}
			if from.BalanceCents != tt.expectedFromBalance || to.BalanceCents != tt.expectedToBalance {
				t.Errorf("balances = %d and %d, want %d and %d", from.BalanceCents, to.BalanceCents, tt.expectedFromBalance, tt.expectedToBalance)
			}
		})
	}
}
