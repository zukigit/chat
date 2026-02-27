package main

import (
	"net/http"

	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/handlers"
	"github.com/zukigit/chat/backend/internal/lib"
)

func main() {
	backendAddr := lib.Getenv("BACKEND_ADDRESS", "localhost:1234")

	authClient, err := clients.NewAuthClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend: %v", err)
	}
	defer authClient.Close()

	authHandler := handlers.NewAuthHandler(authClient)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /signup", authHandler.Signup)

	listenAddr := lib.Getenv("GATEWAY_LISTEN_ADDRESS", ":8080")
	lib.InfoLog.Printf("Gateway listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		lib.ErrorLog.Fatalf("Gateway failed: %v", err)
	}
}
