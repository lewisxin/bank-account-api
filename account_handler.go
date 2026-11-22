package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

const maxInitialBalanceCents = 10_000

type account struct {
	ID           int64 `json:"id"`
	BalanceCents int64 `json:"balance_cents"`
}

type createAccountRequest struct {
	InitialBalanceCents int64 `json:"initial_balance_cents"`
}

type accountHandler struct {
	mu       sync.Mutex
	nextID   int64
	accounts map[int64]account
}

func (h *accountHandler) createAccount(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	input := &createAccountRequest{}
	err := decoder.Decode(&input)
	balanceTypeError, _ := errors.AsType[*json.UnmarshalTypeError](err)
	invalidBalanceValue := balanceTypeError != nil && balanceTypeError.Field == "initial_balance_cents"

	validBody := err == io.EOF
	if (err == nil || invalidBalanceValue) && input != nil {
		var extra json.RawMessage
		validBody = decoder.Decode(&extra) == io.EOF
	}

	status := http.StatusBadRequest
	var response any
	switch {
	case !validBody:
		response = map[string]string{"error": "expected a JSON object with an optional integer initial_balance_cents"}
	case invalidBalanceValue || input.InitialBalanceCents < 0 || input.InitialBalanceCents > maxInitialBalanceCents:
		response = map[string]string{"error": fmt.Sprintf("initial_balance_cents must be an integer between 0 and %d", maxInitialBalanceCents)}
	default:
		status = http.StatusCreated
		response = h.addAccount(input.InitialBalanceCents)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write account response: %v", err)
	}
}

func (h *accountHandler) addAccount(initialBalanceCents int64) account {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	created := account{ID: h.nextID, BalanceCents: initialBalanceCents}
	h.accounts[created.ID] = created
	return created
}
