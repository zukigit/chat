package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ChatClient wraps the gRPC Chat client.
type ChatClient struct {
	client pb.ChatClient
	conn   *grpc.ClientConn
}

// NewChatClient dials the backend gRPC server and returns a ChatClient.
// The caller is responsible for calling Close() when done.
func NewChatClient(backendAddr string) (*ChatClient, error) {
	conn, err := grpc.NewClient(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		client: pb.NewChatClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *ChatClient) Close() {
	c.conn.Close()
}

// CreateConversation forwards a create conversation request to the backend via gRPC.
// For DMs, membersID must contain exactly one peer user ID.
// For groups, membersID lists all non-caller members; name must be non-empty.
func (c *ChatClient) CreateConversation(ctx context.Context, token string, isGroup bool, name string, membersID []string) (string, error) {
	resp, err := c.client.CreateConversation(lib.WithToken(ctx, token), &pb.CreateConversationRequest{
		IsGroup:   isGroup,
		Name:      name,
		MembersId: membersID,
	})
	if err != nil {
		return "", err
	}
	return resp.GetConversationId(), nil
}
