package services

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"context"

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
// and pushes a live NATS event to the recipient.
// It is nil-safe: if s is nil the call is a no-op.
func (s *NotificationServer) Send(ctx context.Context, q *db.Queries, notiParams db.CreateNotificationParams) error {
	if s == nil {
		return nil
	}

	notification, err := q.CreateNotification(ctx, notiParams)
	if err != nil {
		return err
	}

	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	s.publishIfOnline(notiParams.UserID, lib.NotiSubjectPrefix, notificationBytes)
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

// publishIfOnline publishes payload to the per-user NATS subject.
// JetStream retains the message so any active durable consumer (device) receives it.
// All errors are logged and swallowed — delivery via JetStream handles retries.
func (s *NotificationServer) publishIfOnline(userID uuid.UUID, subjectPrefix string, payload []byte) {
	if s.publisher == nil {
		return
	}
	subject := subjectPrefix + userID.String()
	s.publisher.Publish(subject, payload) //nolint:errcheck
}
