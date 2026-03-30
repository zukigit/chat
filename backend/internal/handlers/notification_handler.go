package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// notificationClientInterface is the minimal interface NotificationHandler needs.
type notificationClientInterface interface {
	MarkNotificationRead(ctx context.Context, token, id string) error
}

// NotificationHandler holds dependencies for notification-related HTTP handlers.
type NotificationHandler struct {
	client notificationClientInterface
}

// NewNotificationHandler creates a NotificationHandler with the given gRPC client.
func NewNotificationHandler(client notificationClientInterface) *NotificationHandler {
	return &NotificationHandler{client: client}
}

// MarkNotificationRead handles POST /notifications/read
func (h *NotificationHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "missing or malformed Authorization header",
		})
		return
	}

	var req notificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: fmt.Sprintf("invalid request body: %s", err.Error()),
		})
		return
	}

	if err := h.client.MarkNotificationRead(r.Context(), token, req.ID); err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			lib.WriteJSON(w, http.StatusNotFound, lib.Response{Success: false, Message: st.Message()})
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.PermissionDenied:
			lib.WriteJSON(w, http.StatusForbidden, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{Success: true, Message: "notification status updated"})
}
