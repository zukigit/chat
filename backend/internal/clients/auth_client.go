package clients

import (
	"context"

	"github.com/zukigit/chat/backend/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient wraps the gRPC Auth client to communicate with the backend.
type AuthClient struct {
	client auth.AuthClient
	conn   *grpc.ClientConn
}

// NewAuthClient dials the backend gRPC server and returns an AuthClient.
// The caller is responsible for calling Close() when done.
func NewAuthClient(backendAddr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &AuthClient{
		client: auth.NewAuthClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (a *AuthClient) Close() {
	a.conn.Close()
}

// Login forwards the login request to the backend via gRPC.
func (a *AuthClient) Login(ctx context.Context, username, password string) (token string, err error) {
	resp, err := a.client.Login(ctx, &auth.LoginRequest{
		UserName: username,
		Passwd:   password,
	})
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

// Signup forwards the signup request to the backend via gRPC.
func (a *AuthClient) Signup(ctx context.Context, username, password string) error {
	_, err := a.client.Signup(ctx, &auth.SignupRequest{
		UserName: username,
		Passwd:   password,
	})
	return err
}
