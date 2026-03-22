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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
