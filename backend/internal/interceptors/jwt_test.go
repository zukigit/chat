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

// noopHandler is a gRPC unary handler that always succeeds.
var noopHandler grpc.UnaryHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
	return "ok", nil
}

func unaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

func ctxWithToken(t *testing.T, token string) context.Context {
	t.Helper()
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

func validToken(t *testing.T) string {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-interceptor-secret")
	tok, err := lib.GenerateToken("testuser")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

// ---- Protected method tests ----

func TestUnaryJWT_ProtectedMethod_NoToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-interceptor-secret")
	ctx := context.Background()
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/some.Service/Method"), noopHandler)
	if st, _ := status.FromError(err); st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}
}

func TestUnaryJWT_ProtectedMethod_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-interceptor-secret")
	ctx := ctxWithToken(t, "garbage-token")
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/some.Service/Method"), noopHandler)
	if st, _ := status.FromError(err); st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}
}

func TestUnaryJWT_ProtectedMethod_ValidToken(t *testing.T) {
	tok := validToken(t)
	ctx := ctxWithToken(t, tok)
	resp, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/some.Service/Method"), noopHandler)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// ---- Public method (Login/Signup) tests ----

func TestUnaryJWT_Login_NoToken_Passes(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-interceptor-secret")
	ctx := context.Background()
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/auth.Auth/Login"), noopHandler)
	if err != nil {
		t.Errorf("expected Login without token to pass, got %v", err)
	}
}

func TestUnaryJWT_Login_ValidToken_AlreadyLoggedIn(t *testing.T) {
	tok := validToken(t)
	ctx := ctxWithToken(t, tok)
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/auth.Auth/Login"), noopHandler)
	if st, _ := status.FromError(err); st.Code() != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists when Login called with valid token, got %v", err)
	}
}

func TestUnaryJWT_Signup_ValidToken_AlreadyLoggedIn(t *testing.T) {
	tok := validToken(t)
	ctx := ctxWithToken(t, tok)
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/auth.Auth/Signup"), noopHandler)
	if st, _ := status.FromError(err); st.Code() != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists when Signup called with valid token, got %v", err)
	}
}

func TestUnaryJWT_Login_InvalidToken_Passes(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-interceptor-secret")
	// Invalid token should be treated as "no token" for Login — let it through.
	ctx := ctxWithToken(t, "bad-token")
	_, err := interceptors.UnaryJWTInterceptor(ctx, nil, unaryInfo("/auth.Auth/Login"), noopHandler)
	if err != nil {
		t.Errorf("expected Login with invalid token to pass through, got %v", err)
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
