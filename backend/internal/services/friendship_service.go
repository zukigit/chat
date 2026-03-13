package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
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
}

// NewFriendshipServer creates a new FriendshipServer instance.
func NewFriendshipServer(sqlDB *sql.DB) *FriendshipServer {
	return &FriendshipServer{sqlDB: sqlDB}
}

// callerUUID parses the caller's UUID from context and returns it.
// Returns an error gRPC status if the token did not carry a valid UUID —
// which should not happen in practice since the JWT interceptor sets it.
func callerUUID(ctx context.Context) (uuid.UUID, error) {
	raw := lib.CallerIDFrom(ctx)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.Internal, "invalid caller user_id in context: %v", err)
	}
	return id, nil
}

// orderedUUIDPair returns the two UUIDs sorted lexicographically so that
// first.String() < second.String(), satisfying the DB CHECK constraint.
func orderedUUIDPair(a, b uuid.UUID) (first, second uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

// SendFriendRequest handles a friend request from the caller to target_username.
// It creates the friendship row and notifies the target user.
func (s *FriendshipServer) SendFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	callerID, err := callerUUID(ctx)
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
		lib.ErrorLog.Printf("SendFriendRequest: begin tx: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	// Resolve target username → user_id.
	targetUser, err := q.GetUserByUsername(ctx, target)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.InvalidArgument, "user %q not found", target)
	}
	if err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: get target user: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}
	targetID := targetUser.UserID

	first, second := orderedUUIDPair(callerID, targetID)

	// Read the existing row (if any) to decide which write to perform.
	var doWrite func(*db.Queries) (db.Friendship, error)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		RequesterUserid: first,
		AddresseeUserid: second,
	})
	switch {
	case err == sql.ErrNoRows:
		// No prior relationship — INSERT a fresh request.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.SendFriendRequest(ctx, db.SendFriendRequestParams{
				RequesterUserid: first,
				AddresseeUserid: second,
				InitiatorUserid: callerID,
			})
		}
	case err != nil:
		lib.ErrorLog.Printf("SendFriendRequest: get friendship: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	case existing.Status == db.FriendshipStatusRejected:
		// Previous request was rejected — allow re-sending by resetting to pending.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.ResetFriendRequest(ctx, db.ResetFriendRequestParams{
				RequesterUserid: first,
				AddresseeUserid: second,
				InitiatorUserid: callerID,
			})
		}
	default:
		return nil, status.Errorf(codes.AlreadyExists, "friend request already exists with status: %s", existing.Status)
	}

	friendship, err := doWrite(q)
	if err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: write: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	if _, err := q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:    targetID,
		SenderID:  callerID,
		Type:      db.NotificationTypeFriendRequest,
		Message:   callerName + " sent you a friend request",
	}); err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: create notification: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: commit: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	return &pb.FriendResponse{Status: string(friendship.Status)}, nil
}

// AcceptFriendRequest accepts a pending friend request from target_username.
// Only the addressee (the one who received the request) may accept it.
// A notification is inserted for the original requester.
func (s *FriendshipServer) AcceptFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	return s.respondToRequest(ctx, req, db.FriendshipStatusAccepted)
}

// RejectFriendRequest rejects a pending friend request from target_username.
// Only the addressee may reject it.
func (s *FriendshipServer) RejectFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	return s.respondToRequest(ctx, req, db.FriendshipStatusRejected)
}

// respondToRequest is shared logic for Accept and Reject: it finds the pending
// friendship, verifies that the caller is the addressee, updates the status,
// and (on accept) notifies the original requester.
func (s *FriendshipServer) respondToRequest(ctx context.Context, req *pb.FriendRequest, newStatus db.FriendshipStatus) (*pb.FriendResponse, error) {
	callerID, err := callerUUID(ctx)
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

	// Resolve target username → user_id.
	targetUser, err := q.GetUserByUsername(ctx, target)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.InvalidArgument, "user %q not found", target)
	}
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: get target user: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	targetID := targetUser.UserID

	first, second := orderedUUIDPair(callerID, targetID)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		RequesterUserid: first,
		AddresseeUserid: second,
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
		RequesterUserid: first,
		AddresseeUserid: second,
		Status:          newStatus,
	})
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: update status: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// On accept: notify the original requester that their request was accepted.
	if newStatus == db.FriendshipStatusAccepted {
		if _, err := q.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:    targetID, // the original requester
			SenderID:  callerID,
			Type:      db.NotificationTypeFriendRequest,
			Message:   callerName + " accepted your friend request",
		}); err != nil {
			lib.ErrorLog.Printf("respondToRequest: create notification: %v", err)
			// Non-fatal.
		}
	}

	if err := tx.Commit(); err != nil {
		lib.ErrorLog.Printf("respondToRequest: commit: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.FriendResponse{Status: string(updated.Status)}, nil
}
