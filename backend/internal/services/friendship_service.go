package services

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NatsPublisher is a minimal interface for publishing to a NATS subject.
// Implemented by *nats.Conn (via Publish) or any JetStream wrapper.
type NatsPublisher interface {
	Publish(subject string, data []byte) error
}

// FriendshipServer implements the friendship.FriendshipServer interface.
type FriendshipServer struct {
	pb.UnimplementedFriendshipServer
	sqlDB     *sql.DB
	publisher NatsPublisher // nil means NATS publish is disabled (e.g. in tests)
}

// NewFriendshipServer creates a new FriendshipServer instance.
// publisher may be nil, in which case live NATS notifications are skipped.
func NewFriendshipServer(sqlDB *sql.DB, publisher NatsPublisher) *FriendshipServer {
	return &FriendshipServer{sqlDB: sqlDB, publisher: publisher}
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
	case existing.Status == db.FriendshipStatusRejected:
		// Previous request was rejected — allow re-sending by resetting to pending.
		doWrite = func(qt *db.Queries) (db.Friendship, error) {
			return qt.ResetFriendRequest(ctx, db.ResetFriendRequestParams{
				User1Userid:     first,
				User2Userid:     second,
				InitiatorUserid: callerID,
			})
		}
	default:
		return nil, status.Errorf(codes.AlreadyExists, "friend request already exists with status: %s", existing.Status)
	}

	friendship, err := doWrite(q)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: write: %v", err)
	}

	notification, err := q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:   targetID,
		SenderID: callerID,
		Type:     db.NotificationTypeFriendRequest,
		Message:  callerName + " sent you a friend request",
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: create notification: %v", err)
	}

	notificationByte, err := json.Marshal(notification)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SendFriendRequest: marshal notification: %v", err)
	}

	// Live push — non-fatal if the target user is offline.
	s.publishIfOnline(targetID, notificationByte)

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

// RejectFriendRequest rejects a pending friend request from target_username.
// Only the addressee may reject it.
func (s *FriendshipServer) RejectFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.FriendResponse, error) {
	return s.respondToRequest(ctx, req, db.FriendshipStatusRejected)
}

// respondToRequest is shared logic for Accept and Reject: it finds the pending
// friendship, verifies that the caller is the addressee, updates the status,
// and (on accept) notifies the original requester.
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

	// On accept: notify the original requester that their request was accepted.
	if newStatus == db.FriendshipStatusAccepted {
		notification, err := q.CreateNotification(ctx, db.CreateNotificationParams{
			UserID:   targetID, // the original requester
			SenderID: callerID,
			Type:     db.NotificationTypeFriendRequest,
			Message:  callerName + " accepted your friend request",
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "respondToRequest: create notification: %v", err)
		}

		notificationByte, err := json.Marshal(notification)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "respondToRequest: marshal notification: %v", err)
		}

		// Live push — non-fatal if the requester is offline.
		s.publishIfOnline(targetID, notificationByte)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "respondToRequest: commit: %v", err)
	}

	return &pb.FriendResponse{Status: string(updated.Status)}, nil
}

// publishIfOnline looks up an active notification session for the given user and,
// if one exists, publishes a JSON notification to its listen_path.
// All errors are logged and swallowed — a missing or offline session is normal.
func (s *FriendshipServer) publishIfOnline(userID uuid.UUID, notificationByte []byte) {
	if s.publisher == nil {
		return
	}

	ctx := context.Background()

	conn, err := s.sqlDB.Conn(ctx)
	if err != nil {
		lib.ErrorLog.Printf("publishIfOnline: get conn: %v", err)
		return
	}
	defer conn.Close()

	q := db.New(conn)

	sessions, err := q.GetSession(ctx, db.GetSessionParams{
		UserUserid: userID,
		Type:       db.SessionTypeNotification,
	})
	if err != nil {
		lib.ErrorLog.Printf("publishIfOnline: get session: %v", err)
		return
	}
	if len(sessions) == 0 {
		lib.ErrorLog.Printf("there is no sessions for user %v", userID)
		// No active sessions — user is offline.
		return
	}

	for _, session := range sessions {
		subject := session.ListenPath.String
		if err := s.publisher.Publish(subject, notificationByte); err != nil {
			lib.ErrorLog.Printf("publishIfOnline: publish to %s: %v", subject, err)
			continue
		}
	}
}
