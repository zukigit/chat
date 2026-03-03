package clients

import (
	"context"

	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// FriendshipClient wraps the gRPC Friendship client.
type FriendshipClient struct {
	client pb.FriendshipClient
	conn   *grpc.ClientConn
}

// NewFriendshipClient dials the backend gRPC server and returns a FriendshipClient.
// The caller is responsible for calling Close() when done.
func NewFriendshipClient(backendAddr string) (*FriendshipClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &FriendshipClient{
		client: pb.NewFriendshipClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *FriendshipClient) Close() {
	c.conn.Close()
}

// withToken attaches the Bearer token to the outgoing gRPC context.
func withToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// SendFriendRequest forwards a friend request to the backend via gRPC.
func (c *FriendshipClient) SendFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.SendFriendRequest(withToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}

// AcceptFriendRequest accepts a pending friend request via gRPC.
func (c *FriendshipClient) AcceptFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.AcceptFriendRequest(withToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}

// RejectFriendRequest rejects a pending friend request via gRPC.
func (c *FriendshipClient) RejectFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.RejectFriendRequest(withToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}
