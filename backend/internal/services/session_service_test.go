package services_test

import (
	"context"
	"testing"

	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc/codes"
)

func TestAddSession(t *testing.T) {
	sqlDB := setupTestDB(t)
	sessionServer := services.NewSessionServer(sqlDB)

	ids := createTestUsers(t, sqlDB, "alice")

	cases := []struct {
		name    string
		ctx     context.Context
		reqType string
		wantErr codes.Code
	}{
		{"valid chat session", ctxWithUser("alice", ids["alice"]), "chat", codes.OK},
		{"valid notification session", ctxWithUser("alice", ids["alice"]), "notification", codes.OK},
		{"empty type", ctxWithUser("alice", ids["alice"]), "", codes.InvalidArgument},
		{"no auth context", context.Background(), "chat", codes.Unauthenticated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sessionServer.AddSession(tc.ctx, &pb.AddSessionRequest{Type: tc.reqType})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestSetSessionStatus(t *testing.T) {
	sqlDB := setupTestDB(t)
	sessionServer := services.NewSessionServer(sqlDB)

	ids := createTestUsers(t, sqlDB, "alice")
	ctx := ctxWithUser("alice", ids["alice"])

	// Create a session so SetSessionStatus has a real record to update.
	resp, err := sessionServer.AddSession(ctx, &pb.AddSessionRequest{Type: "chat"})
	if err != nil {
		t.Fatalf("setup AddSession: %v", err)
	}
	sessionID := resp.SessionId

	cases := []struct {
		name      string
		ctx       context.Context
		sessionID string
		reqStatus string
		wantErr   codes.Code
	}{
		{"empty status", ctx, sessionID, "", codes.InvalidArgument},
		{"empty session ID", ctx, "", "active", codes.InvalidArgument},
		{"no auth context", context.Background(), sessionID, "active", codes.OK},
		{"no session id", ctx, "invalid-uuid", "active", codes.InvalidArgument},
		{"valid active status", ctx, sessionID, "active", codes.OK},
		{"valid idle status", ctx, sessionID, "idel", codes.OK},
		{"valid terminate status", ctx, sessionID, "terminate", codes.OK},
		{"valid new status", ctx, sessionID, "new", codes.OK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sessionServer.SetSessionStatus(tc.ctx, &pb.SetSessionStatusRequest{
				SessionId: tc.sessionID,
				Status:    tc.reqStatus,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}
