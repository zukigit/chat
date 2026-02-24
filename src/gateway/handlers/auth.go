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

// /login
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

// /signup
func SignupHandler(q *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req signupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
				Success: false,
				Message: "invalid request body",
			})
			return
		}

		if req.Username == "" {
			lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
				Success: false,
				Message: "username is required",
			})
			return
		}

		if req.Type == db.SignupTypeEmail {
			if req.Password == "" {
				lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
					Success: false,
					Message: "password is required for email signup",
				})
				return
			}

			// Hash password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
					Success: false,
					Message: "internal server error",
				})
				return
			}

			// Create user in database with hashed password
			_, err = q.CreateUser(r.Context(), db.CreateUserParams{
				UserName:     req.Username,
				HashedPasswd: string(hashedPassword),
				SignupType:   db.SignupTypeEmail,
			})
			if err != nil {
				lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
					Success: false,
					Message: "failed to create user",
				})
				return
			}

			lib.WriteJSON(w, http.StatusCreated, lib.Response{
				Success: true,
				Message: "user registered successfully",
			})
			return
		}

		if req.Type == db.SignupTypeGoogle {
			if req.Code == "" {
				lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
					Success: false,
					Message: "code is required for google signup",
				})
				return
			}

			// TODO: Verify google code with Google OAuth
			// For now, creating user with empty password since Google handles authentication
			_, err := q.CreateUser(r.Context(), db.CreateUserParams{
				UserName:     req.Username,
				HashedPasswd: "", // Google OAuth doesn't use passwords
				SignupType:   db.SignupTypeGoogle,
			})
			if err != nil {
				lib.WriteJSON(w, http.StatusInternalServerError, lib.Response{
					Success: false,
					Message: "failed to create user",
				})
				return
			}

			lib.WriteJSON(w, http.StatusCreated, lib.Response{
				Success: true,
				Message: "user registered via google successfully",
			})
			return
		}

		lib.WriteJSON(w, http.StatusBadRequest, lib.Response{
			Success: false,
			Message: "invalid signup type.",
		})
	}
}
