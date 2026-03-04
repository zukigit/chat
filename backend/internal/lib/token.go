package lib

import (
	"net/http"
	"strings"
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
