package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zukigit/chat/backend/internal/handlers"
	pb "github.com/zukigit/chat/backend/proto/session"
)

// ── Mock session client ───────────────────────────────────────────────────────

type mockSessionClient struct {
	addSessionFn       func(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error)
	setSessionStatusFn func(ctx context.Context, token, sessionID, status string) error
}

func (m *mockSessionClient) AddSession(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error) {
	return m.addSessionFn(ctx, token, sessionType)
}

func (m *mockSessionClient) SetSessionStatus(ctx context.Context, token, sessionID, status string) error {
	return m.setSessionStatusFn(ctx, token, sessionID, status)
}

// noopSetStatus is a helper that always succeeds.
func noopSetStatus(_ context.Context, _, _, _ string) error { return nil }

// ── NATS testcontainer helpers ────────────────────────────────────────────────

// setupNATS starts a throwaway NATS server with JetStream enabled and returns
// a connected nats.Conn and jetstream.JetStream. Both are cleaned up when the
// test finishes.
func setupNATS(t *testing.T) (*natsgo.Conn, jetstream.JetStream) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.12.4-alpine3.22",
		Cmd:          []string{"-js"}, // enable JetStream
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForLog("Server is ready"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start NATS container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get NATS container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatalf("failed to get NATS container port: %v", err)
	}

	natsURL := fmt.Sprintf("nats://%s:%s", host, port.Port())
	nc, err := natsgo.Connect(natsURL)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	t.Cleanup(func() { nc.Close() })

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream context: %v", err)
	}

	return nc, js
}

// createTestStream creates a JetStream stream with the given name and subjects.
func createTestStream(t *testing.T, js jetstream.JetStream, name string, subjects []string) jetstream.Stream {
	t.Helper()
	stream, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	})
	if err != nil {
		t.Fatalf("failed to create stream %q: %v", name, err)
	}
	return stream
}

// dialWS connects a WebSocket client to the given httptest.Server URL path.
func dialWS(t *testing.T, srv *httptest.Server, path, token string) (*websocket.Conn, *http.Response) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + path
	if token != "" {
		wsURL += "?token=" + token
	}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v (HTTP %v)", err, resp)
	}
	return conn, resp
}

// ── Unit tests (mock client, mock stream) ─────────────────────────────────────

// TestNotificationSession_MissingToken verifies that a request without a token
// is rejected before the WebSocket upgrade even begins.
func TestNotificationSession_MissingToken(t *testing.T) {
	// Use a nil stream — it must never be reached.
	h := handlers.NewSessionHandler(&mockSessionClient{
		addSessionFn:       func(_ context.Context, _, _ string) (*pb.AddSessionResponse, error) { return nil, nil },
		setSessionStatusFn: noopSetStatus,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/session/notification", nil)
	rec := httptest.NewRecorder()
	h.NotificationSession(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

// TestNotificationSession_AddSessionError verifies that a gRPC / service error
// from AddSession propagates to a 500 response before the upgrade.
func TestNotificationSession_AddSessionError(t *testing.T) {
	h := handlers.NewSessionHandler(&mockSessionClient{
		addSessionFn: func(_ context.Context, _, _ string) (*pb.AddSessionResponse, error) {
			return &pb.AddSessionResponse{}, fmt.Errorf("db exploded")
		},
		setSessionStatusFn: noopSetStatus,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/session/notification?token=mytoken", nil)
	rec := httptest.NewRecorder()
	h.NotificationSession(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// ── Integration tests (testcontainers NATS) ───────────────────────────────────

// TestNotificationSession_Integration_MessageForwarding verifies the end-to-end
// flow: WebSocket upgrade → NATS consumer creation → message forwarded to WS.
func TestNotificationSession_Integration_MessageForwarding(t *testing.T) {
	const (
		streamName = "NOTIFICATIONS"
		listenPath = "notifications.user.alice"
	)

	_, js := setupNATS(t)
	stream := createTestStream(t, js, streamName, []string{"notifications.>"})

	client := &mockSessionClient{
		addSessionFn: func(_ context.Context, _, _ string) (*pb.AddSessionResponse, error) {
			return &pb.AddSessionResponse{
				SessionId:  "sess-1",
				ListenPath: listenPath,
			}, nil
		},
		setSessionStatusFn: noopSetStatus,
	}

	h := handlers.NewSessionHandler(client, stream)
	srv := httptest.NewServer(http.HandlerFunc(h.NotificationSession))
	defer srv.Close()

	// Do NOT defer conn.Close() before the read — closing the WS would cancel
	// the handler's consumer context and race with the forwarding callback.
	conn, _ := dialWS(t, srv, "/session/notification", "test-token")

	// Give the ordered consumer goroutine a moment to attach.
	time.Sleep(150 * time.Millisecond)

	// Publish a message to NATS on the subject the consumer is watching.
	want := "hello from nats"
	if _, err := js.Publish(context.Background(), listenPath, []byte(want)); err != nil {
		conn.Close()
		t.Fatalf("failed to publish NATS message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, got, err := conn.ReadMessage()
	// Close the WS connection only after we have finished reading.
	conn.Close()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != want {
		t.Errorf("message: got %q, want %q", string(got), want)
	}
}

// TestNotificationSession_Integration_StatusLifecycle verifies that
// SetSessionStatus is called with "active" on connect and "terminate" on
// disconnect.
func TestNotificationSession_Integration_StatusLifecycle(t *testing.T) {
	const (
		streamName = "STATUS_LIFECYCLE"
		listenPath = "notifications.user.bob"
	)

	_, js := setupNATS(t)
	stream := createTestStream(t, js, streamName, []string{"notifications.>"})

	var (
		statusCalls []string
	)

	client := &mockSessionClient{
		addSessionFn: func(_ context.Context, _, _ string) (*pb.AddSessionResponse, error) {
			return &pb.AddSessionResponse{
				SessionId:  "sess-2",
				ListenPath: listenPath,
			}, nil
		},
		setSessionStatusFn: func(_ context.Context, _, _, status string) error {
			statusCalls = append(statusCalls, status)
			return nil
		},
	}

	h := handlers.NewSessionHandler(client, stream)
	srv := httptest.NewServer(http.HandlerFunc(h.NotificationSession))
	defer srv.Close()

	conn, _ := dialWS(t, srv, "/session/notification", "tok")

	// Give the handler time to call SetSessionStatus("active").
	time.Sleep(100 * time.Millisecond)

	// Close the client connection — the handler's goroutine detects this and
	// calls cancel(), which eventually triggers the deferred terminate call.
	conn.Close()

	// Allow the deferred cleanup to run.
	time.Sleep(200 * time.Millisecond)

	srv.Close()

	if len(statusCalls) < 2 {
		t.Fatalf("expected at least 2 SetSessionStatus calls, got %d: %v", len(statusCalls), statusCalls)
	}
	if statusCalls[0] != "active" {
		t.Errorf("first SetSessionStatus: want %q, got %q", "active", statusCalls[0])
	}
	// The deferred terminate call must appear.
	found := false
	for _, s := range statusCalls {
		if s == "terminate" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a 'terminate' SetSessionStatus call, got: %v", statusCalls)
	}
}

// TestNotificationSession_Integration_TokenFromQueryParam verifies the handler
// also accepts a token from the "token" query parameter (no Bearer header),
// which is the typical WebSocket authentication flow.
func TestNotificationSession_Integration_TokenFromQueryParam(t *testing.T) {
	const (
		streamName = "QUERY_PARAM"
		listenPath = "notifications.user.carol"
	)

	_, js := setupNATS(t)
	stream := createTestStream(t, js, streamName, []string{"notifications.>"})

	addCalled := false
	client := &mockSessionClient{
		addSessionFn: func(_ context.Context, token, _ string) (*pb.AddSessionResponse, error) {
			if token != "querytoken" {
				return nil, fmt.Errorf("unexpected token %q", token)
			}
			addCalled = true
			return &pb.AddSessionResponse{
				SessionId:  "sess-3",
				ListenPath: listenPath,
			}, nil
		},
		setSessionStatusFn: noopSetStatus,
	}

	h := handlers.NewSessionHandler(client, stream)
	srv := httptest.NewServer(http.HandlerFunc(h.NotificationSession))
	defer srv.Close()

	conn, _ := dialWS(t, srv, "/session/notification", "querytoken")
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	if !addCalled {
		t.Error("AddSession was not called — query-param token was not forwarded")
	}
}
