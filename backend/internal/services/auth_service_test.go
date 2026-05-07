package services_test

import (
	"database/sql"
	"testing"

	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/services"
	"github.com/zukigit/chat/backend/proto/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin(t *testing.T) {
	sqlDb := setupTestDB(t)
	q := db.New(sqlDb)

	hashPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	_, err = q.CreateUser(t.Context(), db.CreateUserParams{
		UserName: "test",
		HashedPasswd: sql.NullString{
			String: string(hashPassword),
			Valid:  true,
		},
		SignupType: db.SignupTypeEmail,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{"valid credentials", "test", "password", false},
		{"invalid password", "test", "wrongpassword", true},
		{"non-existent user", "nonexistent", "password", true},
		{"empty username", "", "password", true},
		{"empty password", "test", "", true},
	}

	authServer := services.NewAuthServer(sqlDb)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = authServer.Login(t.Context(), &auth.LoginRequest{
				UserName: tt.username,
				Passwd:   tt.password,
			})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSignup(t *testing.T) {
	sqlDb := setupTestDB(t)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{"user1", "user1", "password", false},
		{"user2", "user2", "password", false},
		{"empty username", "", "password", true},
		{"empty password", "test", "", true},
		{"duplicate username", "user1", "password", true},
	}

	authServer := services.NewAuthServer(sqlDb)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authServer.Signup(t.Context(), &auth.SignupRequest{
				UserName: tt.username,
				Passwd:   tt.password,
			})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSearchUsers(t *testing.T) {
	sqlDb := setupTestDB(t)
	q := db.New(sqlDb)

	hashPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	users := []db.CreateUserParams{
		{UserName: "zuki", HashedPasswd: sql.NullString{String: string(hashPassword), Valid: true}, SignupType: db.SignupTypeEmail},
		{UserName: "zuki_sama", HashedPasswd: sql.NullString{String: string(hashPassword), Valid: true}, SignupType: db.SignupTypeEmail},
		{UserName: "alice", HashedPasswd: sql.NullString{String: string(hashPassword), Valid: true}, SignupType: db.SignupTypeEmail},
		{UserName: "bob", HashedPasswd: sql.NullString{String: string(hashPassword), Valid: true}, SignupType: db.SignupTypeEmail},
	}

	for _, u := range users {
		if _, err := q.CreateUser(t.Context(), u); err != nil {
			t.Fatalf("failed to create user %q: %v", u.UserName, err)
		}
	}

	// Create a friendship between zuki and alice (accepted).
	zuki, err := q.GetUserByUsername(t.Context(), "zuki")
	if err != nil {
		t.Fatalf("failed to fetch zuki: %v", err)
	}
	alice, err := q.GetUserByUsername(t.Context(), "alice")
	if err != nil {
		t.Fatalf("failed to fetch alice: %v", err)
	}
	// Ensure canonical ordering (smaller UUID first).
	first, second := zuki.UserID, alice.UserID
	if first.String() > second.String() {
		first, second = second, first
	}
	_, err = q.SendFriendRequest(t.Context(), db.SendFriendRequestParams{
		User1Userid:     first,
		User2Userid:     second,
		InitiatorUserid: zuki.UserID,
	})
	if err != nil {
		t.Fatalf("failed to create friendship: %v", err)
	}
	_, err = q.UpdateFriendshipStatus(t.Context(), db.UpdateFriendshipStatusParams{
		User1Userid: first,
		User2Userid: second,
		Status:      db.FriendshipStatusAccepted,
	})
	if err != nil {
		t.Fatalf("failed to update friendship status: %v", err)
	}

	authServer := services.NewAuthServer(sqlDb)

	// SearchUsers returns all matching users (including friends), with
	// friendship_status populated for existing friendships. Use bob as the caller.
	caller, err := q.GetUserByUsername(t.Context(), "bob")
	if err != nil {
		t.Fatalf("failed to fetch caller user: %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantErr   bool
	}{
		{"exact match username", "zuki", 2, false},
		{"partial match username", "zuk", 2, false},
		{"case insensitive", "ZUKI", 2, false},
		{"no results", "xyz123", 0, false},
		{"empty query", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ctxWithUser(caller.UserName, caller.UserID)
			resp, err := authServer.SearchUsers(ctx, &auth.SearchUsersRequest{
				Query: tt.query,
			})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resp.Users) != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, len(resp.Users))
			}
		})
	}

	// Verify friendship_status is populated correctly.
	// Search as zuki — should see alice with friendship_status="accepted".
	zukiCtx := ctxWithUser(zuki.UserName, zuki.UserID)
	resp, err := authServer.SearchUsers(zukiCtx, &auth.SearchUsersRequest{Query: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Users))
	}
	if resp.Users[0].FriendshipStatus != "accepted" {
		t.Errorf("expected friendship_status=%q, got %q", "accepted", resp.Users[0].FriendshipStatus)
	}

	// Search as bob — should see alice with empty friendship_status.
	bobCtx := ctxWithUser(caller.UserName, caller.UserID)
	resp, err = authServer.SearchUsers(bobCtx, &auth.SearchUsersRequest{Query: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Users))
	}
	if resp.Users[0].FriendshipStatus != "" {
		t.Errorf("expected empty friendship_status, got %q", resp.Users[0].FriendshipStatus)
	}
}
