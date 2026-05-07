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

// GetListenPath asks the backend for the NATS subject path to listen on.
// The backend validates the JWT and returns a path like sessions.chat.<user_id>.<login_id>.
func (c *SessionClient) GetListenPath(ctx context.Context, token, listenType string) (string, error) {
	resp, err := c.client.GetListenPath(lib.WithToken(ctx, token), &pb.GetListenPathRequest{
		Type: listenType,
	})
	if err != nil {
		return "", err
	}
	return resp.ListenPath, nil
}
