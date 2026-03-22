package lib

import (
	"context"
	"log"
	"os"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	InfoLog  = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	ErrorLog = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	WarnLog  = log.New(os.Stdout, "[WARN] ", log.LstdFlags)
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey string

// ContextKeyUsername is the context key used to store the authenticated username
// after JWT validation by the interceptor.
const ContextKeyUsername contextKey = "username"

// ContextKeyUserID is the context key used to store the authenticated user's UUID
// after JWT validation by the interceptor.
const ContextKeyUserID contextKey = "user_id"

const NotiSubjectPrefix = "sessions.noti."
const ChatSubjectPrefix = "sessions.chat."

// CallerFrom extracts the authenticated username from the request context.
// Returns "" if not present (should not happen for protected methods).
func CallerFrom(ctx context.Context) string {
	username, ok := ctx.Value(ContextKeyUsername).(string)
	if !ok {
		return ""
	}
	return username
}

// CallerIDFrom extracts the authenticated user's UUID string from the request context.
// Returns "" if not present.
func CallerIDFrom(ctx context.Context) string {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

// orderedPair returns (a, b) sorted lexicographically so that a < b,
// satisfying the DB CHECK (requester_username < addressee_username) constraint.
func OrderedPair(x, y string) (first, second string) {
	if x < y {
		return x, y
	}
	return y, x
}

// CallerUUID parses the caller's UUID from context and returns it.
// Returns an error gRPC status if the token did not carry a valid UUID —
// which should not happen in practice since the JWT interceptor sets it.
func CallerUUID(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(CallerIDFrom(ctx))
	if err != nil {
		return uuid.Nil, status.Errorf(codes.Internal, "invalid caller user_id in context: %v", err)
	}
	return id, nil
}

// OrderedUUIDPair returns the two UUIDs sorted lexicographically so that
// first.String() < second.String(), satisfying the DB CHECK constraint.
func OrderedUUIDPair(a, b uuid.UUID) (first, second uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
