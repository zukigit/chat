package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NotificationClient wraps the gRPC Notification client.
type NotificationClient struct {
	client pb.NotificationClient
	conn   *grpc.ClientConn
}

// NewNotificationClient dials the backend gRPC server and returns a NotificationClient.
// The caller is responsible for calling Close() when done.
func NewNotificationClient(backendAddr string) (*NotificationClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &NotificationClient{
		client: pb.NewNotificationClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *NotificationClient) Close() {
	c.conn.Close()
}

// MarkNotificationRead marks a notification as read via gRPC.
func (c *NotificationClient) MarkNotificationRead(ctx context.Context, token, id string) error {
	_, err := c.client.MarkNotificationRead(lib.WithToken(ctx, token), &pb.MarkNotificationReadRequest{
		Id: id,
	})
	return err
}
