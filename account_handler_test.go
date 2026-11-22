package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type stubAccountStorage struct {
	addAccountFunc func(initialBalanceCents int64) account
	depositFunc    func(id, amountCents int64) (account, error)
	transferFunc   func(fromID, toID, amountCents int64) (account, account, error)
}

func (s stubAccountStorage) addAccount(initialBalanceCents int64) account {
	return s.addAccountFunc(initialBalanceCents)
}

func (s stubAccountStorage) deposit(id, amountCents int64) (account, error) {
	return s.depositFunc(id, amountCents)
}

func (s stubAccountStorage) transfer(fromID, toID, amountCents int64) (account, account, error) {
	return s.transferFunc(fromID, toID, amountCents)
}

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
			store := stubAccountStorage{
				addAccountFunc: func(initialBalanceCents int64) account {
					return account{ID: 1, BalanceCents: initialBalanceCents}
				},
			}
			handler := &accountHandler{store: store}
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.createAccount(response, request)

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
			handler := &accountHandler{store: stubAccountStorage{}}
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.createAccount(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}

func TestDepositSuccess(t *testing.T) {
	store := stubAccountStorage{
		depositFunc: func(id, amountCents int64) (account, error) {
			return account{ID: id, BalanceCents: 9500 + amountCents}, nil
		},
	}
	handler := &accountHandler{store: store}
	request := httptest.NewRequest(http.MethodPost, "/account/1/deposit", strings.NewReader(`{"amount_cents":1500}`))
	request.SetPathValue("id", "1")
	response := httptest.NewRecorder()
	handler.deposit(response, request)

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
			storeErr:     errAccountNotFound,
			expectedBody: `{"error":"account not found"}`,
		},
		{
			name:         "balance overflow",
			accountID:    "1",
			requestBody:  `{"amount_cents":100}`,
			storeErr:     errBalanceOverflow,
			expectedBody: `{"error":"deposit would exceed the maximum balance"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := stubAccountStorage{
				depositFunc: func(_, _ int64) (account, error) {
					return account{}, tt.storeErr
				},
			}
			handler := &accountHandler{store: store}
			request := httptest.NewRequest(http.MethodPost, "/account/"+tt.accountID+"/deposit", strings.NewReader(tt.requestBody))
			request.SetPathValue("id", tt.accountID)
			response := httptest.NewRecorder()
			handler.deposit(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}
