package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatHandler holds dependencies for chat-related HTTP handlers.
type ChatHandler struct {
	client *clients.ChatClient
}

// NewChatHandler creates a ChatHandler with the given gRPC client.
func NewChatHandler(client *clients.ChatClient) *ChatHandler {
	return &ChatHandler{client: client}
}

// CreateConversation handles POST /conversations
func (h *ChatHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "missing or malformed Authorization header",
		})
		return
	}

	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	conversationID, err := h.client.CreateConversation(r.Context(), token, req.IsGroup, req.Name, req.MembersID)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
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

	lib.WriteJSON(w, http.StatusCreated, lib.Response{
		Success: true,
		Message: "conversation created",
		Data:    map[string]int64{"conversation_id": conversationID},
	})
}

// GetMessages handles GET /conversations/messages
// Query params: conversation_id (required), limit (optional), cursor (optional)
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "missing or malformed Authorization header",
		})
		return
	}

	q := r.URL.Query()

	convID, err := strconv.ParseInt(q.Get("conversation_id"), 10, 64)
	if err != nil || convID == 0 {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	var limit int32
	if s := q.Get("limit"); s != "" {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: "invalid limit"})
			return
		}
		limit = int32(v)
	}

	var cursor int64
	if s := q.Get("cursor"); s != "" {
		cursor, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{Success: false, Message: "invalid cursor"})
			return
		}
	}

	resp, err := h.client.GetMessages(r.Context(), token, convID, limit, cursor)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
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

	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Data:    resp,
	})
}
