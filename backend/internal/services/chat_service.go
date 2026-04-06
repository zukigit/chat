package services

import (
	"context"
	"database/sql"
	"fmt"

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
}

// NewChatServer creates a new ChatServer instance.
func NewChatServer(sqlDB *sql.DB) *ChatServer {
	return &ChatServer{sqlDB: sqlDB}
}

// CreateConversation creates a new group conversation or a DM between two users.
// For DMs (is_group=false), if a conversation already exists between the two users
// the existing conversation_id is returned without creating a duplicate.
func (s *ChatServer) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.CreateConversationResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.GetMembersId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "members_id must not be empty")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "CreateConversation: begin tx: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	var conversationID int64

	if req.GetIsGroup() {
		conversationID, err = s.createGroupConversation(ctx, q, callerID, req)
	} else {
		conversationID, err = s.createOrGetDmConversation(ctx, q, callerID, req)
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "CreateConversation: commit: %v", err)
	}

	return &pb.CreateConversationResponse{
		ConversationId: fmt.Sprintf("%d", conversationID),
	}, nil
}

func (s *ChatServer) createGroupConversation(ctx context.Context, q *db.Queries, callerID uuid.UUID, req *pb.CreateConversationRequest) (int64, error) {
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
	for _, memberIDStr := range req.GetMembersId() {
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			return 0, status.Errorf(codes.InvalidArgument, "invalid member_id %q: %v", memberIDStr, err)
		}
		if _, err := q.AddMemberWithRole(ctx, db.AddMemberWithRoleParams{
			ConversationID: conv.ID,
			UserID:         memberID,
			Role:           db.MemberRoleMember,
		}); err != nil {
			return 0, status.Errorf(codes.Internal, "createGroupConversation: add member %s: %v", memberIDStr, err)
		}
	}

	return conv.ID, nil
}

func (s *ChatServer) createOrGetDmConversation(ctx context.Context, q *db.Queries, callerID uuid.UUID, req *pb.CreateConversationRequest) (int64, error) {
	if len(req.GetMembersId()) != 1 {
		return 0, status.Error(codes.InvalidArgument, "DM conversations require exactly one member_id")
	}

	peerID, err := uuid.Parse(req.GetMembersId()[0])
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid member_id %q: %v", req.GetMembersId()[0], err)
	}
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
