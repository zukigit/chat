package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// FriendshipClient wraps the gRPC Friendship client.
type FriendshipClient struct {
	client pb.FriendshipClient
	conn   *grpc.ClientConn
}

// NewFriendshipClient dials the backend gRPC server and returns a FriendshipClient.
// The caller is responsible for calling Close() when done.
func NewFriendshipClient(backendAddr string) (*FriendshipClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
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

// SendFriendRequest forwards a friend request to the backend via gRPC.
func (c *FriendshipClient) SendFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.SendFriendRequest(lib.WithToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}

// AcceptFriendRequest accepts a pending friend request via gRPC.
func (c *FriendshipClient) AcceptFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.AcceptFriendRequest(lib.WithToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}

// GetFriends returns the list of accepted friends for the caller.
func (c *FriendshipClient) GetFriends(ctx context.Context, token string) (*pb.GetFriendsResponse, error) {
	return c.client.GetFriends(lib.WithToken(ctx, token), &pb.GetFriendsRequest{})
}

// RejectFriendRequest rejects a pending friend request via gRPC.
func (c *FriendshipClient) RejectFriendRequest(ctx context.Context, token, targetUsername string) error {
	_, err := c.client.RejectFriendRequest(lib.WithToken(ctx, token), &pb.FriendRequest{
		TargetUsername: targetUsername,
	})
	return err
}
