package services

import (
	"context"
	"database/sql"

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

func (s *SessionServer) SetSession(ctx context.Context, request *session.SessionRequest) (*session.SessionResponse, error) {
	if request.GetListenPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "ListenPath is required")
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

	_, err = quries.CreateSession(ctx, db.CreateSessionParams{
		UserUserid: callerID,
		Type:       db.SessionType(request.Type),
		ListenPath: request.GetListenPath(),
		Status:     db.SessionStatusActive,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create session: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to commit transaction: %v", err)
	}

	return &session.SessionResponse{}, nil
}
