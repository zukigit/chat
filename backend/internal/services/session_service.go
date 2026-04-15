package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SessionServer struct {
	sqlDB *sql.DB
	session.UnimplementedSessionServer
}

func NewSessionServer(sqlDB *sql.DB) *SessionServer {
	if sqlDB == nil {
		return nil
	}
	return &SessionServer{sqlDB: sqlDB}
}

// ValidateSession checks that login_id exists in the sessions table and returns
// the owning user_id. The JWT itself is already verified by the gRPC interceptor.
func (s *SessionServer) ValidateSession(ctx context.Context, req *session.ValidateSessionRequest) (*session.ValidateSessionResponse, error) {
	loginID, err := uuid.Parse(req.GetLoginId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid login_id")
	}

	q := db.New(s.sqlDB)
	userID, err := q.ValidateSession(ctx, loginID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "session not found or expired")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ValidateSession: %v", err)
	}

	return &session.ValidateSessionResponse{UserId: userID.String()}, nil
}
