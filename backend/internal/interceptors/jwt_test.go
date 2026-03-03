package interceptors_test

import (
	"context"
	"testing"

	"github.com/zukigit/chat/backend/internal/interceptors"
	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-interceptor-secret"

// noopHandler is a gRPC unary handler that always succeeds.
var noopHandler grpc.UnaryHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
	return "ok", nil
}

func unaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

// ctxWithToken returns a context carrying the given token as a Bearer credential.
func ctxWithToken(t *testing.T, token string) context.Context {
	t.Helper()
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

// validToken sets JWT_SECRET and generates a token for "testuser".
func validToken(t *testing.T) string {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	tok, err := lib.GenerateToken("testuser")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

// ---- Protected method tests ----

func TestUnaryJWT_ProtectedMethod(t *testing.T) {
	tok := validToken(t) // also sets JWT_SECRET

	cases := []struct {
		name    string
		ctx     context.Context
		wantErr codes.Code
	}{
		{
			name:    "no_token",
			ctx:     context.Background(),
			wantErr: codes.Unauthenticated,
		},
		{
			name:    "invalid_token",
			ctx:     ctxWithToken(t, "garbage-token"),
			wantErr: codes.Unauthenticated,
		},
		{
			name:    "valid_token",
			ctx:     ctxWithToken(t, tok),
			wantErr: codes.OK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := interceptors.UnaryJWTInterceptor(tc.ctx, nil, unaryInfo("/some.Service/Method"), noopHandler)
			st, _ := status.FromError(err)
			if st.Code() != tc.wantErr {
				t.Errorf("got gRPC code %v, want %v (err: %v)", st.Code(), tc.wantErr, err)
			}
		})
	}
}

// ---- Public method (Login/Signup) tests ----

func TestUnaryJWT_PublicMethods_AlwaysPass(t *testing.T) {
	tok := validToken(t) // also sets JWT_SECRET

	cases := []struct {
		name   string
		method string
		ctx    context.Context
	}{
		{"login_no_token", "/auth.Auth/Login", context.Background()},
		{"login_invalid_token", "/auth.Auth/Login", ctxWithToken(t, "bad-token")},
		{"login_valid_token", "/auth.Auth/Login", ctxWithToken(t, tok)},
		{"signup_no_token", "/auth.Auth/Signup", context.Background()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := interceptors.UnaryJWTInterceptor(tc.ctx, nil, unaryInfo(tc.method), noopHandler)
			if err != nil {
				t.Errorf("expected public method %q to pass through, got %v", tc.method, err)
			}
		})
	}
}

// ---- Username injected into context ----

func TestUnaryJWT_UsernameInContext(t *testing.T) {
	tok := validToken(t)
	ctx := ctxWithToken(t, tok)

	var capturedUsername string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedUsername, _ = ctx.Value(lib.ContextKeyUsername).(string)
		return "ok", nil
	}

	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/some.Service/Method"), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUsername != "testuser" {
		t.Errorf("expected username 'testuser' in context, got %q", capturedUsername)
	}
}
