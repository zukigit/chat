package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	pb "github.com/zukigit/chat/backend/proto/notification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NatsPublisher is a minimal interface for publishing to a NATS subject.
// Implemented by *nats.Conn (via Publish) or any JetStream wrapper.
type NatsPublisher interface {
	Publish(subject string, data []byte) error
}

// NotificationServer handles all notification concerns: persisting to the
// database and delivering live pushes via NATS.
type NotificationServer struct {
	pb.UnimplementedNotificationServer
	sqlDB     *sql.DB
	publisher NatsPublisher // nil disables live NATS pushes (e.g. in tests)
}

// NewNotificationServer creates a new NotificationServer.
// publisher may be nil to disable live NATS pushes.
func NewNotificationServer(sqlDB *sql.DB, publisher NatsPublisher) *NotificationServer {
	return &NotificationServer{sqlDB: sqlDB, publisher: publisher}
}

// Send persists a notification using q (which may wrap an active transaction)
// and pushes a live NATS event to the recipient if they have an active session.
// It is nil-safe: if s is nil the call is a no-op.
func (s *NotificationServer) Send(ctx context.Context, q *db.Queries, recipientID, senderID uuid.UUID, notifType db.NotificationType, message string) error {
	if s == nil {
		return nil
	}

	notification, err := q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:   recipientID,
		SenderID: uuid.NullUUID{UUID: senderID, Valid: true},
		Type:     notifType,
		Message:  message,
	})
	if err != nil {
		return err
	}

	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Live push — non-fatal if the recipient is offline.
	s.publishIfOnline(recipientID, notificationBytes)
	return nil
}

// MarkNotificationRead implements notification.NotificationServer.
// It marks a single notification as read (the only supported status).
func (s *NotificationServer) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.MarkNotificationReadResponse, error) {
	id, err := strconv.ParseInt(req.GetId(), 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notification id %q: %v", req.GetId(), err)
	}

	q := db.New(s.sqlDB)
	if _, err := q.MarkNotificationAsRead(ctx, id); err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "notification %d not found", id)
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "MarkNotificationRead: %v", err)
	}

	return &pb.MarkNotificationReadResponse{}, nil
}

// publishIfOnline looks up active notification sessions for userID and publishes
// notificationBytes to each session's listen_path via NATS.
// All errors are logged and swallowed — an offline user is normal.
func (s *NotificationServer) publishIfOnline(userID uuid.UUID, notificationBytes []byte) {
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
		return
	}

	for _, session := range sessions {
		subject := session.ListenPath.String
		if err := s.publisher.Publish(subject, notificationBytes); err != nil {
			lib.ErrorLog.Printf("publishIfOnline: publish to %s: %v", subject, err)
			continue
		}
	}
}
