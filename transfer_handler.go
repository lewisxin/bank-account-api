package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type transferRequest struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	AmountCents   int64 `json:"amount_cents"`
}

type transferResponse struct {
	FromAccount account `json:"from_account"`
	ToAccount   account `json:"to_account"`
}

type transferHandler struct {
	store accountStorage
}

func (h *transferHandler) transfer(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input transferRequest
	var extra json.RawMessage

	status := http.StatusBadRequest
	var response any
	if decoder.Decode(&input) != nil || decoder.Decode(&extra) != io.EOF {
		response = map[string]string{"error": "expected a JSON object with integer from_account_id, to_account_id and amount_cents"}
	} else {
		from, to, err := h.store.transfer(input.FromAccountID, input.ToAccountID, input.AmountCents)
		switch err {
		case nil:
			status = http.StatusOK
			response = transferResponse{FromAccount: from, ToAccount: to}
		case errInvalidTransferAccounts, errInvalidTransferAmount:
			response = map[string]string{"error": err.Error()}
		case errAccountNotFound:
			status = http.StatusNotFound
			response = map[string]string{"error": err.Error()}
		case errInsufficientFunds:
			status = http.StatusConflict
			response = map[string]string{"error": err.Error()}
		case errBalanceOverflow:
			status = http.StatusConflict
			response = map[string]string{"error": "transfer would exceed the destination's maximum balance"}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write transfer response: %v", err)
	}
}
