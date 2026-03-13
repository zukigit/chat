package interceptors

import (
	"context"
	"strings"

	"github.com/zukigit/chat/backend/internal/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// publicMethods lists gRPC full method names that do NOT require a JWT token.
// However, if a valid token IS present, those methods will reject the request
// (the user is already logged in).
var publicMethods = map[string]bool{
	"/auth.Auth/Login":  true,
	"/auth.Auth/Signup": true,
}

// extractBearerToken reads the "authorization" metadata header and returns
// the raw token string (stripping the "Bearer " prefix).
// Returns ("", false) if the header is absent or malformed.
func extractBearerToken(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", false
	}
	raw := values[0]
	after, found := strings.CutPrefix(raw, "Bearer ")
	if !found || after == "" {
		return "", false
	}
	return after, true
}

// UnaryJWTInterceptor validates JWT tokens for every incoming unary RPC.
//   - Public methods (Login / Signup):
//     · No need token.
//   - All other methods:
//     · No token or invalid token → reject with Unauthenticated.
//     · Valid token → proceed, username injected into context.
func UnaryJWTInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	tokenStr, hasToken := extractBearerToken(ctx)

	if publicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	// Protected method: require a valid token.
	if !hasToken {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	claims, err := lib.ValidateToken(tokenStr)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	ctx = context.WithValue(ctx, lib.ContextKeyUsername, claims.Username)
	ctx = context.WithValue(ctx, lib.ContextKeyUserID, claims.UserID)
	return handler(ctx, req)
}

// StreamJWTInterceptor applies the same JWT logic to streaming RPCs.
func StreamJWTInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	tokenStr, hasToken := extractBearerToken(ctx)

	if publicMethods[info.FullMethod] {
		return handler(srv, ss)
	}

	if !hasToken {
		return status.Error(codes.Unauthenticated, "missing authorization token")
	}

	claims, err := lib.ValidateToken(tokenStr)
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	ctx = context.WithValue(ctx, lib.ContextKeyUsername, claims.Username)
	ctx = context.WithValue(ctx, lib.ContextKeyUserID, claims.UserID)
	return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
}

// wrappedStream replaces the context inside a grpc.ServerStream.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }
