package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zukigit/chat/backend/internal/lib"
)

// /login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	//TODO: Implement actual authentication logic with backend api

	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Message: "login successful",
		Data:    map[string]string{"token": "your-jwt-token-here"},
	})
}

// /signup
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	lib.WriteJSON(w, http.StatusCreated, lib.Response{
		Success: true,
		Message: "user registered successfully",
	})
}
