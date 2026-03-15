package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SessionServer struct {
	sqlDB *sql.DB
	session.UnimplementedSessionServer
}

func NewSessionServer(sqlDB *sql.DB) *SessionServer {
	if sqlDB != nil {
		return &SessionServer{
			sqlDB: sqlDB,
		}
	}
	return nil
}

func (s *SessionServer) AddSession(ctx context.Context, request *session.AddSessionRequest) (*session.AddSessionResponse, error) {
	if request.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "Type is required")
	}

	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Failed to get caller UUID")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	quries := db.New(tx)

	row, err := quries.CreateSession(ctx, db.CreateSessionParams{
		UserUserid: callerID,
		Type:       db.SessionType(request.Type),
		Status:     db.SessionStatusNew,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create session: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to commit transaction: %v", err)
	}

	return &session.AddSessionResponse{
		SessionId:  row.ID.String(),
		ListenPath: fmt.Sprintf("NOTIFICATIONS.%s", row.ID.String()),
	}, nil
}

func (s *SessionServer) SetSessionStatus(ctx context.Context, request *session.SetSessionStatusRequest) (*session.SetSessionStatusResponse, error) {
	if request.GetStatus() == "" {
		return nil, status.Error(codes.InvalidArgument, "Status is required")
	}

	sessionID, err := uuid.Parse(request.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid session ID")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	quries := db.New(tx)

	err = quries.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:     sessionID,
		Status: db.SessionStatus(request.Status),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update session status: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to commit transaction: %v", err)
	}

	return &session.SetSessionStatusResponse{}, nil
}
