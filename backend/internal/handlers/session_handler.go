package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionHandler holds dependencies for session-related HTTP handlers.
type SessionHandler struct {
	client     *clients.SessionClient
	chatClient *clients.ChatClient
	stream     jetstream.Stream
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client *clients.SessionClient, chatClient *clients.ChatClient, stream jetstream.Stream) *SessionHandler {
	return &SessionHandler{client: client, chatClient: chatClient, stream: stream}
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
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	// Extract LoginID from the token without verifying the signature — the
	// backend already validated it. Used to name the durable consumer so
	// this login session resumes from where it left off on reconnect.
	var notiConsumerName string
	if claims, err := lib.ParseTokenUnverified(token); err == nil && claims.LoginID != "" {
		notiConsumerName = "noti-" + claims.LoginID
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
		Durable:           notiConsumerName,
		FilterSubjects:    []string{addSessionResp.ListenPath},
		AckPolicy:         jetstream.AckExplicitPolicy,
		DeliverPolicy:     jetstream.DeliverNewPolicy,
		InactiveThreshold: 24 * time.Hour,
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
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	// Block until the client disconnects (the read goroutine calls cancel() on
	// any WS error, which unblocks this and lets the deferred cleanup run).
	<-ctx.Done()
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {
	lib.InfoLog.Printf("new chat session request from %s", r.RemoteAddr)

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

	addSessionResp, err := s.client.AddSession(r.Context(), token, string(db.SessionTypeChat))
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	// Extract LoginID from the token without verifying the signature — the
	// backend already validated it. Used to name the durable consumer so
	// this login session resumes from where it left off on reconnect.
	var chatConsumerName string
	if claims, err := lib.ParseTokenUnverified(token); err == nil && claims.LoginID != "" {
		chatConsumerName = "chat-" + claims.LoginID
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
	lib.InfoLog.Printf("chat session established: sessionId: %s", addSessionResp.SessionId)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:           chatConsumerName,
		FilterSubjects:    []string{addSessionResp.ListenPath},
		AckPolicy:         jetstream.AckExplicitPolicy,
		DeliverPolicy:     jetstream.DeliverNewPolicy,
		InactiveThreshold: 72 * time.Hour,
	})
	if err != nil {
		closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to create consumer: %v", err))
		_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
		return
	}

	// Read messages from the WS client and forward them to the chat backend.
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				lib.InfoLog.Printf("closing chat session: sessionId: %s, err: %v", addSessionResp.SessionId, err)
				cancel()
				return
			}

			var req chatSendRequest
			if err := json.Unmarshal(data, &req); err != nil {
				lib.ErrorLog.Printf("chat session: invalid message from client: %v", err)
				continue
			}

			if _, err := s.chatClient.SendMessage(ctx, token, req.ConversationID, req.Content, req.MessageType, req.ReplyToMessageID); err != nil {
				lib.ErrorLog.Printf("chat session: SendMessage: %v", err)
			}
		}
	}()

	// Deliver incoming NATS messages (sent by other users) to the WS client.
	cc, err := consumer.Consume(
		func(msg jetstream.Msg) {
			if err := conn.WriteMessage(websocket.TextMessage, msg.Data()); err != nil {
				lib.ErrorLog.Printf("Error writing to websocket: sessionId: %s, err: %v", addSessionResp.SessionId, err)
				return // don't ack the message if we failed to write to the client, so it can be retried
			}

			// Notify the backend that the message was delivered to this user.
			var env lib.ChatEnvelope
			var message db.Message
			if err := json.Unmarshal(msg.Data(), &env); err != nil {
				lib.ErrorLog.Printf("chat session: unmarshal envelope: sessionId: %s, err: %v", addSessionResp.SessionId, err)
				goto ack
			}

			if env.Type != lib.ChatEventMessage {
				goto ack
			}

			if err := json.Unmarshal(env.Data, &message); err != nil {
				lib.ErrorLog.Printf("chat session: unmarshal message: sessionId: %s, err: %v", addSessionResp.SessionId, err)
				goto ack
			}

			if err := s.chatClient.UpdateLastDeliveredMessage(context.Background(), token, message.ConversationID, message.ID, message.SenderID.String()); err != nil {
				lib.ErrorLog.Printf("chat session: UpdateLastDeliveredMessage: sessionId: %s, err: %v", addSessionResp.SessionId, err)
			}

		ack:
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
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	// Block until the client disconnects.
	<-ctx.Done()
}
