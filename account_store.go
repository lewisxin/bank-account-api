package main

import (
	"errors"
	"math"
	"sync"
)

type account struct {
	ID           int64 `json:"id"`
	BalanceCents int64 `json:"balance_cents"`
}

type accountStorage interface {
	addAccount(initialBalanceCents int64) account
	deposit(id, amountCents int64) (account, error)
	transfer(fromID, toID, amountCents int64) (account, account, error)
}

type accountStore struct {
	mu       sync.Mutex
	nextID   int64
	accounts map[int64]account
}

var (
	errAccountNotFound         = errors.New("account not found")
	errBalanceOverflow         = errors.New("balance would exceed the maximum")
	errInsufficientFunds       = errors.New("insufficient funds")
	errInvalidTransferAccounts = errors.New("from_account_id and to_account_id must be distinct positive integers")
	errInvalidTransferAmount   = errors.New("amount_cents must be a positive integer")
)

func newAccountStore() *accountStore {
	return &accountStore{accounts: make(map[int64]account)}
}

func (s *accountStore) addAccount(initialBalanceCents int64) account {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	created := account{ID: s.nextID, BalanceCents: initialBalanceCents}
	s.accounts[created.ID] = created
	return created
}

func (s *accountStore) deposit(id, amountCents int64) (account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.accounts[id]
	if !exists {
		return account{}, errAccountNotFound
	}
	if amountCents > math.MaxInt64-current.BalanceCents {
		return account{}, errBalanceOverflow
	}
	current.BalanceCents += amountCents
	s.accounts[id] = current
	return current, nil
}

func (s *accountStore) transfer(fromID, toID, amountCents int64) (account, account, error) {
	if fromID <= 0 || toID <= 0 || fromID == toID {
		return account{}, account{}, errInvalidTransferAccounts
	}
	if amountCents <= 0 {
		return account{}, account{}, errInvalidTransferAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	from, fromExists := s.accounts[fromID]
	to, toExists := s.accounts[toID]
	if !fromExists || !toExists {
		return account{}, account{}, errAccountNotFound
	}
	if from.BalanceCents < amountCents {
		return account{}, account{}, errInsufficientFunds
	}
	if amountCents > math.MaxInt64-to.BalanceCents {
		return account{}, account{}, errBalanceOverflow
	}

	from.BalanceCents -= amountCents
	to.BalanceCents += amountCents
	s.accounts[fromID] = from
	s.accounts[toID] = to
	return from, to, nil
}
