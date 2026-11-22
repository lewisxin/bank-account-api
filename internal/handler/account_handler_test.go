package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/lewisxin/bank-account-api/internal/store"
)

func TestCreateAccountSuccess(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          string
		expectedBalanceCents int64
	}{
		{
			name:                 "with initial balance in cents",
			requestBody:          `{"initial_balance_cents":2500}`,
			expectedBalanceCents: 2500,
		},
		{
			name:                 "with explicit zero",
			requestBody:          `{"initial_balance_cents":0}`,
			expectedBalanceCents: 0,
		},
		{
			name:                 "at maximum balance",
			requestBody:          `{"initial_balance_cents":10000}`,
			expectedBalanceCents: 10000,
		},
		{
			name:                 "without initial balance",
			requestBody:          `{}`,
			expectedBalanceCents: 0,
		},
		{
			name:                 "with empty body",
			requestBody:          "",
			expectedBalanceCents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := store.NewMockAccountStore(t)
			mock.AddAccountFunc = func(initialBalanceCents int64) store.Account {
				if initialBalanceCents != tt.expectedBalanceCents {
					t.Errorf("AddAccount(%d), want AddAccount(%d)", initialBalanceCents, tt.expectedBalanceCents)
				}
				return store.Account{ID: 1, BalanceCents: tt.expectedBalanceCents}
			}
			handler := NewAccounts(mock)
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.Create(response, request)

			body := response.Body.String()
			want := `"balance_cents":` + strconv.FormatInt(tt.expectedBalanceCents, 10)
			if !strings.Contains(body, want+",") && !strings.Contains(body, want+"}") {
				t.Errorf("body = %s, want field %s", body, want)
			}
		})
	}
}

func TestCreateAccountBadRequests(t *testing.T) {
	const invalidBalance = `{"error":"initial_balance_cents must be an integer between 0 and 10000"}`
	const invalidRequest = `{"error":"expected a JSON object with an optional integer initial_balance_cents"}`

	tests := []struct {
		name         string
		requestBody  string
		expectedBody string
	}{
		{
			name:         "negative balance",
			requestBody:  `{"initial_balance_cents":-1}`,
			expectedBody: invalidBalance,
		},
		{
			name:         "above maximum balance",
			requestBody:  `{"initial_balance_cents":10001}`,
			expectedBody: invalidBalance,
		},
		{
			name:         "fractional cents",
			requestBody:  `{"initial_balance_cents":0.5}`,
			expectedBody: invalidBalance,
		},
		{
			name:         "malformed JSON",
			requestBody:  `{"initial_balance_cents":`,
			expectedBody: invalidRequest,
		},
		{
			name:         "null request body",
			requestBody:  `null`,
			expectedBody: invalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAccounts(store.NewMockAccountStore(t))
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.Create(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}

func TestDepositSuccess(t *testing.T) {
	mock := store.NewMockAccountStore(t)
	mock.DepositFunc = func(id, amountCents int64) (store.Account, error) {
		if id != 1 || amountCents != 1500 {
			t.Errorf("Deposit(%d, %d), want Deposit(1, 1500)", id, amountCents)
		}
		return store.Account{ID: 1, BalanceCents: 11000}, nil
	}
	handler := NewAccounts(mock)
	request := httptest.NewRequest(http.MethodPost, "/account/1/deposit", strings.NewReader(`{"amount_cents":1500}`))
	request.SetPathValue("id", "1")
	response := httptest.NewRecorder()
	handler.Deposit(response, request)

	if body := strings.TrimSpace(response.Body.String()); body != `{"id":1,"balance_cents":11000}` {
		t.Errorf("unexpected response: %s", body)
	}
}

func TestDepositErrors(t *testing.T) {
	const invalidAmount = `{"error":"expected a JSON object with a positive integer amount_cents"}`

	tests := []struct {
		name         string
		accountID    string
		requestBody  string
		storeErr     error
		expectedBody string
	}{
		{
			name:         "invalid account ID",
			accountID:    "abc",
			requestBody:  `{"amount_cents":100}`,
			expectedBody: `{"error":"account ID must be a positive integer"}`,
		},
		{
			name:         "zero amount",
			accountID:    "1",
			requestBody:  `{"amount_cents":0}`,
			expectedBody: invalidAmount,
		},
		{
			name:         "negative amount",
			accountID:    "1",
			requestBody:  `{"amount_cents":-100}`,
			expectedBody: invalidAmount,
		},
		{
			name:         "fractional cents",
			accountID:    "1",
			requestBody:  `{"amount_cents":0.5}`,
			expectedBody: invalidAmount,
		},
		{
			name:         "malformed JSON",
			accountID:    "1",
			requestBody:  `{"amount_cents":`,
			expectedBody: invalidAmount,
		},
		{
			name:         "account does not exist",
			accountID:    "2",
			requestBody:  `{"amount_cents":100}`,
			storeErr:     store.ErrAccountNotFound,
			expectedBody: `{"error":"account not found"}`,
		},
		{
			name:         "balance overflow",
			accountID:    "1",
			requestBody:  `{"amount_cents":100}`,
			storeErr:     store.ErrBalanceOverflow,
			expectedBody: `{"error":"deposit would exceed the maximum balance"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := store.NewMockAccountStore(t)
			if tt.storeErr != nil {
				mock.DepositFunc = func(_, _ int64) (store.Account, error) {
					return store.Account{}, tt.storeErr
				}
			}
			handler := NewAccounts(mock)
			request := httptest.NewRequest(http.MethodPost, "/account/"+tt.accountID+"/deposit", strings.NewReader(tt.requestBody))
			request.SetPathValue("id", tt.accountID)
			response := httptest.NewRecorder()
			handler.Deposit(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}

func TestTransferSuccess(t *testing.T) {
	mock := store.NewMockAccountStore(t)
	mock.TransferFunc = func(fromID, toID, amountCents int64) (store.Account, store.Account, error) {
		if fromID != 1 || toID != 2 || amountCents != 500 {
			t.Errorf("Transfer(%d, %d, %d), want Transfer(1, 2, 500)", fromID, toID, amountCents)
		}
		from := store.Account{ID: 1, BalanceCents: 2000}
		to := store.Account{ID: 2, BalanceCents: 1500}
		return from, to, nil
	}
	handler := NewAccounts(mock)
	request := httptest.NewRequest(http.MethodPost, "/account/transfer", strings.NewReader(`{"from_account_id":1,"to_account_id":2,"amount_cents":500}`))
	response := httptest.NewRecorder()
	handler.Transfer(response, request)

	const expectedBody = `{"from_account":{"id":1,"balance_cents":2000},"to_account":{"id":2,"balance_cents":1500}}`
	if body := strings.TrimSpace(response.Body.String()); body != expectedBody {
		t.Errorf("body = %s, want %s", body, expectedBody)
	}
}

func TestTransferErrors(t *testing.T) {
	const invalidRequest = `{"error":"expected a JSON object with integer from_account_id, to_account_id and amount_cents"}`
	tests := []struct {
		name         string
		requestBody  string
		storeErr     error
		expectedBody string
	}{
		{
			name:         "same account",
			requestBody:  `{"from_account_id":1,"to_account_id":1,"amount_cents":100}`,
			storeErr:     store.ErrInvalidTransferAccounts,
			expectedBody: `{"error":"from_account_id and to_account_id must be distinct positive integers"}`,
		},
		{
			name:         "non-positive amount",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":0}`,
			storeErr:     store.ErrInvalidTransferAmount,
			expectedBody: `{"error":"amount_cents must be a positive integer"}`,
		},
		{
			name:         "fractional cents",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":0.5}`,
			expectedBody: invalidRequest,
		},
		{
			name:         "missing destination account",
			requestBody:  `{"from_account_id":1,"to_account_id":3,"amount_cents":100}`,
			storeErr:     store.ErrAccountNotFound,
			expectedBody: `{"error":"account not found"}`,
		},
		{
			name:         "insufficient funds",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":1001}`,
			storeErr:     store.ErrInsufficientFunds,
			expectedBody: `{"error":"insufficient funds"}`,
		},
		{
			name:         "destination balance overflow",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":100}`,
			storeErr:     store.ErrBalanceOverflow,
			expectedBody: `{"error":"transfer would exceed the destination's maximum balance"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := store.NewMockAccountStore(t)
			if tt.storeErr != nil {
				mock.TransferFunc = func(_, _, _ int64) (store.Account, store.Account, error) {
					return store.Account{}, store.Account{}, tt.storeErr
				}
			}
			handler := NewAccounts(mock)
			request := httptest.NewRequest(http.MethodPost, "/account/transfer", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.Transfer(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}
