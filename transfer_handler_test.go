package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransferSuccess(t *testing.T) {
	store := stubAccountStorage{
		transferFunc: func(fromID, toID, amountCents int64) (account, account, error) {
			from := account{ID: fromID, BalanceCents: 2500 - amountCents}
			to := account{ID: toID, BalanceCents: 1000 + amountCents}
			return from, to, nil
		},
	}
	handler := &transferHandler{store: store}
	request := httptest.NewRequest(http.MethodPost, "/transfer", strings.NewReader(`{"from_account_id":1,"to_account_id":2,"amount_cents":500}`))
	response := httptest.NewRecorder()
	handler.transfer(response, request)

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
			storeErr:     errInvalidTransferAccounts,
			expectedBody: `{"error":"from_account_id and to_account_id must be distinct positive integers"}`,
		},
		{
			name:         "non-positive amount",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":0}`,
			storeErr:     errInvalidTransferAmount,
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
			storeErr:     errAccountNotFound,
			expectedBody: `{"error":"account not found"}`,
		},
		{
			name:         "insufficient funds",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":1001}`,
			storeErr:     errInsufficientFunds,
			expectedBody: `{"error":"insufficient funds"}`,
		},
		{
			name:         "destination balance overflow",
			requestBody:  `{"from_account_id":1,"to_account_id":2,"amount_cents":100}`,
			storeErr:     errBalanceOverflow,
			expectedBody: `{"error":"transfer would exceed the destination's maximum balance"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := stubAccountStorage{
				transferFunc: func(_, _, _ int64) (account, account, error) {
					return account{}, account{}, tt.storeErr
				},
			}
			handler := &transferHandler{store: store}
			request := httptest.NewRequest(http.MethodPost, "/transfer", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.transfer(response, request)

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}
