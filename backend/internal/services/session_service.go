package services

import (
	"context"
	"database/sql"

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
	if request.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "SessionId is required")
	}

	_, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to begin transaction: %v", err)
	}

	return nil, nil
}
