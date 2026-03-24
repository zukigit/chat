package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SessionClient wraps the gRPC Session client.
type SessionClient struct {
	client pb.SessionClient
	conn   *grpc.ClientConn
}

// NewSessionClient dials the backend gRPC server and returns a SessionClient.
// The caller is responsible for calling Close() when done.
func NewSessionClient(backendAddr string) (*SessionClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SessionClient{
		client: pb.NewSessionClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *SessionClient) Close() {
	c.conn.Close()
}

// AddSession forwards an add session request to the backend via gRPC.
func (c *SessionClient) AddSession(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error) {
	return c.client.AddSession(lib.WithToken(ctx, token), &pb.AddSessionRequest{
		Type: sessionType,
	})
}

// SetSessionStatus forwards a set session status request to the backend via gRPC.
func (c *SessionClient) SetSessionStatus(ctx context.Context, token, sessionID, status string) error {
	_, err := c.client.SetSessionStatus(lib.WithToken(ctx, token), &pb.SetSessionStatusRequest{
		SessionId: sessionID,
		Status:    status,
	})
	return err
}

// DeleteSession forwards a delete session request to the backend via gRPC.
func (c *SessionClient) DeleteSession(ctx context.Context, token, sessionID string) error {
	_, err := c.client.DeleteSession(lib.WithToken(ctx, token), &pb.DeleteSessionRequest{
		SessionId: sessionID,
	})
	return err
}
