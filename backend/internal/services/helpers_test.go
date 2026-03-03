package services_test

import (
	"database/sql"
	"testing"

	"github.com/zukigit/chat/backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// createTestUsers creates test users in the database for use in tests.
// It fails the test immediately if any user cannot be created.
func createTestUsers(t *testing.T, sqlDB *sql.DB, usernames ...string) {
	t.Helper()
	q := db.New(sqlDB)
	hashed, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("createTestUsers: hash password: %v", err)
	}
	for _, username := range usernames {
		if _, err := q.CreateUser(t.Context(), db.CreateUserParams{
			UserName:     username,
			HashedPasswd: string(hashed),
			SignupType:   db.SignupTypeEmail,
		}); err != nil {
			t.Fatalf("createTestUsers: create %q: %v", username, err)
		}
	}
}
