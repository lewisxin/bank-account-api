package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
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
			handler := &accountHandler{accounts: make(map[int64]account)}
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.createAccount(response, request)

			if response.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusCreated, response.Body)
			}
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
			handler := &accountHandler{accounts: make(map[int64]account)}
			request := httptest.NewRequest(http.MethodPost, "/account", strings.NewReader(tt.requestBody))
			response := httptest.NewRecorder()
			handler.createAccount(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusBadRequest, response.Body)
			}

			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
		})
	}
}

func TestCreateAccountIncrementsID(t *testing.T) {
	handler := &accountHandler{accounts: make(map[int64]account)}

	for wantID := 1; wantID <= 2; wantID++ {
		request := httptest.NewRequest(http.MethodPost, "/account", nil)
		response := httptest.NewRecorder()
		handler.createAccount(response, request)

		var got struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
			t.Fatalf("request %d: decode account response: %v", wantID, err)
		}
		if got.ID != wantID {
			t.Errorf("request %d: id = %d, want %d", wantID, got.ID, wantID)
		}
	}
}

func TestDepositSuccess(t *testing.T) {
	handler := &accountHandler{accounts: map[int64]account{1: {ID: 1, BalanceCents: 9500}}}
	request := httptest.NewRequest(http.MethodPost, "/account/1/deposit", strings.NewReader(`{"amount_cents":1500}`))
	request.SetPathValue("id", "1")
	response := httptest.NewRecorder()
	handler.deposit(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusOK, response.Body)
	}
	if body := strings.TrimSpace(response.Body.String()); body != `{"id":1,"balance_cents":11000}` {
		t.Errorf("unexpected response: %s", body)
	}
	if balance := handler.accounts[1].BalanceCents; balance != 11000 {
		t.Errorf("stored balance = %d, want 11000", balance)
	}
}

func TestDepositErrors(t *testing.T) {
	const invalidAmount = `{"error":"expected a JSON object with a positive integer amount_cents"}`

	tests := []struct {
		name           string
		accountID      string
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid account ID",
			accountID:      "abc",
			requestBody:    `{"amount_cents":100}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"account ID must be a positive integer"}`,
		},
		{
			name:           "zero amount",
			accountID:      "1",
			requestBody:    `{"amount_cents":0}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   invalidAmount,
		},
		{
			name:           "negative amount",
			accountID:      "1",
			requestBody:    `{"amount_cents":-100}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   invalidAmount,
		},
		{
			name:           "fractional cents",
			accountID:      "1",
			requestBody:    `{"amount_cents":0.5}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   invalidAmount,
		},
		{
			name:           "malformed JSON",
			accountID:      "1",
			requestBody:    `{"amount_cents":`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   invalidAmount,
		},
		{
			name:           "account does not exist",
			accountID:      "2",
			requestBody:    `{"amount_cents":100}`,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"account not found"}`,
		},
		{
			name:           "balance overflow",
			accountID:      "1",
			requestBody:    `{"amount_cents":9223372036854775807}`,
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"deposit would exceed the maximum balance"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialBalance := int64(2500)
			handler := &accountHandler{accounts: map[int64]account{1: {ID: 1, BalanceCents: initialBalance}}}
			request := httptest.NewRequest(http.MethodPost, "/account/"+tt.accountID+"/deposit", strings.NewReader(tt.requestBody))
			request.SetPathValue("id", tt.accountID)
			response := httptest.NewRecorder()
			handler.deposit(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, tt.expectedStatus, response.Body)
			}
			if body := strings.TrimSpace(response.Body.String()); body != tt.expectedBody {
				t.Errorf("body = %s, want %s", body, tt.expectedBody)
			}
			if balance := handler.accounts[1].BalanceCents; balance != initialBalance {
				t.Errorf("stored balance = %d, want %d", balance, initialBalance)
			}
		})
	}
}
