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
		case codes.NotFound, codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
				Success: false,
				Message: "invalid credentials",
			})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
				Success: false,
				Message: "internal server error",
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
		case codes.AlreadyExists:
			lib.WriteJSON(w, http.StatusConflict, lib.Response{
				Success: false,
				Message: "username already taken",
			})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
				Success: false,
				Message: "internal server error",
			})
		}
		return
	}

	lib.WriteJSON(w, http.StatusCreated, lib.Response{
		Success: true,
		Message: "user registered successfully",
	})
}
