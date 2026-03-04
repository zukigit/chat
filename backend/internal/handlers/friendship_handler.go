package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// friendshipClientInterface is the minimal interface FriendshipHandler needs.
type friendshipClientInterface interface {
	SendFriendRequest(ctx context.Context, token, targetUsername string) error
	AcceptFriendRequest(ctx context.Context, token, targetUsername string) error
	RejectFriendRequest(ctx context.Context, token, targetUsername string) error
}

// FriendshipHandler holds dependencies for friendship-related HTTP handlers.
type FriendshipHandler struct {
	client friendshipClientInterface
}

// NewFriendshipHandler creates a FriendshipHandler with the given gRPC client.
func NewFriendshipHandler(client friendshipClientInterface) *FriendshipHandler {
	return &FriendshipHandler{client: client}
}

// friendshipRequest is the shared request body for all friendship endpoints.
type friendshipRequest struct {
	Username string `json:"username"`
}

// handleFriendshipAction is shared logic for Send / Accept / Reject.
func (h *FriendshipHandler) handleFriendshipAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(ctx context.Context, token, target string) error,
	successMsg string,
) {
	token, ok := lib.BearerToken(r)
	if !ok {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "missing or malformed Authorization header",
		})
		return
	}

	var req friendshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body: username is required",
		})
		return
	}

	if err := action(r.Context(), token, req.Username); err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.AlreadyExists:
			lib.WriteJSON(w, http.StatusConflict, lib.Response{Success: false, Message: st.Message()})
		case codes.NotFound:
			lib.WriteJSON(w, http.StatusNotFound, lib.Response{Success: false, Message: st.Message()})
		case codes.PermissionDenied:
			lib.WriteJSON(w, http.StatusForbidden, lib.Response{Success: false, Message: st.Message()})
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: st.Message()})
		case codes.FailedPrecondition:
			lib.WriteJSON(w, http.StatusConflict, lib.Response{Success: false, Message: st.Message()})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{Success: true, Message: successMsg})
}

// SendFriendRequest handles POST /friends/request
func (h *FriendshipHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.handleFriendshipAction(w, r, h.client.SendFriendRequest, "friend request sent")
}

// AcceptFriendRequest handles POST /friends/accept
func (h *FriendshipHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.handleFriendshipAction(w, r, h.client.AcceptFriendRequest, "friend request accepted")
}

// RejectFriendRequest handles POST /friends/reject
func (h *FriendshipHandler) RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.handleFriendshipAction(w, r, h.client.RejectFriendRequest, "friend request rejected")
}
