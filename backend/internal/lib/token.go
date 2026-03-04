package lib

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// BearerToken extracts the raw JWT from the HTTP Authorization header.
// Returns ("", false) if absent or malformed.
func BearerToken(r *http.Request) (string, bool) {
	raw := r.Header.Get("Authorization")
	token, found := strings.CutPrefix(raw, "Bearer ")
	if !found || token == "" {
		return "", false
	}
	return token, true
}

// withToken attaches the Bearer token to the outgoing gRPC context.
func WithToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}
