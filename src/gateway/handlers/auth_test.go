package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zukigit/chat/src/gateway/handlers"
	db "github.com/zukigit/chat/src/lib/db"
	"golang.org/x/crypto/bcrypt"
)

// schemaPath resolves sqls/schema.sql relative to this test file.
func schemaPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "sqls", "schema.sql")
}

// setupTestDB spins up a PostgreSQL container, runs the schema, and returns a ready *sql.DB.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "chat",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "chat",
		},
		// Postgres logs the ready message twice during initialisation.
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://chat:test@%s:%s/chat?sslmode=disable", host, port.Port())
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	schema, err := os.ReadFile(schemaPath())
	if err != nil {
		t.Fatalf("failed to read schema.sql: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, string(schema)); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	return sqlDB
}

func TestLoginHandler(t *testing.T) {
	sqlDB := setupTestDB(t)
	q := db.New(sqlDB)

	// Seed a test user with a known bcrypt-hashed password.
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	_, err = sqlDB.Exec(
		`INSERT INTO users (user_name, hashed_passwd, signup_type) VALUES ($1, $2, $3)`,
		"testuser", string(hash), "email",
	)
	if err != nil {
		t.Fatalf("failed to seed test user: %v", err)
	}

	handler := handlers.LoginHandler(q)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "valid credentials",
			body:       map[string]string{"username": "testuser", "password": "password123"},
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "wrong password",
			body:       map[string]string{"username": "testuser", "password": "wrongpass"},
			wantStatus: http.StatusUnauthorized,
			wantOK:     false,
		},
		{
			name:       "user not found",
			body:       map[string]string{"username": "nobody", "password": "password123"},
			wantStatus: http.StatusUnauthorized,
			wantOK:     false,
		},
		{
			name:       "missing username",
			body:       map[string]string{"username": "", "password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
		{
			name:       "missing password",
			body:       map[string]string{"username": "testuser", "password": ""},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			r := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}

			var resp struct {
				Success bool `json:"success"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Errorf("failed to decode response: %v", err)
			}
			if resp.Success != tt.wantOK {
				t.Errorf("success = %v, want %v", resp.Success, tt.wantOK)
			}
		})
	}
}
