package services_test

import (
	"context"
	"testing"

	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/session"
	"google.golang.org/grpc/codes"
)

func TestGetListenPath(t *testing.T) {
	sessionServer := services.NewSessionServer()

	cases := []struct {
		name    string
		userID  string
		loginID string
		reqType string
		wantErr codes.Code
		want    string
	}{
		{
			name:    "chat type",
			userID:  "user-1",
			loginID: "login-1",
			reqType: "chat",
			wantErr: codes.OK,
			want:    lib.ChatSubjectPrefix + "user-1",
		},
		{
			name:    "notification type",
			userID:  "user-2",
			loginID: "login-2",
			reqType: "notification",
			wantErr: codes.OK,
			want:    lib.NotiSubjectPrefix + "user-2",
		},
		{
			name:    "unknown type",
			userID:  "user-3",
			loginID: "login-3",
			reqType: "unknown",
			wantErr: codes.InvalidArgument,
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), lib.ContextKeyUserID, tc.userID)
			ctx = context.WithValue(ctx, lib.ContextKeyLoginID, tc.loginID)

			resp, err := sessionServer.GetListenPath(ctx, &pb.GetListenPathRequest{
				Type: tc.reqType,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
			if tc.wantErr == codes.OK && resp.GetListenPath() != tc.want {
				t.Errorf("listen_path: got %q, want %q", resp.GetListenPath(), tc.want)
			}
		})
	}
}
