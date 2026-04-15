package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/backend/internal/clients"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler holds dependencies for auth-related HTTP handlers.
type AuthHandler struct {
	authClient *clients.AuthClient
}

// NewAuthHandler creates an AuthHandler with the given gRPC auth client.
func NewAuthHandler(authClient *clients.AuthClient) *AuthHandler {
	return &AuthHandler{authClient: authClient}
}

// Login handles POST /login
// Decodes credentials from the request body, forwards them to the backend via gRPC,
// and returns a JWT token on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.authClient == nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: "authClient is nil",
		})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	token, err := h.authClient.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		grpcStatus, _ := status.FromError(err)
		switch grpcStatus.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		}
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Message: "login successful",
		Data:    map[string]string{"token": token},
	})
}

// Signup handles POST /signup
// Decodes signup details from the request body and forwards them to the backend via gRPC.
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if h.authClient == nil {
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
			Success: false,
			Message: "authClient is nil",
		})
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	if err := h.authClient.Signup(r.Context(), req.Username, req.Password); err != nil {
		grpcStatus, _ := status.FromError(err)
		switch grpcStatus.Code() {
		case codes.InvalidArgument:
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		case codes.AlreadyExists:
			lib.WriteJSON(w, http.StatusConflict, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
				Success: false,
				Message: grpcStatus.Message(),
			})
		}
		return
	}

	lib.WriteJSON(w, http.StatusCreated, lib.Response{
		Success: true,
		Message: "user registered successfully",
	})
}

// Logout handles POST /logout
// Deletes the caller's session from the backend, invalidating the login_id.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok || token == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Missing token"})
		return
	}

	if err := h.authClient.Logout(r.Context(), token); err != nil {
		grpcStatus, _ := status.FromError(err)
		switch grpcStatus.Code() {
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: grpcStatus.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: grpcStatus.Message()})
		}
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{Success: true, Message: "logged out"})
}
