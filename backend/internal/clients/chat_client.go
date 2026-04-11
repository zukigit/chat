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
// For DMs, membersUsername must contain exactly one peer username.
// For groups, membersUsername lists all non-caller members; name must be non-empty.
func (c *ChatClient) CreateConversation(ctx context.Context, token string, isGroup bool, name string, membersUsername []string) (int64, error) {
	resp, err := c.client.CreateConversation(lib.WithToken(ctx, token), &pb.CreateConversationRequest{
		IsGroup:         isGroup,
		Name:            name,
		MembersUsername: membersUsername,
	})
	if err != nil {
		return 0, err
	}
	return resp.GetConversationId(), nil
}

// SendMessage sends a message to a conversation via gRPC.
// messageType defaults to "text" if empty. replyToMessageID is 0 if not a reply.
func (c *ChatClient) SendMessage(ctx context.Context, token string, conversationID int64, content, messageType string, replyToMessageID int64) (int64, error) {
	req := &pb.SendMessageRequest{
		ConversationId: conversationID,
		Content:        content,
		MessageType:    messageType,
	}
	if replyToMessageID != 0 {
		req.ReplyToMessageId = &replyToMessageID
	}
	resp, err := c.client.SendMessage(lib.WithToken(ctx, token), req)
	if err != nil {
		return 0, err
	}
	return resp.GetMessageId(), nil
}

// GetMessages retrieves paginated messages from a conversation via gRPC.
// cursor is the last seen message_id (0 for the first page).
func (c *ChatClient) GetMessages(ctx context.Context, token string, conversationID int64, limit int32, cursor int64) (*pb.GetMessagesResponse, error) {
	return c.client.GetMessages(lib.WithToken(ctx, token), &pb.GetMessagesRequest{
		ConversationId: conversationID,
		Limit:          limit,
		Cursor:         cursor,
	})
}

// UpdateLastDeliveredMessage tells the backend that the given message was delivered to the caller.
func (c *ChatClient) UpdateLastDeliveredMessage(ctx context.Context, token string, conversationID, messageID int64) error {
	_, err := c.client.UpdateLastDeliveredMessage(lib.WithToken(ctx, token), &pb.UpdateMessageRequest{
		ConversationId: conversationID,
		MessageId:      messageID,
	})
	return err
}
