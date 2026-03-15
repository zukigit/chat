package handlers

import "context"

// sessionClientInterface is the minimal interface SessionHandler needs.
type sessionClientInterface interface {
	AddSession(ctx context.Context, token, sessionType string) error
	SetSessionStatus(ctx context.Context, token, sessionID, status string) error
}

// SessionHandler holds dependencies for session-related HTTP handlers.
type SessionHandler struct {
	client sessionClientInterface
}

// NewSessionHandler creates a SessionHandler with the given gRPC client.
func NewSessionHandler(client sessionClientInterface) *SessionHandler {
	return &SessionHandler{client: client}
}
