package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"

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

// SearchUsers handles GET /users/search?q=<query>
// Returns a list of users matching the query by username or display name.
func (h *AuthHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	token, ok := lib.BearerToken(r)
	if !ok || token == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "Missing token"})
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "q query parameter is required",
		})
		return
	}

	resp, err := h.authClient.SearchUsers(r.Context(), token, query)
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
		Message: "users found",
		Data:    resp.Users,
	})
}

// GetGithubOAuthURL handles POST /oauth/github/url
// Returns the GitHub OAuth authorization URL.
func (h *AuthHandler) GetGithubOAuthURL(w http.ResponseWriter, r *http.Request) {
	url, err := h.authClient.GetGithubOAuthURL(r.Context())
	if err != nil {
		grpcStatus, _ := status.FromError(err)
		lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: grpcStatus.Message()})
		return
	}
	lib.WriteJSON(w, http.StatusOK, lib.Response{Success: true, Data: map[string]string{"url": url}})
}

// GithubOAuthCallback handles GET /oauth/github/callback
// GitHub redirects here after user authorization. The gateway exchanges the code
// for a short-lived token and redirects to the frontend.
func (h *AuthHandler) GithubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	frontendURL := lib.Getenv("FRONTEND_URL", "")
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, frontendURL+"?error=missing_code", http.StatusFound)
		return
	}
	shortLivedToken, username, err := h.authClient.GithubOAuthCallback(r.Context(), code)
	if err != nil {
		grpcStatus, _ := status.FromError(err)
		http.Redirect(w, r, frontendURL+"?error="+url.QueryEscape(grpcStatus.Message()), http.StatusFound)
		return
	}
	_ = username
	http.Redirect(w, r, frontendURL+"/callback?token="+url.QueryEscape(shortLivedToken), http.StatusFound)
}

// ExchangeToken handles POST /token/exchange
// Exchanges a short-lived OAuth token for a long-lived session token.
func (h *AuthHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	shortLivedToken, ok := lib.BearerToken(r)
	if !ok || shortLivedToken == "" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: "missing or malformed Authorization header"})
		return
	}
	longLivedToken, username, err := h.authClient.ExchangeToken(r.Context(), shortLivedToken)
	if err != nil {
		grpcStatus, _ := status.FromError(err)
		switch grpcStatus.Code() {
		case codes.Unauthenticated:
			lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{Success: false, Message: grpcStatus.Message()})
		default:
			lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{Success: false, Message: grpcStatus.Message()})
		}
		return
	}
	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Data:    map[string]string{"token": longLivedToken, "username": username},
	})
}
