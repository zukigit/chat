package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
)

// SessionHandler holds dependencies for session-related HTTP handlers.
type SessionHandler struct {
	client *clients.SessionClient
	stream jetstream.Stream
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client *clients.SessionClient, stream jetstream.Stream) *SessionHandler {
	return &SessionHandler{client: client, stream: stream}
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
			Message: fmt.Sprintf("Failed to add session: %v", err),
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
	defer func() {
		s.client.DeleteSession(context.Background(), token, addSessionResp.SessionId)

		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:           addSessionResp.SessionId,
		FilterSubjects:    []string{addSessionResp.ListenPath},
		AckPolicy:         jetstream.AckExplicitPolicy,
		InactiveThreshold: 24 * time.Hour, // auto-delete consumer if no client is consuming
	})
	if err != nil {
		closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to create consumer: %v", err))
		_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
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

	// it will go with another goroutine
	cc, err := consumer.Consume(
		func(msg jetstream.Msg) {
			if err := conn.WriteMessage(websocket.TextMessage, msg.Data()); err != nil {
				lib.ErrorLog.Printf("Error writing to websocket: sessionId: %s, err: %v", addSessionResp.SessionId, err)
			}
			msg.Ack()
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			if errors.Is(err, jetstream.ErrConsumerDeleted) {
				lib.InfoLog.Printf("consumer deleted by server: sessionId: %s", addSessionResp.SessionId)
				closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "consumer deleted by server")
				_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
				cancel()
				return
			}
			lib.ErrorLog.Printf("consumer error: sessionId: %s, err: %v", addSessionResp.SessionId, err)
		}),
	)
	if err != nil {
		closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to consume messages: %v", err))
		_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
		return
	}
	defer cc.Stop()

	// Set session status to active
	err = s.client.SetSessionStatus(context.Background(), token, addSessionResp.SessionId, string(db.SessionStatusActive))
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to set session status active: %v", err),
		})
		return
	}

	// Block until the client disconnects (the read goroutine calls cancel() on
	// any WS error, which unblocks this and lets the deferred cleanup run).
	<-ctx.Done()
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {

}
