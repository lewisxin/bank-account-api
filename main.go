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
	accounts := &accountHandler{accounts: make(map[int64]account)}
	mux.HandleFunc("POST /account", accounts.createAccount)
	return mux
}
