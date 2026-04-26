package main

import (
	"database/sql"
	"fmt"
	"net"
	"time"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/zukigit/chat/backend/internal/interceptors"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/internal/services"
	"github.com/zukigit/chat/backend/proto/auth"
	"github.com/zukigit/chat/backend/proto/chat"
	"github.com/zukigit/chat/backend/proto/friendship"
	"github.com/zukigit/chat/backend/proto/notification"
	"github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc"
)

func main() {
	dbHost := lib.Getenv("DB_HOST", "localhost")
	dbPasswd := lib.Getenv("DB_PASSWORD", "")
	dbSslMode := lib.Getenv("DB_SSLMODE", "disable")
	dsn := fmt.Sprintf("postgres://chat:%s@%s/chat?sslmode=%s", dbPasswd, dbHost, dbSslMode)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to database: %v", err)
	}

	natsURL := lib.Getenv("NATS_URL", nats.DefaultURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// streams preparation
	js, err := nc.JetStream()
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to create JetStream: %v", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "SESSIONS",
		Subjects: []string{lib.NotiSubjectPrefix + ">", lib.ChatSubjectPrefix + ">"},
		MaxAge:   24 * time.Hour,
		Storage:  nats.FileStorage,
	})
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to create sessions stream: %v", err)
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRecoveryInterceptor,
			interceptors.UnaryJWTInterceptor,
		),
		grpc.ChainStreamInterceptor(
			interceptors.StreamRecoveryInterceptor,
			interceptors.StreamJWTInterceptor,
		),
	)

	auth.RegisterAuthServer(srv, services.NewAuthServer(sqlDB))
	notifServer := services.NewNotificationServer(sqlDB, nc)
	notification.RegisterNotificationServer(srv, notifServer)
	friendship.RegisterFriendshipServer(srv, services.NewFriendshipServer(sqlDB, notifServer))
	session.RegisterSessionServer(srv, services.NewSessionServer(sqlDB))
	chat.RegisterChatServer(srv, services.NewChatServer(sqlDB, notifServer))

	lib.InfoLog.Printf("Backend listening on %s", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"))
	listener, err := net.Listen("tcp", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"))
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to listen on %s: %v", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"), err)
	}

	if err := srv.Serve(listener); err != nil {
		lib.ErrorLog.Fatalf("Failed to serve: %v", err)
	}
}
