package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/notification"
	"google.golang.org/grpc/codes"
)

func TestMarkNotificationRead(t *testing.T) {
	sqlDB := setupTestDB(t)
	notificationServer := services.NewNotificationServer(sqlDB, nil)

	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// Create a notification from alice to bob for use in test cases.
	notification, err := db.New(sqlDB).CreateNotification(context.Background(), db.CreateNotificationParams{
		UserID:   ids["bob"],
		SenderID: ids["alice"],
		Type:     db.NotificationTypeFriendRequest,
		Message:  "alice sent you a friend request",
	})
	if err != nil {
		t.Fatalf("setup: CreateNotification: %v", err)
	}

	cases := []struct {
		name    string
		id      string
		wantErr codes.Code
	}{
		{"valid", fmt.Sprintf("%d", notification.ID), codes.OK},
		{"not found", "999999999", codes.NotFound},
		{"invalid id", "not-a-number", codes.InvalidArgument},
		{"empty id", "", codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := notificationServer.MarkNotificationRead(context.Background(), &pb.MarkNotificationReadRequest{Id: tc.id})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}
