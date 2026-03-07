package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/zukigit/chat/backend/internal/handlers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── Mock client ──────────────────────────────────────────────────────────────

// mockFriendshipClient implements the unexported friendshipClientInterface.
type mockFriendshipClient struct {
	sendFn   func(ctx context.Context, token, target string) error
	acceptFn func(ctx context.Context, token, target string) error
	rejectFn func(ctx context.Context, token, target string) error
}

func (m *mockFriendshipClient) SendFriendRequest(ctx context.Context, token, target string) error {
	return m.sendFn(ctx, token, target)
}

func (m *mockFriendshipClient) AcceptFriendRequest(ctx context.Context, token, target string) error {
	return m.acceptFn(ctx, token, target)
}

func (m *mockFriendshipClient) RejectFriendRequest(ctx context.Context, token, target string) error {
	return m.rejectFn(ctx, token, target)
}

// ── SendFriendRequest ────────────────────────────────────────────────────────

func TestSendFriendRequestHandler(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		body        any
		fn          func(context.Context, string, string) error
		wantCode    int
		wantSuccess bool
	}{
		{
			name:        "success returns 200",
			token:       "tok",
			body:        map[string]string{"username": "bob"},
			fn:          func(_ context.Context, _, _ string) error { return nil },
			wantCode:    http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "missing Authorization header returns 401",
			token:       "",
			body:        map[string]string{"username": "bob"},
			fn:          noop,
			wantCode:    http.StatusUnauthorized,
			wantSuccess: false,
		},
		{
			name:        "empty username in body returns 400",
			token:       "tok",
			body:        map[string]string{},
			fn:          noop,
			wantCode:    http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:  "duplicate request returns 409",
			token: "tok",
			body:  map[string]string{"username": "bob"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.AlreadyExists, "already exists")
			},
			wantCode:    http.StatusConflict,
			wantSuccess: false,
		},
		{
			name:  "user not found returns 404",
			token: "tok",
			body:  map[string]string{"username": "ghost"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.NotFound, "not found")
			},
			wantCode:    http.StatusNotFound,
			wantSuccess: false,
		},
		{
			name:  "self-request returns 400",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.InvalidArgument, "cannot send to yourself")
			},
			wantCode:    http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:  "expired token returns 401",
			token: "expired",
			body:  map[string]string{"username": "bob"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.Unauthenticated, "token expired")
			},
			wantCode:    http.StatusUnauthorized,
			wantSuccess: false,
		},
		{
			name:  "internal error returns 500",
			token: "tok",
			body:  map[string]string{"username": "bob"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.Internal, "db error")
			},
			wantCode:    http.StatusInternalServerError,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewFriendshipHandler(&mockFriendshipClient{
				sendFn: tt.fn, acceptFn: noop, rejectFn: noop,
			})
			code, ok := run(t, h.SendFriendRequest, friendshipReq(t, "/friends/request", tt.token, tt.body))
			if code != tt.wantCode {
				t.Errorf("status: got %d, want %d", code, tt.wantCode)
			}
			if ok != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", ok, tt.wantSuccess)
			}
		})
	}
}

// ── AcceptFriendRequest ──────────────────────────────────────────────────────

func TestAcceptFriendRequestHandler(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		body        any
		fn          func(context.Context, string, string) error
		wantCode    int
		wantSuccess bool
	}{
		{
			name:        "success returns 200",
			token:       "tok",
			body:        map[string]string{"username": "alice"},
			fn:          func(_ context.Context, _, _ string) error { return nil },
			wantCode:    http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "missing Authorization header returns 401",
			token:       "",
			body:        map[string]string{"username": "alice"},
			fn:          noop,
			wantCode:    http.StatusUnauthorized,
			wantSuccess: false,
		},
		{
			name:  "not pending (already accepted) returns 409",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.FailedPrecondition, "not pending")
			},
			wantCode:    http.StatusConflict,
			wantSuccess: false,
		},
		{
			name:  "permission denied returns 403",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.PermissionDenied, "not the addressee")
			},
			wantCode:    http.StatusForbidden,
			wantSuccess: false,
		},
		{
			name:  "request not found returns 404",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.NotFound, "not found")
			},
			wantCode:    http.StatusNotFound,
			wantSuccess: false,
		},
		{
			name:  "internal error returns 500",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.Internal, "db error")
			},
			wantCode:    http.StatusInternalServerError,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewFriendshipHandler(&mockFriendshipClient{
				sendFn: noop, acceptFn: tt.fn, rejectFn: noop,
			})
			code, ok := run(t, h.AcceptFriendRequest, friendshipReq(t, "/friends/accept", tt.token, tt.body))
			if code != tt.wantCode {
				t.Errorf("status: got %d, want %d", code, tt.wantCode)
			}
			if ok != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", ok, tt.wantSuccess)
			}
		})
	}
}

// ── RejectFriendRequest ──────────────────────────────────────────────────────

func TestRejectFriendRequestHandler(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		body        any
		fn          func(context.Context, string, string) error
		wantCode    int
		wantSuccess bool
	}{
		{
			name:        "success returns 200",
			token:       "tok",
			body:        map[string]string{"username": "alice"},
			fn:          func(_ context.Context, _, _ string) error { return nil },
			wantCode:    http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "missing Authorization header returns 401",
			token:       "",
			body:        map[string]string{"username": "alice"},
			fn:          noop,
			wantCode:    http.StatusUnauthorized,
			wantSuccess: false,
		},
		{
			name:  "not pending (already rejected) returns 409",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.FailedPrecondition, "not pending")
			},
			wantCode:    http.StatusConflict,
			wantSuccess: false,
		},
		{
			name:  "permission denied returns 403",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.PermissionDenied, "not the addressee")
			},
			wantCode:    http.StatusForbidden,
			wantSuccess: false,
		},
		{
			name:  "request not found returns 404",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.NotFound, "not found")
			},
			wantCode:    http.StatusNotFound,
			wantSuccess: false,
		},
		{
			name:  "internal error returns 500",
			token: "tok",
			body:  map[string]string{"username": "alice"},
			fn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.Internal, "db error")
			},
			wantCode:    http.StatusInternalServerError,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewFriendshipHandler(&mockFriendshipClient{
				sendFn: noop, acceptFn: noop, rejectFn: tt.fn,
			})
			code, ok := run(t, h.RejectFriendRequest, friendshipReq(t, "/friends/reject", tt.token, tt.body))
			if code != tt.wantCode {
				t.Errorf("status: got %d, want %d", code, tt.wantCode)
			}
			if ok != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", ok, tt.wantSuccess)
			}
		})
	}
}
