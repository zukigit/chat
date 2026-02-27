package services

import (
	"context"
	"database/sql"

	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/proto/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	if req.Passwd == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Get queries
	queries := db.New(s.sqlDB)

	// Get user from database
	user, err := queries.GetUserByUsername(ctx, req.UserName)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}
	if err != nil {
		lib.ErrorLog.Printf("Failed to get user: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPasswd), []byte(req.Passwd))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
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
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	if req.Passwd == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Start a database transaction
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		lib.ErrorLog.Printf("Failed to begin transaction: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	defer tx.Rollback()

	// Create queries with the transaction
	queries := db.New(tx)

	_, err = queries.GetUserByUsername(ctx, req.UserName)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "username already taken")
	}

	if err != sql.ErrNoRows {
		lib.ErrorLog.Printf("Failed to check user existence: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	hashedPasswd, err := bcrypt.GenerateFromPassword([]byte(req.Passwd), bcrypt.DefaultCost)
	if err != nil {
		lib.ErrorLog.Printf("Failed to hash password: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		UserName:     req.UserName,
		HashedPasswd: string(hashedPasswd),
		SignupType:   db.SignupTypeEmail,
	})
	if err != nil {
		lib.ErrorLog.Printf("Failed to create user: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		lib.ErrorLog.Printf("Failed to commit transaction: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &auth.SignupResponse{}, nil
}
