package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
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
	js         nats.JetStreamContext
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client *clients.SessionClient, chatClient *clients.ChatClient, js nats.JetStreamContext) *SessionHandler {
	return &SessionHandler{client: client, chatClient: chatClient, js: js}
}

func (s *SessionHandler) NotificationSession(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Missing token"})
		return
	}

	// Decode claims without signature verification (gateway has no JWT secret).
	// Actual auth is enforced by the gRPC interceptor inside ValidateSession.
	claims, err := lib.ParseTokenUnverified(token)
	if err != nil || claims.LoginID == "" || claims.UserID == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Invalid token"})
		return
	}

	// Confirm the session is still active (user has not logged out).
	if _, err := s.client.ValidateSession(r.Context(), token, claims.LoginID); err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to upgrade to websocket: %v", err),
		})
		return
	}
	defer func() {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}()

	// Read channel from WS client so we detect close/disconnect.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	// Deliver incoming NATS messages to the WS client.
	sub, err := s.js.Subscribe(lib.NotiSubjectPrefix+claims.UserID, func(msg *nats.Msg) {
		if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
			lib.ErrorLog.Printf("Error writing to websocket: loginID: %s, err: %v", claims.LoginID, err)
			return
		}
		msg.Ack()
	}, nats.Durable("noti-"+claims.LoginID), nats.BindStream("SESSIONS"))
	if err != nil {
		lib.ErrorLog.Printf("Failed to subscribe: %v", err)
		cancel()
		return
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
}

func (s *SessionHandler) GetChatEnvelopeRequestVersion(w http.ResponseWriter, r *http.Request) {
	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Data:    map[string]int{"chat_request_version": lib.ChatRequestEnvelopeVersion},
	})
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Missing token"})
		return
	}

	claims, err := lib.ParseTokenUnverified(token)
	if err != nil || claims.LoginID == "" || claims.UserID == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Invalid token"})
		return
	}

	if _, err := s.client.ValidateSession(r.Context(), token, claims.LoginID); err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: fmt.Sprintf("Failed to upgrade to websocket: %v", err),
		})
		return
	}
	defer func() {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}()

	// Read messages from the WS client and forward them to the chat backend.
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}

			var env lib.ChatRequestEnvelope
			if err := json.Unmarshal(data, &env); err != nil {
				lib.ErrorLog.Printf("chat session: invalid message from client: %v", err)
				continue
			}

			switch env.Type {
			case lib.ChatRequestSend:
				var req sendMessageRequest
				if err := json.Unmarshal(env.Data, &req); err != nil {
					lib.ErrorLog.Printf("chat session: unmarshal send request: %v", err)
					continue
				}
				if _, err := s.chatClient.SendMessage(ctx, token, req.ConversationID, req.Content, req.MessageType, req.ReplyToMessageID); err != nil {
					lib.ErrorLog.Printf("chat session: SendMessage: %v", err)
				}
			case lib.ChatRequestRead:
				var req readMessageRequest
				if err := json.Unmarshal(env.Data, &req); err != nil {
					lib.ErrorLog.Printf("chat session: unmarshal read request: %v", err)
					continue
				}
				if err := s.chatClient.UpdateLastReadMessage(ctx, token, req.ConversationID, req.MessageID, req.SenderID); err != nil {
					lib.ErrorLog.Printf("chat session: UpdateLastReadMessage: %v", err)
				}
			default:
				lib.ErrorLog.Printf("chat session: unknown request type: %q", env.Type)
			}
		}
	}()

	sub, err := s.js.Subscribe(lib.ChatSubjectPrefix+claims.UserID, func(msg *nats.Msg) {
		var env lib.ChatResponseEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			lib.ErrorLog.Printf("chat session: unmarshal envelope: loginID: %s, err: %v", claims.LoginID, err)
			msg.Ack()
			return
		}

		var message db.Message
		isMessageEvent := env.Type == lib.ChatEventMessage
		if isMessageEvent {
			if err := json.Unmarshal(env.Data, &message); err != nil {
				lib.ErrorLog.Printf("chat session: unmarshal message: loginID: %s, err: %v", claims.LoginID, err)
				msg.Ack()
				return
			}
			if message.SenderLoginID.String() == claims.LoginID {
				msg.Ack()
				return
			}
		}

		if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
			lib.ErrorLog.Printf("Error writing to websocket: loginID: %s, err: %v", claims.LoginID, err)
			return
		}

		if isMessageEvent {
			if err := s.chatClient.UpdateLastDeliveredMessage(context.Background(), token, message.ConversationID, message.ID, message.SenderID.String()); err != nil {
				lib.ErrorLog.Printf("chat session: UpdateLastDeliveredMessage: loginID: %s, err: %v", claims.LoginID, err)
			}
		}

		msg.Ack()
	}, nats.Durable("chat-"+claims.LoginID), nats.BindStream("SESSIONS"))
	if err != nil {
		lib.ErrorLog.Printf("Failed to subscribe: %v", err)
		cancel()
		return
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
}
