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

ctx, cancel := context.WithCancel(r.Context())
defer cancel()

consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
Durable:           "noti-" + claims.LoginID,
FilterSubjects:    []string{lib.NotiSubjectPrefix + claims.UserID},
AckPolicy:         jetstream.AckExplicitPolicy,
DeliverPolicy:     jetstream.DeliverNewPolicy,
InactiveThreshold: 72 * time.Hour,
})
if err != nil {
closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to create consumer: %v", err))
_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
return
}

// Read channel from WS client so we detect close/disconnect.
go func() {
for {
if _, _, err := conn.ReadMessage(); err != nil {
lib.InfoLog.Printf("noti session closing: loginID: %s, err: %v", claims.LoginID, err)
cancel()
return
}
}
}()

cc, err := consumer.Consume(
func(msg jetstream.Msg) {
if err := conn.WriteMessage(websocket.TextMessage, msg.Data()); err != nil {
lib.ErrorLog.Printf("Error writing to websocket: loginID: %s, err: %v", claims.LoginID, err)
}
msg.Ack()
},
jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
if errors.Is(err, jetstream.ErrConsumerDeleted) {
lib.InfoLog.Printf("consumer deleted by server: loginID: %s", claims.LoginID)
closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "consumer deleted by server")
_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
cancel()
return
}
lib.ErrorLog.Printf("consumer error: loginID: %s, err: %v", claims.LoginID, err)
}),
)
if err != nil {
closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to consume messages: %v", err))
_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
return
}
defer cc.Stop()

<-ctx.Done()
}

func (s *SessionHandler) ChatSession(w http.ResponseWriter, r *http.Request) {
lib.InfoLog.Printf("new chat session request from %s", r.RemoteAddr)

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
lib.InfoLog.Printf("chat session established: loginID: %s", claims.LoginID)

ctx, cancel := context.WithCancel(r.Context())
defer cancel()

consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
Durable:           "chat-" + claims.LoginID,
FilterSubjects:    []string{lib.ChatSubjectPrefix + claims.UserID},
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
lib.InfoLog.Printf("closing chat session: loginID: %s, err: %v", claims.LoginID, err)
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

// Deliver incoming NATS messages to the WS client.
cc, err := consumer.Consume(
func(msg jetstream.Msg) {
if err := conn.WriteMessage(websocket.TextMessage, msg.Data()); err != nil {
lib.ErrorLog.Printf("Error writing to websocket: loginID: %s, err: %v", claims.LoginID, err)
return // don't ack; let JetStream retry
}

var env lib.ChatEnvelope
var message db.Message
if err := json.Unmarshal(msg.Data(), &env); err != nil {
lib.ErrorLog.Printf("chat session: unmarshal envelope: loginID: %s, err: %v", claims.LoginID, err)
goto ack
}

if env.Type != lib.ChatEventMessage {
goto ack
}

if err := json.Unmarshal(env.Data, &message); err != nil {
lib.ErrorLog.Printf("chat session: unmarshal message: loginID: %s, err: %v", claims.LoginID, err)
goto ack
}

if err := s.chatClient.UpdateLastDeliveredMessage(context.Background(), token, message.ConversationID, message.ID, message.SenderID.String()); err != nil {
lib.ErrorLog.Printf("chat session: UpdateLastDeliveredMessage: loginID: %s, err: %v", claims.LoginID, err)
}

ack:
msg.Ack()
},
jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
if errors.Is(err, jetstream.ErrConsumerDeleted) {
lib.InfoLog.Printf("consumer deleted by server: loginID: %s", claims.LoginID)
closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "consumer deleted by server")
_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
cancel()
return
}
lib.ErrorLog.Printf("consumer error: loginID: %s, err: %v", claims.LoginID, err)
}),
)
if err != nil {
closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to consume messages: %v", err))
_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
return
}
defer cc.Stop()

<-ctx.Done()
}
