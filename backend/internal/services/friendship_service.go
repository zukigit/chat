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
}

// NewFriendshipServer creates a new FriendshipServer instance.
func NewFriendshipServer(sqlDB *sql.DB) *FriendshipServer {
	return &FriendshipServer{sqlDB: sqlDB}
}

// SendFriendRequest handles a friend request from the caller to target_username.
// It creates the friendship row and notifies the target user.
func (s *FriendshipServer) SendFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	caller := lib.CallerFrom(ctx)
	target := req.GetTargetUsername()

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "target_username is required")
	}
	if caller == target {
		return nil, status.Error(codes.InvalidArgument, "cannot send a friend request to yourself")
	}

	first, second := lib.OrderedPair(caller, target)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: begin tx: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}
	defer tx.Rollback()

	q := db.New(tx)

	if _, err = q.GetUserByUsername(ctx, caller); err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: get user caller: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "caller %s not found: %v", caller, err)
	}
	if _, err = q.GetUserByUsername(ctx, target); err != nil {
		lib.ErrorLog.Printf("SendFriendRequest: get user target: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "target %s not found: %v", target, err)
	}

	// Read the existing row (if any) to decide which write to perform.
	// Using a closure keeps the single transaction, single notification, single commit pattern.
	var doWrite func(*db.Queries) (db.Friendship, error)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		RequesterUsername: first,
		AddresseeUsername: second,
	})
	switch {
	case err == sql.ErrNoRows:
		// No prior relationship — INSERT a fresh request.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.SendFriendRequest(ctx, db.SendFriendRequestParams{
				RequesterUsername: first,
				AddresseeUsername: second,
			})
		}
	case err != nil:
		lib.ErrorLog.Printf("SendFriendRequest: get friendship: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	case existing.Status == db.FriendshipStatusRejected:
		// Previous request was rejected — allow re-sending by resetting to pending.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.UpdateFriendshipStatus(ctx, db.UpdateFriendshipStatusParams{
				RequesterUsername: first,
				AddresseeUsername: second,
				Status:            db.FriendshipStatusPending,
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
		UserUsername: target,
		Type:         db.NotificationTypeFriendRequest,
		MessageID:    sql.NullInt64{Valid: false},
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
	caller := lib.CallerFrom(ctx)
	target := req.GetTargetUsername()

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "target_username is required")
	}

	first, second := lib.OrderedPair(caller, target)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: begin tx: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	defer tx.Rollback()

	q := db.New(tx)

	existing, err := q.GetFriendship(ctx, db.GetFriendshipParams{
		RequesterUsername: first,
		AddresseeUsername: second,
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

	// The real addressee is whoever did NOT initiate the request.
	// Because we always store rows in lexicographic order, the original
	// requester is the one whose username is lexicographically smaller — but
	// we track who actually sent the request by checking who is NOT the
	// "addressee_username" in the stored row (the larger username).
	// The caller must be the target of the original request.
	// Since the DB row's requester_username < addressee_username always,
	// we compare caller against the stored addressee_username:
	// the addressee is whoever is NOT the original sender (i.e., target here).
	if caller == existing.RequesterUsername {
		return nil, status.Error(codes.PermissionDenied, "only the recipient can respond to a friend request")
	}

	updated, err := q.UpdateFriendshipStatus(ctx, db.UpdateFriendshipStatusParams{
		RequesterUsername: first,
		AddresseeUsername: second,
		Status:            newStatus,
	})
	if err != nil {
		lib.ErrorLog.Printf("respondToRequest: update status: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// On accept: notify the original requester that their request was accepted.
	if newStatus == db.FriendshipStatusAccepted {
		if _, err := q.CreateNotification(ctx, db.CreateNotificationParams{
			UserUsername: existing.RequesterUsername,
			Type:         db.NotificationTypeFriendRequest,
			MessageID:    sql.NullInt64{Valid: false},
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
