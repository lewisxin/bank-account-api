package store

import "context"

type mockTestReporter interface {
	Cleanup(func())
	Error(args ...any)
	Fatalf(format string, args ...any)
	Helper()
}

// MockAccountStore expects one call to each configured function.
type MockAccountStore struct {
	t              mockTestReporter
	AddAccountFunc func(initialBalanceCents int64) (Account, error)
	DepositFunc    func(id, amountCents int64) (Account, error)
	TransferFunc   func(fromID, toID, amountCents int64) (Account, Account, error)
}

// NewMockAccountStore registers a cleanup that checks for missing calls.
func NewMockAccountStore(t mockTestReporter) *MockAccountStore {
	mock := &MockAccountStore{t: t}
	t.Cleanup(func() {
		if mock.AddAccountFunc != nil {
			t.Error("expected one call to AddAccount")
		}
		if mock.DepositFunc != nil {
			t.Error("expected one call to Deposit")
		}
		if mock.TransferFunc != nil {
			t.Error("expected one call to Transfer")
		}
	})
	return mock
}

func (m *MockAccountStore) AddAccount(_ context.Context, initialBalanceCents int64) (Account, error) {
	m.t.Helper()
	call := m.AddAccountFunc
	if call == nil {
		m.t.Fatalf("unexpected AddAccount(%d)", initialBalanceCents)
	}
	m.AddAccountFunc = nil
	return call(initialBalanceCents)
}

func (m *MockAccountStore) Deposit(_ context.Context, id, amountCents int64) (Account, error) {
	m.t.Helper()
	call := m.DepositFunc
	if call == nil {
		m.t.Fatalf("unexpected Deposit(%d, %d)", id, amountCents)
	}
	m.DepositFunc = nil
	return call(id, amountCents)
}

func (m *MockAccountStore) Transfer(_ context.Context, fromID, toID, amountCents int64) (Account, Account, error) {
	m.t.Helper()
	call := m.TransferFunc
	if call == nil {
		m.t.Fatalf("unexpected Transfer(%d, %d, %d)", fromID, toID, amountCents)
	}
	m.TransferFunc = nil
	return call(fromID, toID, amountCents)
}
