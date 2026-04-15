package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
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
	if sqlDB != nil {
		return &AuthServer{
			sqlDB: sqlDB,
		}
	}

	return nil
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

	// Verify password — reject OAuth accounts that have no password set.
	if !user.HashedPasswd.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPasswd.String), []byte(req.Passwd))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	// Generate JWT token (contains a fresh login_id UUID)
	token, err := lib.GenerateToken(user.UserID.String(), req.UserName)
	if err != nil {
		lib.ErrorLog.Printf("Failed to generate JWT token: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Parse the claims back to get the login_id we embedded.
	claims, err := lib.ValidateToken(token)
	if err != nil {
		err = fmt.Errorf("failed to re-parse token claims: %w", err)
		return nil, status.Errorf(codes.Internal, "internal server error: %s", err)
	}

	loginID, err := uuid.Parse(claims.LoginID)
	if err != nil {
		lib.ErrorLog.Printf("Failed to parse login_id: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Record this login session so ValidateSession and publishIfOnline can find it.
	queries2 := db.New(s.sqlDB)
	if err := queries2.CreateSession(ctx, db.CreateSessionParams{
		UserUserid: user.UserID,
		LoginID:    loginID,
	}); err != nil {
		lib.ErrorLog.Printf("Failed to create session: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &auth.LoginResponse{Token: token}, nil
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
		HashedPasswd: sql.NullString{String: string(hashedPasswd), Valid: true},
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

// Logout deletes the caller's session row, invalidating the login_id embedded
// in the JWT. The gRPC interceptor already verified the token and stored
// login_id in the context.
func (s *AuthServer) Logout(ctx context.Context, _ *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	loginIDStr := lib.CallerLoginID(ctx)
	loginID, err := uuid.Parse(loginIDStr)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid login_id in token")
	}

	q := db.New(s.sqlDB)
	if err := q.DeleteSessionByLoginID(ctx, loginID); err != nil && err != sql.ErrNoRows {
		lib.ErrorLog.Printf("Logout: delete session: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &auth.LogoutResponse{}, nil
}
