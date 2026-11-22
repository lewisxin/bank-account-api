package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/lewisxin/bank-account-api/internal/store"
)

const maxInitialBalanceCents = 10_000

type createAccountRequest struct {
	InitialBalanceCents int64 `json:"initial_balance_cents"`
}

type depositRequest struct {
	AmountCents int64 `json:"amount_cents"`
}

type transferRequest struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	AmountCents   int64 `json:"amount_cents"`
}

type transferResponse struct {
	FromAccount store.Account `json:"from_account"`
	ToAccount   store.Account `json:"to_account"`
}

type AccountStore interface {
	AddAccount(ctx context.Context, initialBalanceCents int64) (store.Account, error)
	Deposit(ctx context.Context, id, amountCents int64) (store.Account, error)
	Transfer(ctx context.Context, fromID, toID, amountCents int64) (store.Account, store.Account, error)
}

type Accounts struct {
	store AccountStore
}

func NewAccounts(store AccountStore) *Accounts {
	return &Accounts{store: store}
}

func (h *Accounts) Create(w http.ResponseWriter, r *http.Request) {
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
		created, err := h.store.AddAccount(r.Context(), input.InitialBalanceCents)
		if err != nil {
			log.Printf("create account: %v", err)
			status = http.StatusInternalServerError
			response = map[string]string{"error": "internal server error"}
		} else {
			status = http.StatusCreated
			response = created
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write account response: %v", err)
	}
}

func (h *Accounts) Deposit(w http.ResponseWriter, r *http.Request) {
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
		updated, err := h.store.Deposit(r.Context(), id, input.AmountCents)
		switch err {
		case store.ErrAccountNotFound:
			status = http.StatusNotFound
			response = map[string]string{"error": err.Error()}
		case store.ErrBalanceOverflow:
			status = http.StatusConflict
			response = map[string]string{"error": "deposit would exceed the maximum balance"}
		case nil:
			status = http.StatusOK
			response = updated
		default:
			log.Printf("deposit: %v", err)
			status = http.StatusInternalServerError
			response = map[string]string{"error": "internal server error"}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write deposit response: %v", err)
	}
}

func (h *Accounts) Transfer(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input transferRequest
	var extra json.RawMessage

	status := http.StatusBadRequest
	var response any
	if decoder.Decode(&input) != nil || decoder.Decode(&extra) != io.EOF {
		response = map[string]string{"error": "expected a JSON object with integer from_account_id, to_account_id and amount_cents"}
	} else {
		from, to, err := h.store.Transfer(r.Context(), input.FromAccountID, input.ToAccountID, input.AmountCents)
		switch err {
		case nil:
			status = http.StatusOK
			response = transferResponse{FromAccount: from, ToAccount: to}
		case store.ErrInvalidTransferAccounts, store.ErrInvalidTransferAmount:
			response = map[string]string{"error": err.Error()}
		case store.ErrAccountNotFound:
			status = http.StatusNotFound
			response = map[string]string{"error": err.Error()}
		case store.ErrInsufficientFunds:
			status = http.StatusConflict
			response = map[string]string{"error": err.Error()}
		case store.ErrBalanceOverflow:
			status = http.StatusConflict
			response = map[string]string{"error": "transfer would exceed the destination's maximum balance"}
		default:
			log.Printf("transfer: %v", err)
			status = http.StatusInternalServerError
			response = map[string]string{"error": "internal server error"}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write transfer response: %v", err)
	}
}
