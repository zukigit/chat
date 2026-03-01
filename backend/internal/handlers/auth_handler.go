package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authClientInterface is the minimal interface AuthHandler needs.
// The concrete *clients.AuthClient satisfies it automatically.
type authClientInterface interface {
	Login(ctx context.Context, username, password string) (string, error)
	Signup(ctx context.Context, username, password string) error
}

// AuthHandler holds dependencies for auth-related HTTP handlers.
type AuthHandler struct {
	authClient authClientInterface
}

// NewAuthHandler creates an AuthHandler with the given gRPC auth client.
func NewAuthHandler(authClient authClientInterface) *AuthHandler {
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
		lib.ErrorLog.Printf("Signup failed: %v", err)
		grpcStatus, _ := status.FromError(err)
		switch grpcStatus.Code() {
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
