package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zukigit/chat/backend/internal/handlers"
	pb "github.com/zukigit/chat/backend/proto/session"
)

type mockSessionClient struct {
	pb.SessionClient
	AddSessionFunc       func(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error)
	SetSessionStatusFunc func(ctx context.Context, token, sessionID, status string) error
}

func (m *mockSessionClient) AddSession(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error) {
	if m.AddSessionFunc != nil {
		return m.AddSessionFunc(ctx, token, sessionType)
	}
	return &pb.AddSessionResponse{
		SessionId:  "mock-session-id",
		ListenPath: "NOTIFICATIONS.mock-session-id",
	}, nil
}

func (m *mockSessionClient) SetSessionStatus(ctx context.Context, token, sessionID, status string) error {
	if m.SetSessionStatusFunc != nil {
		return m.SetSessionStatusFunc(ctx, token, sessionID, status)
	}
	return nil
}

func setupNatsContainer(ctx context.Context, t *testing.T) (*nats.Conn, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.9",
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForListeningPort("4222/tcp"),
	}

	natsC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := natsC.Host(ctx)
	require.NoError(t, err)

	port, err := natsC.MappedPort(ctx, "4222")
	require.NoError(t, err)

	natsURL := "nats://" + host + ":" + port.Port()
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)

	cleanup := func() {
		nc.Close()
		natsC.Terminate(ctx)
	}

	return nc, cleanup
}

func TestNotificationSession(t *testing.T) {
	ctx := context.Background()
	nc, cleanup := setupNatsContainer(ctx, t)
	defer cleanup()

	mockClient := &mockSessionClient{
		AddSessionFunc: func(ctx context.Context, token, sessionType string) (*pb.AddSessionResponse, error) {
			require.Equal(t, "mock-token", token)
			require.Equal(t, "NOTIFICATION", sessionType)
			return &pb.AddSessionResponse{
				SessionId:  "123-456",
				ListenPath: "NOTIFICATIONS.123-456",
			}, nil
		},
	}

	handler := handlers.NewSessionHandler(mockClient, nc)

	server := httptest.NewServer(http.HandlerFunc(handler.NotificationSession))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "?token=mock-token"

	dialer := websocket.Dialer{}
	wsConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err, "could not establish websocket connection")
	defer wsConn.Close()

	// Wait briefly for the server to subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish to the test NATS server
	err = nc.Publish("NOTIFICATIONS.123-456", []byte(`{"type":"new_message", "content":"hello server"}`))
	require.NoError(t, err)

	// Read from WS client
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, msg, err := wsConn.ReadMessage()
	require.NoError(t, err, "could not read message from websocket")
	require.Equal(t, websocket.TextMessage, msgType)
	require.Equal(t, `{"type":"new_message", "content":"hello server"}`, string(msg))
}
