package lib_test

import (
	"testing"

	"github.com/zukigit/chat/backend/internal/lib"
)

func TestGenerateAndValidateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-123")

	username := "alice"
	tokenStr, err := lib.GenerateToken(username)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	claims, err := lib.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if claims.Username != username {
		t.Errorf("claims.Username = %q, want %q", claims.Username, username)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-123")

	_, err := lib.ValidateToken("this.is.not.a.jwt")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-123")

	tokenStr, err := lib.GenerateToken("bob")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Flip the last character to tamper with the signature.
	tampered := tokenStr[:len(tokenStr)-1] + "X"
	_, err = lib.ValidateToken(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-a")
	tokenStr, err := lib.GenerateToken("carol")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Validate with a different secret.
	t.Setenv("JWT_SECRET", "secret-b")
	_, err = lib.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error when using wrong secret, got nil")
	}
}
