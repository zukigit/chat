package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	var opts []grpc.DialOption
	if lib.Getenv("GRPC_TLS_MODE", "") == "disable" {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}
	conn, err := grpc.NewClient(backendAddr, opts...)
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

// GetListenPath asks the backend for the NATS subject path and durable consumer name.
// The backend validates the JWT and returns a path like sessions.chat.<user_id>
// and a consumer name like chat-<login_id>.
func (c *SessionClient) GetListenPath(ctx context.Context, token, listenType string) (listenPath string, consumerName string, err error) {
	resp, err := c.client.GetListenPath(lib.WithToken(ctx, token), &pb.GetListenPathRequest{
		Type: listenType,
	})
	if err != nil {
		return "", "", err
	}
	return resp.ListenPath, resp.ConsumerName, nil
}

// Ping sends a ping to the backend and expects a "pong" response.
func (c *SessionClient) Ping(ctx context.Context) error {
	resp, err := c.client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return err
	}
	lib.InfoLog.Printf("Backend ping response: %s", resp.GetMessage())
	return nil
}
