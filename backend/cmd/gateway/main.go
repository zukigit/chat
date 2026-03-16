package main

import (
	"net/http"

	gorhandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/handlers"
	"github.com/zukigit/chat/backend/internal/lib"
)

func main() {
	backendAddr := lib.Getenv("BACKEND_LISTEN_ADDRESS", "localhost:1234")
	natsURL := lib.Getenv("NATS_URL", nats.DefaultURL)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	authClient, err := clients.NewAuthClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend: %v", err)
	}
	defer authClient.Close()

	friendshipClient, err := clients.NewFriendshipClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend (friendship): %v", err)
	}
	defer friendshipClient.Close()

	sessionClient, err := clients.NewSessionClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend (session): %v", err)
	}
	defer sessionClient.Close()

	authHandler := handlers.NewAuthHandler(authClient)
	friendshipHandler := handlers.NewFriendshipHandler(friendshipClient)
	sessionHandler := handlers.NewSessionHandler(sessionClient, nc)

	r := mux.NewRouter()
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/signup", authHandler.Signup).Methods(http.MethodPost)

	r.HandleFunc("/friends/request", friendshipHandler.SendFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/friends/accept", friendshipHandler.AcceptFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/friends/reject", friendshipHandler.RejectFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/session/notification", sessionHandler.NotificationSession).Methods(http.MethodPost)
	r.HandleFunc("/session/chat", sessionHandler.ChatSession).Methods(http.MethodPost)

	cors := gorhandlers.CORS(
		gorhandlers.AllowedOrigins([]string{lib.Getenv("FRONTEND_URL", "http://localhost:5173")}),
		gorhandlers.AllowedMethods([]string{"GET", "POST"}),
		gorhandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	listenAddr := lib.Getenv("GATEWAY_LISTEN_ADDRESS", ":8080")
	lib.InfoLog.Printf("Gateway listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, cors(r)); err != nil {
		lib.ErrorLog.Fatalf("Gateway failed: %v", err)
	}
}
