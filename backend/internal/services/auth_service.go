package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/proto/auth"
	"golang.org/x/crypto/bcrypt"
)

// AuthServer implements the auth.AuthServer interface
type AuthServer struct {
	auth.UnimplementedAuthServer
	sqlDB *sql.DB
}

// NewAuthServer creates a new AuthServer instance
func NewAuthServer(sqlDB *sql.DB) *AuthServer {
	return &AuthServer{
		sqlDB: sqlDB,
	}
}

// Login handles user login
func (s *AuthServer) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	if req.UserName == "" {
		return nil, fmt.Errorf("username is required")
	}

	if req.Passwd == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Get queries
	queries := db.New(s.sqlDB)

	// Get user from database
	user, err := queries.GetUserByUsername(ctx, req.UserName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid username or password")
	}
	if err != nil {
		lib.ErrorLog.Printf("Failed to get user: %v", err)
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPasswd), []byte(req.Passwd))
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// TODO: Generate JWT token
	token := "token_" + req.UserName

	return &auth.LoginResponse{
		Token: token,
	}, nil
}

// Signup handles user signup
func (s *AuthServer) Signup(ctx context.Context, req *auth.SignupRequest) (*auth.SignupResponse, error) {
	if req.UserName == "" {
		return nil, fmt.Errorf("username is required")
	}

	if req.Passwd == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Start a database transaction
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("Failed to begin transaction: %v", err)
		return nil, err
	}
	defer tx.Rollback()

	// Create queries with the transaction
	queries := db.New(tx)

	_, err = queries.GetUserByUsername(ctx, req.UserName)
	if err == nil {
		return nil, fmt.Errorf("user already exists")
	}

	if err != sql.ErrNoRows {
		lib.ErrorLog.Printf("Failed to check user existence: %v", err)
		return nil, err
	}

	hashedPasswd, err := bcrypt.GenerateFromPassword([]byte(req.Passwd), bcrypt.DefaultCost)
	if err != nil {
		lib.ErrorLog.Printf("Failed to hash password: %v", err)
		return nil, err
	}

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		UserName:     req.UserName,
		HashedPasswd: string(hashedPasswd),
		SignupType:   db.SignupTypeEmail,
	})
	if err != nil {
		lib.ErrorLog.Printf("Failed to create user: %v", err)
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		lib.ErrorLog.Printf("Failed to commit transaction: %v", err)
		return nil, err
	}

	return &auth.SignupResponse{}, nil
}
