package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zukigit/chat/backend/internal/lib"
)

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

// noop is a convenience no-op for mock methods that should not be called.
func noop(_ context.Context, _, _ string) error { return nil }

// friendshipReq builds a POST request with an optional Bearer token and JSON body.
func friendshipReq(t *testing.T, path, token string, body any) *http.Request {
	t.Helper()
	req := postRequest(t, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// run fires a single handler call and returns the status code and success flag.
func run(t *testing.T, h func(http.ResponseWriter, *http.Request), req *http.Request) (int, bool) {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, req)
	resp := decodeResponse(t, rec)
	return rec.Code, resp.Success
}
