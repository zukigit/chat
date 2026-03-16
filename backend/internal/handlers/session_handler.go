package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
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
	nc     *nats.Conn
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client sessionClientInterface, nc *nats.Conn) *SessionHandler {
	return &SessionHandler{client: client, nc: nc}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// allow all origins for now
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
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: "Failed to add session"})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		lib.ErrorLog.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	sub, err := s.nc.SubscribeSync(addSessionResp.ListenPath)
	if err != nil {
		lib.ErrorLog.Printf("Failed to subscribe to nats subject: %v", err)
		return
	}
	defer sub.Unsubscribe()

	// Handle disconnect to update status
	defer func() {
		_ = s.client.SetSessionStatus(context.Background(), token, addSessionResp.SessionId, string(db.SessionStatusTerminate))
	}()

	// Read channel from WS client so we detect close/disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				conn.Close()
				break
			}
		}
	}()

	// Loop to read from NATS and write to WS
	for {
		msg, err := sub.NextMsg(0) // wait indefinitely
		if err != nil {
			if err == nats.ErrConnectionClosed || err == nats.ErrBadSubscription {
				break
			}
			lib.ErrorLog.Printf("Error reading from nats: %v", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
			lib.ErrorLog.Printf("Error writing to websocket: %v", err)
			break
		}
	}
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {

}
