package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/chat"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatServer implements the chat.ChatServer interface.
type ChatServer struct {
	pb.UnimplementedChatServer
	sqlDB *sql.DB
	notif *NotificationServer // nil disables notifications (e.g. in tests)
}

// NewChatServer creates a new ChatServer instance.
// notif may be nil, in which case notifications are skipped.
func NewChatServer(sqlDB *sql.DB, notif *NotificationServer) *ChatServer {
	return &ChatServer{sqlDB: sqlDB, notif: notif}
}

// CreateConversation creates a new group conversation or a DM between two users.
// For DMs (is_group=false), if a conversation already exists between the two users
// the existing conversation_id is returned without creating a duplicate.
func (s *ChatServer) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.CreateConversationResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.GetMembersUsername()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "members_username must not be empty")
	}

	// Resolve usernames to UUIDs and verify each member is an accepted friend.
	q0 := db.New(s.sqlDB)
	memberIDs := make([]uuid.UUID, 0, len(req.GetMembersUsername()))
	for _, username := range req.GetMembersUsername() {
		user, err := q0.GetUserByUsername(ctx, username)
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %q not found", username)
		}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "CreateConversation: lookup user %q: %v", username, err)
		}
		memberID := user.UserID
		if memberID == callerID {
			memberIDs = append(memberIDs, memberID)
			continue // self-as-member is validated downstream
		}
		first, second := lib.OrderedUUIDPair(callerID, memberID)
		friendship, err := q0.GetFriendship(ctx, db.GetFriendshipParams{
			User1Userid: first,
			User2Userid: second,
		})
		if err == sql.ErrNoRows || (err == nil && friendship.Status != db.FriendshipStatusAccepted) {
			return nil, status.Errorf(codes.PermissionDenied, "user %q is not a friend", username)
		}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "CreateConversation: check friendship: %v", err)
		}
		memberIDs = append(memberIDs, memberID)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "CreateConversation: begin tx: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	var conversationID int64

	if req.GetIsGroup() {
		conversationID, err = s.createGroupConversation(ctx, q, callerID, req, memberIDs)
	} else {
		conversationID, err = s.createOrGetDmConversation(ctx, q, callerID, memberIDs)
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "CreateConversation: commit: %v", err)
	}

	return &pb.CreateConversationResponse{
		ConversationId: conversationID,
	}, nil
}

func (s *ChatServer) createGroupConversation(ctx context.Context, q *db.Queries, callerID uuid.UUID, req *pb.CreateConversationRequest, memberIDs []uuid.UUID) (int64, error) {
	if req.GetName() == "" {
		return 0, status.Error(codes.InvalidArgument, "name is required for group conversations")
	}

	conv, err := q.CreateConversation(ctx, db.CreateConversationParams{
		IsGroup: true,
		Name:    sql.NullString{Valid: true, String: req.GetName()},
	})
	if err != nil {
		return 0, status.Errorf(codes.Internal, "createGroupConversation: create conversation: %v", err)
	}

	// Add the caller as the group owner.
	if _, err := q.AddMemberWithRole(ctx, db.AddMemberWithRoleParams{
		ConversationID: conv.ID,
		UserID:         callerID,
		Role:           db.MemberRoleOwner,
	}); err != nil {
		return 0, status.Errorf(codes.Internal, "createGroupConversation: add owner: %v", err)
	}

	// Add each requested member as a regular member.
	for _, memberID := range memberIDs {
		if _, err := q.AddMemberWithRole(ctx, db.AddMemberWithRoleParams{
			ConversationID: conv.ID,
			UserID:         memberID,
			Role:           db.MemberRoleMember,
		}); err != nil {
			return 0, status.Errorf(codes.Internal, "createGroupConversation: add member %s: %v", memberID, err)
		}
	}

	return conv.ID, nil
}

func (s *ChatServer) createOrGetDmConversation(ctx context.Context, q *db.Queries, callerID uuid.UUID, memberIDs []uuid.UUID) (int64, error) {
	if len(memberIDs) != 1 {
		return 0, status.Error(codes.InvalidArgument, "DM conversations require exactly one member_username")
	}

	peerID := memberIDs[0]
	if callerID == peerID {
		return 0, status.Error(codes.InvalidArgument, "cannot create a DM with yourself")
	}

	first, second := lib.OrderedUUIDPair(callerID, peerID)

	// Return existing DM if one already exists.
	existing, err := q.GetDmPeer(ctx, db.GetDmPeerParams{
		User1ID: first,
		User2ID: second,
	})
	if err == nil {
		return existing.ConversationID, nil
	}
	if err != sql.ErrNoRows {
		return 0, status.Errorf(codes.Internal, "createOrGetDmConversation: get dm peer: %v", err)
	}

	conv, err := q.CreateConversation(ctx, db.CreateConversationParams{
		IsGroup: false,
		Name:    sql.NullString{Valid: false},
	})
	if err != nil {
		return 0, status.Errorf(codes.Internal, "createOrGetDmConversation: create conversation: %v", err)
	}

	for _, memberID := range []uuid.UUID{callerID, peerID} {
		if _, err := q.AddMemberWithRole(ctx, db.AddMemberWithRoleParams{
			ConversationID: conv.ID,
			UserID:         memberID,
			Role:           db.MemberRoleMember,
		}); err != nil {
			return 0, status.Errorf(codes.Internal, "createOrGetDmConversation: add member: %v", err)
		}
	}

	if _, err := q.CreateDmPeer(ctx, db.CreateDmPeerParams{
		User1ID:        first,
		User2ID:        second,
		ConversationID: conv.ID,
	}); err != nil {
		return 0, status.Errorf(codes.Internal, "createOrGetDmConversation: create dm peer: %v", err)
	}

	return conv.ID, nil
}

const (
	defaultMessageLimit = 50
	maxMessageLimit     = 100
)

// SendMessage posts a message to a conversation on behalf of the authenticated caller.
// The caller must be a member of the conversation.
func (s *ChatServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetContent() == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if req.GetConversationId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	msgType := db.MessageType(req.GetMessageType())
	if msgType == "" {
		msgType = db.MessageTypeText
	}

	switch msgType {
	case db.MessageTypeText, db.MessageTypeImage, db.MessageTypeFile, db.MessageTypeAudio:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid message_type %q", msgType)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendMessage: begin tx: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	isMember, err := q.IsMember(ctx, db.IsMemberParams{
		ConversationID: req.GetConversationId(),
		UserID:         callerID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendMessage: check membership: %v", err)
	}
	if !isMember {
		return nil, status.Error(codes.PermissionDenied, "caller is not a member of this conversation")
	}

	var replyTo sql.NullInt64
	if r := req.GetReplyToMessageId(); r != 0 {
		replyTo = sql.NullInt64{Valid: true, Int64: r}
	}

	msg, err := q.SendMessage(ctx, db.SendMessageParams{
		ConversationID:   req.GetConversationId(),
		SenderID:         callerID,
		ReplyToMessageID: replyTo,
		Content:          req.GetContent(),
		MessageType:      msgType,
		MediaUrl:         sql.NullString{},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendMessage: insert: %v", err)
	}

	// Notify all conversation members except the sender.
	members, err := q.GetConversationMembers(ctx, req.GetConversationId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendMessage: get members: %v", err)
	}
	callerName := lib.CallerFrom(ctx)
	for _, m := range members {
		if m.UserID == callerID {
			continue
		}
		if err := s.notif.Send(ctx, q, db.CreateNotificationParams{
			UserID:      m.UserID,
			SenderID:    uuid.NullUUID{Valid: true, UUID: callerID},
			Type:        db.NotificationTypeMessage,
			Message:     fmt.Sprintf("%s sent a message", callerName),
			ReferenceID: sql.NullInt64{Valid: true, Int64: req.GetConversationId()},
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "SendMessage: notify member: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "SendMessage: commit: %v", err)
	}

	// Publish the saved message (including its ID) to each member's chat session via NATS.
	if s.notif != nil {
		msgBytes, err := lib.NewChatEnvelope(lib.ChatEventMessage, msg)
		if err == nil {
			for _, m := range members {
				if m.UserID == callerID {
					continue
				}
				s.notif.publish(m.UserID, db.SessionTypeChat, msgBytes)
			}
		}
	}

	return &pb.SendMessageResponse{MessageId: msg.ID}, nil
}

// GetMessages returns a paginated list of messages in a conversation.
// The caller must be a member of the conversation.
// cursor is the last seen message_id (0 for the first page).
// limit defaults to 50 (max 100).
func (s *ChatServer) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetConversationId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	limit := int32(defaultMessageLimit)
	if req.GetLimit() > 0 {
		limit = req.GetLimit()
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}

	q := db.New(s.sqlDB)

	isMember, err := q.IsMember(ctx, db.IsMemberParams{
		ConversationID: req.GetConversationId(),
		UserID:         callerID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "GetMessages: check membership: %v", err)
	}
	if !isMember {
		return nil, status.Error(codes.PermissionDenied, "caller is not a member of this conversation")
	}

	rows, err := q.GetConversationMessages(ctx, db.GetConversationMessagesParams{
		ConversationID: req.GetConversationId(),
		ID:             req.GetCursor(),
		Limit:          limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "GetMessages: query: %v", err)
	}

	messages := make([]*pb.Message, 0, len(rows))
	for _, r := range rows {
		m := &pb.Message{
			MessageId:   r.ID,
			SenderId:    r.SenderID.String(),
			Content:     r.Content,
			MessageType: string(r.MessageType),
			IsEdited:    r.IsEdited,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		}
		if r.ReplyToMessageID.Valid {
			m.ReplyToMessageId = r.ReplyToMessageID.Int64
		}
		messages = append(messages, m)
	}

	var nextCursor int64
	if int32(len(rows)) == limit {
		nextCursor = rows[len(rows)-1].ID
	}

	return &pb.GetMessagesResponse{
		Messages:   messages,
		NextCursor: nextCursor,
	}, nil
}

// UpdateLastDeliveredMessage marks a message as delivered for the calling user
// and notifies the original sender via NATS.
func (s *ChatServer) UpdateLastDeliveredMessage(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.UpdateMessageResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetConversationId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if req.GetMessageId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}

	q := db.New(s.sqlDB)
	result, err := q.UpdateLastDeliveredMessageID(ctx, db.UpdateLastDeliveredMessageIDParams{
		ConversationID:         req.GetConversationId(),
		UserID:                 callerID,
		LastDeliveredMessageID: req.GetMessageId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "UpdateLastDeliveredMessage: %v", err)
	}

	// Skip the delivery receipt if the DB row was not updated — meaning
	// last_delivered_message_id was already >= req.GetMessageId().
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return &pb.UpdateMessageResponse{}, nil
	}

	// Notify the original sender that their message was delivered.
	if s.notif != nil && req.GetUserId() != "" {
		senderID, err := uuid.Parse(req.GetUserId())
		if err == nil {
			receiptBytes, err := lib.NewChatEnvelope(lib.ChatEventDelivered, lib.DeliveredEvent{
				ConversationID: req.GetConversationId(),
				MessageID:      req.GetMessageId(),
			})
			if err == nil {
				s.notif.publish(senderID, db.SessionTypeChat, receiptBytes)
			}
		}
	}

	return &pb.UpdateMessageResponse{}, nil
}
