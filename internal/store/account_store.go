package store

import (
	"errors"
	"math"
	"sync"
)

type Account struct {
	ID           int64 `json:"id"`
	BalanceCents int64 `json:"balance_cents"`
}

type Accounts struct {
	mu       sync.Mutex
	nextID   int64
	accounts map[int64]Account
}

var (
	ErrAccountNotFound         = errors.New("account not found")
	ErrBalanceOverflow         = errors.New("balance would exceed the maximum")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrInvalidTransferAccounts = errors.New("from_account_id and to_account_id must be distinct positive integers")
	ErrInvalidTransferAmount   = errors.New("amount_cents must be a positive integer")
)

func NewAccounts() *Accounts {
	return &Accounts{accounts: make(map[int64]Account)}
}

func (s *Accounts) AddAccount(initialBalanceCents int64) Account {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	created := Account{ID: s.nextID, BalanceCents: initialBalanceCents}
	s.accounts[created.ID] = created
	return created
}

func (s *Accounts) Deposit(id, amountCents int64) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.accounts[id]
	if !exists {
		return Account{}, ErrAccountNotFound
	}
	if amountCents > math.MaxInt64-current.BalanceCents {
		return Account{}, ErrBalanceOverflow
	}
	current.BalanceCents += amountCents
	s.accounts[id] = current
	return current, nil
}

func (s *Accounts) Transfer(fromID, toID, amountCents int64) (Account, Account, error) {
	if fromID <= 0 || toID <= 0 || fromID == toID {
		return Account{}, Account{}, ErrInvalidTransferAccounts
	}
	if amountCents <= 0 {
		return Account{}, Account{}, ErrInvalidTransferAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	from, fromExists := s.accounts[fromID]
	to, toExists := s.accounts[toID]
	if !fromExists || !toExists {
		return Account{}, Account{}, ErrAccountNotFound
	}
	if from.BalanceCents < amountCents {
		return Account{}, Account{}, ErrInsufficientFunds
	}
	if amountCents > math.MaxInt64-to.BalanceCents {
		return Account{}, Account{}, ErrBalanceOverflow
	}

	from.BalanceCents -= amountCents
	to.BalanceCents += amountCents
	s.accounts[fromID] = from
	s.accounts[toID] = to
	return from, to, nil
}
