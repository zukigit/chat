package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// schemaPath returns the absolute path to sqls/init/schema.sql.
func schemaPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "sqls", "init", "schema.sql")
}

// setupTestDB starts a throwaway Postgres container, mounts schema.sql via
// /docker-entrypoint-initdb.d so Postgres runs it automatically, and returns
// a connected *sql.DB. Container and DB are cleaned up when the test ends.
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

// createTestUsers inserts users via db.CreateUser and returns a map of
// username → user_id (UUID) for use in test contexts.
// Fails the test immediately if any user cannot be created.
func createTestUsers(t *testing.T, sqlDB *sql.DB, usernames ...string) map[string]uuid.UUID {
	t.Helper()
	q := db.New(sqlDB)
	hashed, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("createTestUsers: hash password: %v", err)
	}
	ids := make(map[string]uuid.UUID, len(usernames))
	for _, username := range usernames {
		u, err := q.CreateUser(t.Context(), db.CreateUserParams{
			UserName: username,
			HashedPasswd: sql.NullString{
				Valid:  true,
				String: string(hashed),
			},
			SignupType: db.SignupTypeEmail,
		})
		if err != nil {
			t.Fatalf("createTestUsers: create %q: %v", username, err)
		}
		ids[username] = u.UserID
	}
	return ids
}

// makeFriends creates an accepted friendship between two users directly via the DB.
func makeFriends(t *testing.T, sqlDB *sql.DB, id1, id2 uuid.UUID) {
	t.Helper()
	q := db.New(sqlDB)
	first, second := lib.OrderedUUIDPair(id1, id2)
	if _, err := q.SendFriendRequest(context.Background(), db.SendFriendRequestParams{
		User1Userid:     first,
		User2Userid:     second,
		InitiatorUserid: id1,
	}); err != nil {
		t.Fatalf("makeFriends: send request: %v", err)
	}
	if _, err := q.UpdateFriendshipStatus(context.Background(), db.UpdateFriendshipStatusParams{
		User1Userid: first,
		User2Userid: second,
		Status:      db.FriendshipStatusAccepted,
	}); err != nil {
		t.Fatalf("makeFriends: accept: %v", err)
	}
}

func ctxWithUser(username string, userID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), lib.ContextKeyUsername, username)
	ctx = context.WithValue(ctx, lib.ContextKeyUserID, userID.String())
	return ctx
}

func grpcCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	st, _ := status.FromError(err)
	return st.Code()
}

// setupTestNats starts a throwaway NATS container and returns a connected
// *nats.Conn. The container and connection are cleaned up when the test ends.
func setupTestNats(t *testing.T) *nats.Conn {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("setupTestNats: start container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("setupTestNats: get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatalf("setupTestNats: get port: %v", err)
	}

	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%s", host, port.Port()))
	if err != nil {
		t.Fatalf("setupTestNats: connect: %v", err)
	}
	t.Cleanup(nc.Close)

	return nc
}
