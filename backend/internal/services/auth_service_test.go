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

	authServer := services.NewAuthServer(sqlDb)

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
			resp, err := authServer.SearchUsers(t.Context(), &auth.SearchUsersRequest{
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
}
