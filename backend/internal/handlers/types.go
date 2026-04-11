package handlers

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// loginRequest struct for /login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// signupRequest struct for /signup
type signupRequest struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"passwd,omitempty"` // required for email signup
	Code     string `json:"code,omitempty"`   // required for google signup
}

// friendshipRequest is the shared request body for all friendship endpoints.
type friendshipRequest struct {
	Username string `json:"username"`
}

// notificationRequest is the request body for notification status updates.
type notificationRequest struct {
	ID string `json:"id"`
}

// createConversationRequest is the request body for POST /conversations.
type createConversationRequest struct {
	IsGroup         bool     `json:"is_group"`
	Name            string   `json:"name,omitempty"`
	MembersUsername []string `json:"members_username"`
}

// chatSendRequest is the JSON payload a client sends over the chat WebSocket
// to post a message to a conversation.
type chatSendRequest struct {
	ConversationID   int64  `json:"conversation_id"`
	Content          string `json:"content"`
	MessageType      string `json:"message_type,omitempty"`
	ReplyToMessageID int64  `json:"reply_to_message_id,omitempty"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
