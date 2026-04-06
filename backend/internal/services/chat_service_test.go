package services_test

import (
	"context"
	"testing"

	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/chat"
	"google.golang.org/grpc/codes"
)

func TestCreateConversation_DM(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")

	cases := []struct {
		name      string
		ctx       context.Context
		membersID []string
		wantErr   codes.Code
	}{
		{"valid", ctxWithUser("alice", ids["alice"]), []string{ids["bob"].String()}, codes.OK},
		{"idempotent", ctxWithUser("alice", ids["alice"]), []string{ids["bob"].String()}, codes.OK},
		{"no auth", context.Background(), []string{ids["bob"].String()}, codes.Internal},
		{"empty members", ctxWithUser("alice", ids["alice"]), []string{}, codes.InvalidArgument},
		{"too many members", ctxWithUser("alice", ids["alice"]), []string{ids["bob"].String(), ids["carol"].String()}, codes.InvalidArgument},
		{"invalid uuid", ctxWithUser("alice", ids["alice"]), []string{"not-a-uuid"}, codes.InvalidArgument},
		{"self dm", ctxWithUser("alice", ids["alice"]), []string{ids["alice"].String()}, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chatServer.CreateConversation(tc.ctx, &pb.CreateConversationRequest{
				IsGroup:   false,
				MembersId: tc.membersID,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestCreateConversation_Group(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	cases := []struct {
		name      string
		ctx       context.Context
		groupName string
		membersID []string
		wantErr   codes.Code
	}{
		{"valid", ctxWithUser("alice", ids["alice"]), "team-chat", []string{ids["bob"].String()}, codes.OK},
		{"no auth", context.Background(), "team-chat", []string{ids["bob"].String()}, codes.Internal},
		{"missing name", ctxWithUser("alice", ids["alice"]), "", []string{ids["bob"].String()}, codes.InvalidArgument},
		{"invalid uuid", ctxWithUser("alice", ids["alice"]), "team-chat", []string{"not-a-uuid"}, codes.InvalidArgument},
		{"empty members", ctxWithUser("alice", ids["alice"]), "solo", []string{}, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chatServer.CreateConversation(tc.ctx, &pb.CreateConversationRequest{
				IsGroup:   true,
				Name:      tc.groupName,
				MembersId: tc.membersID,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}
