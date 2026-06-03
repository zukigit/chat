package services

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SessionServer struct {
	session.UnimplementedSessionServer
}

func NewSessionServer() *SessionServer {
	return &SessionServer{}
}

func (s *SessionServer) Ping(ctx context.Context, req *session.PingRequest) (*session.PingResponse, error) {
	return &session.PingResponse{Message: "pong"}, nil
}

// GetListenPath returns the NATS subject path for the caller based on their JWT claims.
// The JWT is already validated by the gRPC interceptor.
// Format: sessions.noti.<user_id> or sessions.chat.<user_id>
// The caller's login_id is used separately for the durable consumer name.
func (s *SessionServer) GetListenPath(ctx context.Context, req *session.GetListenPathRequest) (*session.GetListenPathResponse, error) {
	userID := lib.CallerIDFrom(ctx)
	loginID := lib.CallerLoginID(ctx)

	if userID == "" || loginID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user_id or login_id in token")
	}

	switch req.GetType() {
	case "chat":
		return &session.GetListenPathResponse{ListenPath: lib.ChatSubjectPrefix + userID, ConsumerName: "chat-" + loginID}, nil
	case "notification":
		return &session.GetListenPathResponse{ListenPath: lib.NotiSubjectPrefix + userID, ConsumerName: "noti-" + loginID}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown type: %q (must be 'chat' or 'notification')", req.GetType())
	}
}
