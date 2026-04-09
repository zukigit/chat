package main

import (
	"context"
	"net/http"

	gorhandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/handlers"
	"github.com/zukigit/chat/backend/internal/lib"
)

func main() {
	// env variables
	backendAddr := lib.Getenv("BACKEND_LISTEN_ADDRESS", "localhost:1234")
	natsURL := lib.Getenv("NATS_URL", nats.DefaultURL)

	// nats connection
	nc, err := nats.Connect(natsURL)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// streams preparation
	js, err := jetstream.New(nc)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to create JetStream: %v", err)
	}

	sessionsStream, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     "SESSIONS",
		Subjects: []string{lib.NotiSubjectPrefix + ">", lib.ChatSubjectPrefix + ">"},
	})
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to create sessions stream: %v", err)
	}

	// clients preparation
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

	notiClient, err := clients.NewNotificationClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend (notification): %v", err)
	}
	defer notiClient.Close()

	chatClient, err := clients.NewChatClient(backendAddr)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to backend (chat): %v", err)
	}
	defer chatClient.Close()

	// handlers preparation
	authHandler := handlers.NewAuthHandler(authClient)
	friendshipHandler := handlers.NewFriendshipHandler(friendshipClient)
	sessionHandler := handlers.NewSessionHandler(sessionClient, chatClient, sessionsStream)
	notificationHandler := handlers.NewNotificationHandler(notiClient)
	chatHandler := handlers.NewChatHandler(chatClient)

	r := mux.NewRouter()
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/signup", authHandler.Signup).Methods(http.MethodPost)

	r.HandleFunc("/friends/request", friendshipHandler.SendFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/friends/accept", friendshipHandler.AcceptFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/friends/reject", friendshipHandler.RejectFriendRequest).Methods(http.MethodPost)
	r.HandleFunc("/notifications/read", notificationHandler.MarkNotificationRead).Methods(http.MethodPost)
	r.HandleFunc("/sessions/notification", sessionHandler.NotificationSession).Methods(http.MethodGet)
	r.HandleFunc("/sessions/chat", sessionHandler.ChatSession).Methods(http.MethodPost)
	r.HandleFunc("/conversations", chatHandler.CreateConversation).Methods(http.MethodPost)
	r.HandleFunc("/conversations/messages", chatHandler.GetMessages).Methods(http.MethodGet)

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
