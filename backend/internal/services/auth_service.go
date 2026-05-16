package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		return nil, status.Errorf(codes.Internal, "login: get user: %v", err)
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
		return nil, status.Errorf(codes.Internal, "login: generate token: %v", err)
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
		return nil, status.Errorf(codes.Internal, "signup: begin tx: %v", err)
	}
	defer tx.Rollback()

	// Create queries with the transaction
	queries := db.New(tx)

	_, err = queries.GetUserByUsername(ctx, req.UserName)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "username already taken")
	}

	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "signup: check username: %v", err)
	}

	hashedPasswd, err := bcrypt.GenerateFromPassword([]byte(req.Passwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "signup: hash password: %v", err)
	}

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		UserName:     req.UserName,
		HashedPasswd: sql.NullString{String: string(hashedPasswd), Valid: true},
		SignupType:   db.SignupTypeEmail,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "signup: create user: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "signup: commit: %v", err)
	}

	return &auth.SignupResponse{}, nil
}

// SearchUsers searches for users by username or display name
func (s *AuthServer) SearchUsers(ctx context.Context, req *auth.SearchUsersRequest) (*auth.SearchUsersResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	callerID, err := lib.CallerUUID(ctx)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.sqlDB)

	users, err := queries.SearchUsers(ctx, db.SearchUsersParams{
		Column1:     sql.NullString{String: req.Query, Valid: true},
		User1Userid: callerID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search users: %v", err)
	}

	var results []*auth.UserResult
	for _, u := range users {
		var friendshipStatus string
		if s, ok := u.FriendshipStatus.(string); ok {
			friendshipStatus = s
		}
		var friendshipInitiatorUserid string
		if s, ok := u.FriendshipInitiatorUserid.(string); ok {
			friendshipInitiatorUserid = s
		}
		result := &auth.UserResult{
			UserId:                    u.UserID.String(),
			UserName:                  u.UserName,
			DisplayName:               u.DisplayName.String,
			AvatarUrl:                 u.AvatarUrl.String,
			FriendshipStatus:          friendshipStatus,
			FriendshipInitiatorUserid: friendshipInitiatorUserid,
		}
		results = append(results, result)
	}

	return &auth.SearchUsersResponse{Users: results}, nil
}

// GetGithubOAuthURL returns the GitHub authorization URL for OAuth flow.
func (s *AuthServer) GetGithubOAuthURL(ctx context.Context, req *auth.GetGithubOAuthURLRequest) (*auth.GetGithubOAuthURLResponse, error) {
	clientID := lib.Getenv("GITHUB_OAUTH_CLIENT_ID", "")
	if clientID == "" {
		return nil, status.Error(codes.InvalidArgument, "GITHUB_OAUTH_CLIENT_ID is not configured")
	}

	gatewayURL := lib.Getenv("GATEWAY_PUBLIC_URL", "")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	redirectURI := gatewayURL + "/oauth/github/callback"
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
	)

	return &auth.GetGithubOAuthURLResponse{Url: authURL}, nil
}

// githubTokenResponse represents the JSON response from GitHub's token endpoint.
type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// githubUser represents the JSON response from GitHub's user info endpoint.
type githubUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GithubOAuthCallback exchanges the GitHub OAuth code for a short-lived JWT.
func (s *AuthServer) GithubOAuthCallback(ctx context.Context, req *auth.GithubOAuthCallbackRequest) (*auth.GithubOAuthCallbackResponse, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "missing authorization code")
	}

	clientID := lib.Getenv("GITHUB_OAUTH_CLIENT_ID", "")
	clientSecret := lib.Getenv("GITHUB_OAUTH_CLIENT_SECRET", "")
	if clientID == "" || clientSecret == "" {
		return nil, status.Error(codes.Internal, "GitHub OAuth credentials not configured")
	}

	gatewayURL := lib.Getenv("GATEWAY_PUBLIC_URL", "")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	// Exchange code for access token
	tokenResp, err := exchangeCodeForToken(clientID, clientSecret, gatewayURL, req.Code)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("failed to exchange code for token: %v", err))
	}

	// Fetch GitHub user profile
	ghUser, err := fetchGitHubUser(tokenResp.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to fetch GitHub user: %v", err))
	}

	username := ghUser.Login
	if username == "" {
		return nil, status.Error(codes.Internal, "GitHub user login is empty")
	}

	queries := db.New(s.sqlDB)

	// Check if user exists
	existingUser, err := queries.GetUserByUsername(ctx, username)
	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "github callback: get user: %v", err)
	}

	if err == sql.ErrNoRows {
		// Signup path: create new user
		tx, err := s.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: begin tx: %v", err)
		}
		defer tx.Rollback()

		txQueries := db.New(tx)

		_, err = txQueries.CreateUser(ctx, db.CreateUserParams{
			UserName:     username,
			HashedPasswd: sql.NullString{Valid: false},
			SignupType:   db.SignupTypeGithub,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: create user: %v", err)
		}

		// Fetch the created user to get the ID
		createdUser, err := txQueries.GetUserByUsername(ctx, username)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: get created user: %v", err)
		}

		// Update profile with display name and avatar URL
		_, err = txQueries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
			UserName:    username,
			DisplayName: sql.NullString{String: ghUser.Name, Valid: ghUser.Name != ""},
			AvatarUrl:   sql.NullString{String: ghUser.AvatarURL, Valid: ghUser.AvatarURL != ""},
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: update profile: %v", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: commit: %v", err)
		}

		// Generate short-lived token
		shortLivedToken, err := lib.GenerateTokenWithExpiry(createdUser.UserID.String(), username, 60*time.Second)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "github callback: generate token: %v", err)
		}

		return &auth.GithubOAuthCallbackResponse{
			ShortLivedToken: shortLivedToken,
			Username:        username,
		}, nil
	}

	// Login path: user exists
	if existingUser.SignupType != db.SignupTypeGithub {
		return nil, status.Error(codes.FailedPrecondition, "this account was created with email/password, not GitHub")
	}

	// Generate short-lived token
	shortLivedToken, err := lib.GenerateTokenWithExpiry(existingUser.UserID.String(), username, 60*time.Second)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "github callback: generate token: %v", err)
	}

	return &auth.GithubOAuthCallbackResponse{
		ShortLivedToken: shortLivedToken,
		Username:        username,
	}, nil
}

// ExchangeToken exchanges a short-lived OAuth token for a long-lived session token.
func (s *AuthServer) ExchangeToken(ctx context.Context, req *auth.ExchangeTokenRequest) (*auth.ExchangeTokenResponse, error) {
	userID, ok := ctx.Value(lib.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid short-lived token")
	}

	username, ok := ctx.Value(lib.ContextKeyUsername).(string)
	if !ok || username == "" {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid short-lived token")
	}

	// Generate long-lived token
	longLivedToken, err := lib.GenerateToken(userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exchange token: generate token: %v", err)
	}

	return &auth.ExchangeTokenResponse{
		Token:    longLivedToken,
		Username: username,
	}, nil
}

// exchangeCodeForToken exchanges the GitHub OAuth code for an access token.
func exchangeCodeForToken(clientID, clientSecret, gatewayURL, code string) (*githubTokenResponse, error) {
	redirectURI := gatewayURL + "/oauth/github/callback"

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in GitHub response")
	}

	return &tokenResp, nil
}

// fetchGitHubUser fetches the user profile from GitHub API.
func fetchGitHubUser(accessToken string) (*githubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}
