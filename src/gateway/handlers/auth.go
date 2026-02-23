package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zukigit/chat/src/gateway/lib"
	db "github.com/zukigit/chat/src/lib/db"
	"golang.org/x/crypto/bcrypt"
)

// LoginHandler returns an http.HandlerFunc that uses the provided db.Queries
// to verify credentials against the database.
func LoginHandler(q *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		user, err := q.GetUserByUsername(r.Context(), req.Username)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				lib.WriteJSON(w, http.StatusUnauthorized, lib.Response{
					Success: false,
					Message: "invalid credentials",
				})
			} else {
				lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
					Success: false,
					Message: "internal server error",
				})
			}
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPasswd), []byte(req.Password)); err != nil {
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
}