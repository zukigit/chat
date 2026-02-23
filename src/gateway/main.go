package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zukigit/chat/src/gateway/handlers"
)

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/login", handlers.LoginHandler).Methods(http.MethodPost)
	return r
}

func main() {
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      newRouter(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Gateway listening on :8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
