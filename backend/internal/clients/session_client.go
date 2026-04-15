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

// ValidateSession asks the backend to confirm that login_id exists in the sessions
// table (i.e. the user has not logged out) and returns the owning user_id.
func (c *SessionClient) ValidateSession(ctx context.Context, token, loginID string) (string, error) {
	resp, err := c.client.ValidateSession(lib.WithToken(ctx, token), &pb.ValidateSessionRequest{
		LoginId: loginID,
	})
	if err != nil {
		return "", err
	}
	return resp.UserId, nil
}
