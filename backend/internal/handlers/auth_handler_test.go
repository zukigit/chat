package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zukigit/chat/backend/internal/handlers"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockAuthClient implements the authClientInterface inside the handlers package.
type mockAuthClient struct {
	loginFn  func(ctx context.Context, username, password string) (string, error)
	signupFn func(ctx context.Context, username, password string) error
}

func (m *mockAuthClient) Login(ctx context.Context, username, password string) (string, error) {
	return m.loginFn(ctx, username, password)
}

func (m *mockAuthClient) Signup(ctx context.Context, username, password string) error {
	return m.signupFn(ctx, username, password)
}

// postRequest builds a POST request with a JSON body.
func postRequest(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// decodeResponse unmarshals the recorder body into lib.Response.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) lib.Response {
	t.Helper()
	var resp lib.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

// ---- Login tests ----

func TestLoginHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		mockFn         func(ctx context.Context, username, password string) (string, error)
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name: "valid credentials returns 200 with token",
			body: map[string]string{"username": "alice", "password": "secret"},
			mockFn: func(_ context.Context, _, _ string) (string, error) {
				return "jwt-token-abc", nil
			},
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name: "wrong password returns 401",
			body: map[string]string{"username": "alice", "password": "wrongpass"},
			mockFn: func(_ context.Context, _, _ string) (string, error) {
				return "", status.Error(codes.Unauthenticated, "invalid username or password")
			},
			wantStatusCode: http.StatusUnauthorized,
			wantSuccess:    false,
		},
		{
			name: "user not found returns 401",
			body: map[string]string{"username": "nobody", "password": "pass"},
			mockFn: func(_ context.Context, _, _ string) (string, error) {
				return "", status.Error(codes.NotFound, "user not found")
			},
			wantStatusCode: http.StatusUnauthorized,
			wantSuccess:    false,
		},
		{
			name: "backend internal error returns 500",
			body: map[string]string{"username": "alice", "password": "secret"},
			mockFn: func(_ context.Context, _, _ string) (string, error) {
				return "", status.Error(codes.Internal, "db connection lost")
			},
			wantStatusCode: http.StatusInternalServerError,
			wantSuccess:    false,
		},
		{
			name:           "malformed JSON body returns 400",
			body:           "not-json",
			mockFn:         nil, // should not be called
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthClient{
				loginFn: tt.mockFn,
				// Signup is not used by the Login handler but must be set.
				signupFn: func(_ context.Context, _, _ string) error { return nil },
			}

			h := handlers.NewAuthHandler(mock)
			rec := httptest.NewRecorder()

			var req *http.Request
			if s, ok := tt.body.(string); ok {
				// Send a raw string to trigger JSON decode error.
				req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(s))
			} else {
				req = postRequest(t, "/login", tt.body)
			}

			h.Login(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatusCode)
			}

			resp := decodeResponse(t, rec)
			if resp.Success != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", resp.Success, tt.wantSuccess)
			}
		})
	}
}

// ---- Signup tests ----

func TestSignupHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		mockFn         func(ctx context.Context, username, password string) error
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name: "valid signup returns 201",
			body: map[string]string{"username": "bob", "passwd": "pass123"},
			mockFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			wantStatusCode: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name: "duplicate username returns 409",
			body: map[string]string{"username": "bob", "passwd": "pass123"},
			mockFn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.AlreadyExists, "username already taken")
			},
			wantStatusCode: http.StatusConflict,
			wantSuccess:    false,
		},
		{
			name: "backend internal error returns 500",
			body: map[string]string{"username": "bob", "passwd": "pass123"},
			mockFn: func(_ context.Context, _, _ string) error {
				return status.Error(codes.Internal, "internal server error")
			},
			wantStatusCode: http.StatusInternalServerError,
			wantSuccess:    false,
		},
		{
			name: "malformed JSON body returns 400",
			body: "not json",
			mockFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantSuccess:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthClient{
				signupFn: tt.mockFn,
			}

			h := handlers.NewAuthHandler(mock)
			rec := httptest.NewRecorder()

			var req *http.Request
			if s, ok := tt.body.(string); ok {
				req = httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(s))
			} else {
				req = postRequest(t, "/signup", tt.body)
			}

			h.Signup(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatusCode)
			}

			resp := decodeResponse(t, rec)
			if resp.Success != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", resp.Success, tt.wantSuccess)
			}
		})
	}
}
