package main

import (
	"database/sql"
	"fmt"
	"net"

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
	dbName := lib.Getenv("DB_NAME", "chat")
	dbUser := lib.Getenv("DB_USER", "chat")
	dbHost := lib.Getenv("DB_HOST", "localhost")
	dbPasswd := lib.Getenv("DB_PASSWORD", "")

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPasswd, dbHost, dbName)

	// TODO: delete this
	fmt.Printf("DSN: %s\n", dsn)

	// database connection
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to connect to database: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to ping database: %v", err)
	}

	// get JetStream
	nats_url := lib.Getenv("NATS_URL", nats.DefaultURL)
	lib.InfoLog.Printf("Connecting to NATS at %s", nats_url)
	js, nc, err := lib.GetJetStream(nats_url)
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to set up JetStream: %v", err)
	}
	defer nc.Close()

	// gRPC server setup
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

	// services registration
	auth.RegisterAuthServer(srv, services.NewAuthServer(sqlDB))
	notifServer := services.NewNotificationServer(sqlDB, js)
	notification.RegisterNotificationServer(srv, notifServer)
	friendship.RegisterFriendshipServer(srv, services.NewFriendshipServer(sqlDB, notifServer))
	session.RegisterSessionServer(srv, services.NewSessionServer())
	chat.RegisterChatServer(srv, services.NewChatServer(sqlDB, notifServer))

	// start listening
	listener, err := net.Listen("tcp", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"))
	if err != nil {
		lib.ErrorLog.Fatalf("Failed to listen on %s: %v", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"), err)
	}
	lib.InfoLog.Printf("Backend listening on %s", lib.Getenv("BACKEND_LISTEN_ADDRESS", ":1234"))

	// serve
	if err := srv.Serve(listener); err != nil {
		lib.ErrorLog.Fatalf("Failed to serve: %v", err)
	}
}
