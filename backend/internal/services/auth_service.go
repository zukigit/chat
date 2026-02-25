package services

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/proto/auth"
)

// AuthServer implements the auth.AuthServer interface
type AuthServer struct {
	auth.UnimplementedAuthServer
}

// NewAuthServer creates a new AuthServer instance
func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

// Login handles user login
func (s *AuthServer) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	lib.InfoLog.Printf("Login request for user: %s", req.UserName)

	// TODO: Implement actual authentication logic
	// For now, just return a dummy token
	token := "dummy_token_" + req.UserName

	return &auth.LoginResponse{
		Token: token,
	}, nil
}

// Signup handles user signup
func (s *AuthServer) Signup(ctx context.Context, req *auth.SignupRequest) (*auth.SignupResponse, error) {
	lib.InfoLog.Printf("Signup request for user: %s", req.UserName)

	// TODO: Implement actual user registration logic

	return &auth.SignupResponse{}, nil
}
