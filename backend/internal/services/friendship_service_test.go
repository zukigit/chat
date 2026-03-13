package services_test

import (
	"testing"

	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/friendship"
	"google.golang.org/grpc/codes"
)

func TestSendFriendRequest(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB)

	// Create two users so foreign-key constraints pass.
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
	friendshipServer := services.NewFriendshipServer(sqlDB)

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
	friendshipServer := services.NewFriendshipServer(sqlDB)

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
		{"already rejected", "bob", "alice", codes.FailedPrecondition},
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

func TestAcceptFriendRequest_NotFound(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB)

	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// No request exists — try to accept.
	_, err := friendshipServer.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
	if got := grpcCode(err); got != codes.NotFound {
		t.Errorf("got %v, want NotFound", got)
	}
}

func TestSendFriendRequest_AfterRejection(t *testing.T) {
	sqlDB := setupTestDB(t)
	friendshipServer := services.NewFriendshipServer(sqlDB)

	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice sends bob a friend request.
	if _, err := friendshipServer.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("initial SendFriendRequest: %v", err)
	}

	// bob rejects it.
	if _, err := friendshipServer.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("RejectFriendRequest: %v", err)
	}

	// alice re-sends — should succeed now that it was rejected.
	resp, err := friendshipServer.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
	if got := grpcCode(err); got != codes.OK {
		t.Fatalf("re-send after rejection: got %v, want OK (err: %v)", got, err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending' after re-send, got %q", resp.Status)
	}
}

func TestAcceptFriendRequest_AfterRejection(t *testing.T) {
	sqlDB := setupTestDB(t)
	fs := services.NewFriendshipServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice → bob, bob rejects
	if _, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("setup SendFriendRequest: %v", err)
	}
	if _, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("setup RejectFriendRequest: %v", err)
	}

	// bob tries to accept the (now rejected) request — should fail
	_, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
	if got := grpcCode(err); got != codes.FailedPrecondition {
		t.Errorf("accept after rejection: got %v, want FailedPrecondition", got)
	}
}

func TestAcceptFriendRequest_AfterAcception(t *testing.T) {
	sqlDB := setupTestDB(t)
	fs := services.NewFriendshipServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice → bob, bob accepts
	if _, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("setup SendFriendRequest: %v", err)
	}
	if _, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("setup AcceptFriendRequest: %v", err)
	}

	// bob tries to accept again — should fail
	_, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
	if got := grpcCode(err); got != codes.FailedPrecondition {
		t.Errorf("accept after acceptance: got %v, want FailedPrecondition", got)
	}
}

func TestRejectFriendRequest_AfterRejection(t *testing.T) {
	sqlDB := setupTestDB(t)
	fs := services.NewFriendshipServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice → bob, bob rejects
	if _, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("setup SendFriendRequest: %v", err)
	}
	if _, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("setup RejectFriendRequest: %v", err)
	}

	// bob tries to reject again — should fail
	_, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
	if got := grpcCode(err); got != codes.FailedPrecondition {
		t.Errorf("reject after rejection: got %v, want FailedPrecondition", got)
	}
}

func TestRejectFriendRequest_AfterAcception(t *testing.T) {
	sqlDB := setupTestDB(t)
	fs := services.NewFriendshipServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice → bob, bob accepts
	if _, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("setup SendFriendRequest: %v", err)
	}
	if _, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("setup AcceptFriendRequest: %v", err)
	}

	// bob tries to reject an already-accepted friendship — should fail
	_, err := fs.RejectFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"})
	if got := grpcCode(err); got != codes.FailedPrecondition {
		t.Errorf("reject after acceptance: got %v, want FailedPrecondition", got)
	}
}

func TestSendFriendRequest_AfterAcception(t *testing.T) {
	sqlDB := setupTestDB(t)
	fs := services.NewFriendshipServer(sqlDB)
	ids := createTestUsers(t, sqlDB, "alice", "bob")

	// alice → bob, bob accepts
	if _, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"}); err != nil {
		t.Fatalf("setup SendFriendRequest: %v", err)
	}
	if _, err := fs.AcceptFriendRequest(ctxWithUser("bob", ids["bob"]), &pb.FriendRequest{TargetUsername: "alice"}); err != nil {
		t.Fatalf("setup AcceptFriendRequest: %v", err)
	}

	// alice tries to send another request to an existing friend — should fail
	_, err := fs.SendFriendRequest(ctxWithUser("alice", ids["alice"]), &pb.FriendRequest{TargetUsername: "bob"})
	if got := grpcCode(err); got != codes.AlreadyExists {
		t.Errorf("send after acceptance: got %v, want AlreadyExists", got)
	}
}
