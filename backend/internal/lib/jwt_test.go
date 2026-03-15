package lib_test

import (
	"testing"

	"github.com/zukigit/chat/backend/internal/lib"
)

const testSecret = "test-secret-123"

// setSecret sets JWT_SECRET for the duration of the test and restores it after.
func setSecret(t *testing.T, secret string) {
	t.Helper()
	t.Setenv("JWT_SECRET", secret)
}

// generateToken generates a token for username, failing the test on error.
func generateToken(t *testing.T, username string) string {
	t.Helper()
	token, err := lib.GenerateToken("00000000-0000-0000-0000-000000000001", username)
	if err != nil {
		t.Fatalf("GenerateToken(%q): %v", username, err)
	}
	return token
}

func TestGenerateAndValidateToken(t *testing.T) {
	setSecret(t, testSecret)

	const username = "alice"
	tokenStr := generateToken(t, username)
	if tokenStr == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	claims, err := lib.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Username != username {
		t.Errorf("claims.Username = %q, want %q", claims.Username, username)
	}
}

func TestValidateToken_InvalidInputs(t *testing.T) {
	setSecret(t, testSecret)

	cases := []struct {
		name  string
		token string
	}{
		{"malformed", "this.is.not.a.jwt"},
		{"empty", ""},
		{"random_string", "notajwtatall"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := lib.ValidateToken(tc.token)
			if err == nil {
				t.Errorf("ValidateToken(%q): expected error, got nil", tc.token)
			}
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	setSecret(t, "secret-a")
	tokenStr := generateToken(t, "carol")

	// Validate with a different secret — should be rejected.
	setSecret(t, "secret-b")
	if _, err := lib.ValidateToken(tokenStr); err == nil {
		t.Fatal("expected error when using wrong secret, got nil")
	}
}
