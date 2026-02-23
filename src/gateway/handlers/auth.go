package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/src/gateway/lib"
)

// POST /login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "username and password are required",
		})
		return
	}

	// TODO: replace with real credential verification
	if req.Username != "admin" || req.Password != "secret" {
		lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
			Success: false,
			Message: "invalid credentials",
		})
		return
	}

	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Message: "login successful",
		Data:    map[string]string{"token": "your-jwt-token-here"},
	})
}