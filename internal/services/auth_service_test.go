package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zukigit/chat/internal/db"
	"github.com/zukigit/chat/internal/services"
	"github.com/zukigit/chat/proto/auth"
	"golang.org/x/crypto/bcrypt"
)

func schemaPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "sqls", "init", "schema.sql")
}

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
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      schemaPath(),
				ContainerFilePath: "/docker-entrypoint-initdb.d/schema.sql",
				FileMode:          0755,
			},
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

	return sqlDB
}

func TestLogin(t *testing.T) {
	sqlDb := setupTestDB(t)
	q := db.New(sqlDb)

	hashPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	_, err = q.CreateUser(t.Context(), db.CreateUserParams{
		UserName:     "test",
		HashedPasswd: string(hashPassword),
		SignupType:   db.SignupTypeEmail,
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
		{"user not found", "unknown", "password", true},
	}

	authServer := services.NewAuthServer(sqlDb)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = authServer.Login(t.Context(), &auth.LoginRequest{
				UserName: tt.username,
				Passwd:   tt.password,
			})
			if err != nil {
				if !tt.wantErr {
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
			if err != nil {
				if !tt.wantErr {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
