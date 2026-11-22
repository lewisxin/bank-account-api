package main

import (
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", newMux()))
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	store := newAccountStore()
	accounts := &accountHandler{store: store}
	transfers := &transferHandler{store: store}
	mux.HandleFunc("POST /account", accounts.createAccount)
	mux.HandleFunc("POST /account/{id}/deposit", accounts.deposit)
	mux.HandleFunc("POST /transfer", transfers.transfer)
	return mux
}
