package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FriendshipHandler holds dependencies for friendship-related HTTP handlers.
type FriendshipHandler struct {
	client *clients.FriendshipClient
}

// NewFriendshipHandler creates a FriendshipHandler with the given gRPC client.
func NewFriendshipHandler(client *clients.FriendshipClient) *FriendshipHandler {
	return &FriendshipHandler{client: client}
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

// GetFriends handles GET /friends
func (h *FriendshipHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "missing or malformed Authorization header",
		})
		return
	}

	resp, err := h.client.GetFriends(r.Context(), token)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: st.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: st.Message()})
		}
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{Success: true, Data: resp.GetFriends()})
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
