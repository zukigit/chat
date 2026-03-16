package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/session"
)

// sessionClientInterface is the minimal interface SessionHandler needs.
type sessionClientInterface interface {
	AddSession(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error)
	SetSessionStatus(ctx context.Context, token, sessionID, status string) error
}

// SessionHandler holds dependencies for session-related HTTP handlers.
type SessionHandler struct {
	client sessionClientInterface
	stream jetstream.Stream
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client sessionClientInterface, stream jetstream.Stream) *SessionHandler {
	return &SessionHandler{client: client, stream: stream}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *SessionHandler) NotificationSession(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "Missing token",
		})
		return
	}

	addSessionResp, err := s.client.AddSession(r.Context(), token, string(db.SessionTypeNotification))
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to add session, sessionId: %s, err: %v", addSessionResp.SessionId, err),
		})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to upgrade to websocket: %v", err),
		})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	consumer, err := s.stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{addSessionResp.ListenPath},
	})
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to create consumer: %v", err),
		})
		return
	}

	// Handle disconnect to update status
	defer func() {
		_ = s.client.SetSessionStatus(context.Background(), token, addSessionResp.SessionId, string(db.SessionStatusTerminate))
	}()

	// Set session status to active
	err = s.client.SetSessionStatus(context.Background(), token, addSessionResp.SessionId, string(db.SessionStatusActive))
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to set session status active: %v", err),
		})
		return
	}

	// Read channel from WS client so we detect close/disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				lib.InfoLog.Printf("closing session: sessionId: %s, err: %v", addSessionResp.SessionId, err)

				// cancel context to stop consumer and exit the read loop
				cancel()
				return
			}
		}
	}()

	consumer.Consume(func(msg jetstream.Msg) {
		if err := conn.WriteMessage(websocket.TextMessage, msg.Data()); err != nil {
			lib.ErrorLog.Printf("Error writing to websocket: sessionId: %s, err: %v", addSessionResp.SessionId, err)

		}
		msg.Ack()
	})

	// Block until the client disconnects (the read goroutine calls cancel() on
	// any WS error, which unblocks this and lets the deferred cleanup run).
	<-ctx.Done()
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {

}
