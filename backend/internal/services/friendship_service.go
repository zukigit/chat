package services

import (
	"context"
	"database/sql"

	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FriendshipServer implements the friendship.FriendshipServer interface.
type FriendshipServer struct {
	pb.UnimplementedFriendshipServer
	sqlDB *sql.DB
	notif *NotificationServer // nil disables notifications (e.g. in tests)
}

// NewFriendshipServer creates a new FriendshipServer instance.
// notif may be nil, in which case notifications are skipped.
func NewFriendshipServer(sqlDB *sql.DB, notif *NotificationServer) *FriendshipServer {
	return &FriendshipServer{sqlDB: sqlDB, notif: notif}
}

// SendFriendRequest handles a friend request from the caller to target_username.
// It creates the friendship row and notifies the target user.
func (s *FriendshipServer) SendFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}
	callerName := lib.CallerFrom(ctx)
	target := req.GetTargetUsername()

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "target_username is required")
	}
	if callerName == target {
		return nil, status.Error(codes.InvalidArgument, "cannot send a friend request to yourself")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: begin tx: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	targetUser, err := q.GetUserByUsername(ctx, target)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.InvalidArgument, "user %q not found", target)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: get target user: %v", err)
	}
	targetID := targetUser.UserID

	first, second := lib.OrderedUUIDPair(callerID, targetID)

	// Read the existing row (if any) to decide which write to perform.
	var doWrite func(*db.Queries) (db.Friendship, error)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		User1Userid: first,
		User2Userid: second,
	})
	switch {
	case err == sql.ErrNoRows:
		// No prior relationship — INSERT a fresh request.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.SendFriendRequest(ctx, db.SendFriendRequestParams{
				User1Userid:     first,
				User2Userid:     second,
				InitiatorUserid: callerID,
			})
		}
	case err != nil:
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: get friendship: %v", err)
	default:
		return nil, status.Errorf(codes.AlreadyExists, "friend request already exists with status: %s", existing.Status)
	}

	friendship, err := doWrite(q)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: write: %v", err)
	}

	if err := s.notif.Send(ctx, q, targetID, callerID, db.NotificationTypeFriendRequest, callerName+" sent you a friend request"); err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: create notification: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: commit: %v", err)
	}

	return &pb.FriendResponse{Status: string(friendship.Status)}, nil
}

// AcceptFriendRequest accepts a pending friend request from target_username.
// Only the addressee (the one who received the request) may accept it.
// A notification is inserted for the original requester.
func (s *FriendshipServer) AcceptFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	return s.respondToRequest(ctx, req, db.FriendshipStatusAccepted)
}

// RejectFriendRequest rejects a pending friend request from target_username
// by deleting the friendship record.
func (s *FriendshipServer) RejectFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}
	target := req.GetTargetUsername()

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "target_username is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("RejectFriendRequest: begin tx: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	defer tx.Rollback()

	q := db.New(tx)

	targetUser, err := q.GetUserByUsername(ctx, target)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.InvalidArgument, "user %q not found", target)
	}
	if err != nil {
		lib.ErrorLog.Printf("RejectFriendRequest: get target user: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	targetID := targetUser.UserID

	first, second := lib.OrderedUUIDPair(callerID, targetID)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		User1Userid: first,
		User2Userid: second,
	})
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "friend request not found")
	}
	if err != nil {
		lib.ErrorLog.Printf("RejectFriendRequest: get friendship: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if existing.Status != db.FriendshipStatusPending {
		return nil, status.Error(codes.FailedPrecondition, "friend request is not pending")
	}

	if callerID == existing.InitiatorUserid {
		return nil, status.Error(codes.PermissionDenied, "only the recipient can respond to a friend request")
	}

	if err := q.DeleteFriendship(ctx, db.DeleteFriendshipParams{
		User1Userid: first,
		User2Userid: second,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "RejectFriendRequest: delete: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "RejectFriendRequest: commit: %v", err)
	}

	return &pb.FriendResponse{Status: "rejected"}, nil
}

// respondToRequest is the shared logic for AcceptFriendRequest: it finds the
// pending friendship, verifies the caller is the addressee, updates status to
// accepted, and notifies the original requester.
func (s *FriendshipServer) respondToRequest(ctx context.Context, req *pb.FriendRequest, newStatus db.FriendshipStatus) (*pb.FriendResponse, error) {
	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}
	callerName := lib.CallerFrom(ctx)
	target := req.GetTargetUsername()

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "target_username is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: begin tx: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	defer tx.Rollback()

	q := db.New(tx)

	targetUser, err := q.GetUserByUsername(ctx, target)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.InvalidArgument, "user %q not found", target)
	}
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: get target user: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	targetID := targetUser.UserID

	first, second := lib.OrderedUUIDPair(callerID, targetID)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		User1Userid: first,
		User2Userid: second,
	})
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "friend request not found")
	}
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: get friendship: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if existing.Status != db.FriendshipStatusPending {
		return nil, status.Error(codes.FailedPrecondition, "friend request is not pending")
	}

	// The caller must be the recipient of the original request, not the initiator.
	if callerID == existing.InitiatorUserid {
		return nil, status.Error(codes.PermissionDenied, "only the recipient can respond to a friend request")
	}

	updated, err := q.UpdateFriendshipStatus(ctx, db.UpdateFriendshipStatusParams{
		User1Userid: first,
		User2Userid: second,
		Status:      newStatus,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "respondToRequest: update status: %v", err)
	}

	// Notify the original requester that their request was accepted.
	if err := s.notif.Send(ctx, q, targetID, callerID, db.NotificationTypeFriendRequest, callerName+" accepted your friend request"); err != nil {
		return nil, status.Errorf(codes.Internal, "respondToRequest: create notification: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "respondToRequest: commit: %v", err)
	}

	return &pb.FriendResponse{Status: string(updated.Status)}, nil
}

