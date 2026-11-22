package main

import (
	"log"
	"net/http"

	"github.com/lewisxin/bank-account-api/internal/handler"
	"github.com/lewisxin/bank-account-api/internal/store"
)

func main() {
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", newMux()))
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	accounts := handler.NewAccounts(store.NewAccounts())
	mux.HandleFunc("POST /account", accounts.Create)
	mux.HandleFunc("POST /account/{id}/deposit", accounts.Deposit)
	mux.HandleFunc("POST /account/transfer", accounts.Transfer)
	return mux
}
