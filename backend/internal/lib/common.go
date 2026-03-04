package lib

import (
	"context"
	"log"
	"os"
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

// callerFrom extracts the authenticated username from the request context.
// Returns "" if not present (should not happen for protected methods).
func CallerFrom(ctx context.Context) string {
	username, _ := ctx.Value(ContextKeyUsername).(string)
	return username
}

// orderedPair returns (a, b) sorted lexicographically so that a < b,
// satisfying the DB CHECK (requester_username < addressee_username) constraint.
func OrderedPair(x, y string) (first, second string) {
	if x < y {
		return x, y
	}
	return y, x
}
