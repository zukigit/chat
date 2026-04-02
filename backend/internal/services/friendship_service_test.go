package services_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc/codes"
)

func TestSendFriendRequest(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB, nil)

	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")

	cases := []struct {
		name    string
		caller  string
		target  string
		wantErr codes.Code
	}{
		{"valid", "alice", "bob", codes.OK},
		{"duplicate", "alice", "bob", codes.AlreadyExists},
		{"reverse duplicate", "bob", "alice", codes.AlreadyExists},
		{"self request", "alice", "alice", codes.InvalidArgument},
		{"empty target", "alice", "", codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ctxWithUser(tc.caller, ids[tc.caller])
			_, err := friendshipServer.SendFriendRequest(ctx, &pb.FriendRequest{TargetUsername: tc.target})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestAcceptFriendRequest(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB, nil)

	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice sends bob a friend request.
	_, err := friendshipServer.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
	if err != nil {
		t.Fatalf("setup: SendFriendRequest: %v", err)
	}

	cases := []struct {
		name    string
		caller  string // who tries to accept
		target  string // whose request they're accepting
		wantErr codes.Code
	}{
		{"requester cannot accept own request", "alice", "bob", codes.PermissionDenied},
		{"addressee accepts", "bob", "alice", codes.OK},
		{"already accepted", "bob", "alice", codes.FailedPrecondition},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ctxWithUser(tc.caller, ids[tc.caller])
			_, err := friendshipServer.AcceptFriendRequest(ctx, &pb.FriendRequest{TargetUsername: tc.target})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestRejectFriendRequest(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB, nil)

	ids := createTestUsers(t, sqlDB, "alice", "bob")

	_, err := friendshipServer.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
	if err != nil {
		t.Fatalf("setup: SendFriendRequest: %v", err)
	}

	cases := []struct {
		name    string
		caller  string
		target  string
		wantErr codes.Code
	}{
		{"requester cannot reject own request", "alice", "bob", codes.PermissionDenied},
		{"addressee rejects", "bob", "alice", codes.OK},
		{"already rejected", "bob", "alice", codes.NotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ctxWithUser(tc.caller, ids[tc.caller])
			_, err := friendshipServer.RejectFriendRequest(ctx, &pb.FriendRequest{TargetUsername: tc.target})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

// TestFriendshipStateTransitions consolidates state-machine edge cases:
// what happens when you send/accept/reject after the relationship is already
// in a terminal or intermediate state.
func TestFriendshipStateTransitions(t *testing.T) {
	type setupStep func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID)
	type finalStep func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code

	mustSend := func(caller, target string) setupStep {
		return func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) {
			t.Helper()
			if _, err := fs.SendFriendRequest(ctxWithUser(caller, ids[caller]), &pb.FriendRequest{TargetUsername: target}); err != nil {
				t.Fatalf("setup SendFriendRequest(%s→%s): %v", caller, target, err)
			}
		}
	}
	mustAccept := func(caller, target string) setupStep {
		return func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) {
			t.Helper()
			if _, err := fs.AcceptFriendRequest(ctxWithUser(caller, ids[caller]), &pb.FriendRequest{TargetUsername: target}); err != nil {
				t.Fatalf("setup AcceptFriendRequest(%s→%s): %v", caller, target, err)
			}
		}
	}
	mustReject := func(caller, target string) setupStep {
		return func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) {
			t.Helper()
			if _, err := fs.RejectFriendRequest(ctxWithUser(caller, ids[caller]), &pb.FriendRequest{TargetUsername: target}); err != nil {
				t.Fatalf("setup RejectFriendRequest(%s→%s): %v", caller, target, err)
			}
		}
	}

	cases := []struct {
		name    string
		setup   []setupStep
		final   finalStep
		wantErr codes.Code
	}{
		{
			name: "accept when no request exists",
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
				return grpcCode(err)
			},
			wantErr: codes.NotFound,
		},
		{
			name:  "send after rejection returns pending status",
			setup: []setupStep{mustSend("alice", "bob"), mustReject("bob", "alice")},
			final: func(t *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				resp, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
				if err == nil && resp.Status != "pending" {
					t.Errorf("expected status 'pending' after re-send, got %q", resp.Status)
				}
				return grpcCode(err)
			},
			wantErr: codes.OK,
		},
		{
			name:  "accept after rejection",
			setup: []setupStep{mustSend("alice", "bob"), mustReject("bob", "alice")},
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
				return grpcCode(err)
			},
			wantErr: codes.NotFound,
		},
		{
			name:  "accept after acceptance",
			setup: []setupStep{mustSend("alice", "bob"), mustAccept("bob", "alice")},
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
				return grpcCode(err)
			},
			wantErr: codes.FailedPrecondition,
		},
		{
			name:  "reject after rejection",
			setup: []setupStep{mustSend("alice", "bob"), mustReject("bob", "alice")},
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
				return grpcCode(err)
			},
			wantErr: codes.NotFound,
		},
		{
			name:  "reject after acceptance",
			setup: []setupStep{mustSend("alice", "bob"), mustAccept("bob", "alice")},
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
				return grpcCode(err)
			},
			wantErr: codes.FailedPrecondition,
		},
		{
			name:  "send after acceptance",
			setup: []setupStep{mustSend("alice", "bob"), mustAccept("bob", "alice")},
			final: func(_ *testing.T, fs *services.FriendshipServer, ids map[string]uuid.UUID) codes.Code {
				_, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
				return grpcCode(err)
			},
			wantErr: codes.AlreadyExists,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := setupTestDB(t)
			fs := services.NewFriendshipServer(sqlDB, nil)
			ids := createTestUsers(t, sqlDB, "alice", "bob")
			for _, step := range tc.setup {
				step(t, fs, ids)
			}
			if got := tc.final(t, fs, ids); got != tc.wantErr {
				t.Errorf("got %v, want %v", got, tc.wantErr)
			}
		})
	}
}
