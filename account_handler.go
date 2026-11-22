package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

const maxInitialBalanceCents = 10_000

type createAccountRequest struct {
	InitialBalanceCents int64 `json:"initial_balance_cents"`
}

type depositRequest struct {
	AmountCents int64 `json:"amount_cents"`
}

type accountHandler struct {
	store accountStorage
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
		response = h.store.addAccount(input.InitialBalanceCents)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write account response: %v", err)
	}
}

func (h *accountHandler) deposit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input depositRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var extra json.RawMessage

	status := http.StatusBadRequest
	var response any
	switch {
	case err != nil || id <= 0:
		response = map[string]string{"error": "account ID must be a positive integer"}
	case decoder.Decode(&input) != nil || input.AmountCents <= 0 || decoder.Decode(&extra) != io.EOF:
		response = map[string]string{"error": "expected a JSON object with a positive integer amount_cents"}
	default:
		updated, err := h.store.deposit(id, input.AmountCents)
		switch err {
		case errAccountNotFound:
			status = http.StatusNotFound
			response = map[string]string{"error": err.Error()}
		case errBalanceOverflow:
			status = http.StatusConflict
			response = map[string]string{"error": "deposit would exceed the maximum balance"}
		case nil:
			status = http.StatusOK
			response = updated
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write deposit response: %v", err)
	}
}
