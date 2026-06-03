package clients

import (
	"context"

	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// AuthClient wraps the gRPC Auth client to communicate with the backend.
type AuthClient struct {
	client auth.AuthClient
	conn   *grpc.ClientConn
}

// NewAuthClient dials the backend gRPC server and returns an AuthClient.
// The caller is responsible for calling Close() when done.
func NewAuthClient(backendAddr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(
		backendAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	)
	if err != nil {
		return nil, err
	}
	return &AuthClient{
		client: auth.NewAuthClient(conn),
		conn:   conn,
	}, nil
}

// Close releases the underlying gRPC connection.
func (a *AuthClient) Close() {
	a.conn.Close()
}

// Login forwards the login request to the backend via gRPC.
func (a *AuthClient) Login(ctx context.Context, username, password string) (token string, err error) {
	resp, err := a.client.Login(ctx, &auth.LoginRequest{
		UserName: username,
		Passwd:   password,
	})
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

// Signup forwards the signup request to the backend via gRPC.
func (a *AuthClient) Signup(ctx context.Context, username, password string) error {
	_, err := a.client.Signup(ctx, &auth.SignupRequest{
		UserName: username,
		Passwd:   password,
	})
	return err
}

// SearchUsers asks the backend to search for users by username or display name.
func (a *AuthClient) SearchUsers(ctx context.Context, token, query string) (*auth.SearchUsersResponse, error) {
	return a.client.SearchUsers(lib.WithToken(ctx, token), &auth.SearchUsersRequest{
		Query: query,
	})
}

// GetGithubOAuthURL retrieves the GitHub OAuth authorization URL.
func (a *AuthClient) GetGithubOAuthURL(ctx context.Context) (string, error) {
	resp, err := a.client.GetGithubOAuthURL(ctx, &auth.GetGithubOAuthURLRequest{})
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

// GithubOAuthCallback exchanges the GitHub OAuth code for a short-lived JWT.
func (a *AuthClient) GithubOAuthCallback(ctx context.Context, code string) (shortLivedToken, username string, err error) {
	resp, err := a.client.GithubOAuthCallback(ctx, &auth.GithubOAuthCallbackRequest{Code: code})
	if err != nil {
		return "", "", err
	}
	return resp.ShortLivedToken, resp.Username, nil
}

// ExchangeToken exchanges a short-lived OAuth token for a long-lived session token.
func (a *AuthClient) ExchangeToken(ctx context.Context, shortLivedToken string) (longLivedToken, username string, err error) {
	resp, err := a.client.ExchangeToken(
		lib.WithToken(ctx, shortLivedToken),
		&auth.ExchangeTokenRequest{},
	)
	if err != nil {
		return "", "", err
	}
	return resp.Token, resp.Username, nil
}

// SetupKeys forwards the E2EE key setup request to the backend via gRPC.
func (a *AuthClient) SetupKeys(ctx context.Context, token, publicKey, encryptedPrivateKey string) (bool, error) {
	resp, err := a.client.SetupKeys(
		lib.WithToken(ctx, token),
		&auth.SetupKeysRequest{
			PublicKey:           publicKey,
			EncryptedPrivateKey: encryptedPrivateKey,
		},
	)
	if err != nil {
		return false, err
	}
	return resp.IsE2EeReady, nil
}

// GetMyKeys retrieves the user's E2EE keys from the backend via gRPC.
func (a *AuthClient) GetMyKeys(ctx context.Context, token string) (encryptedPrivateKey, publicKey string, isE2eeReady bool, err error) {
	resp, err := a.client.GetMyKeys(
		lib.WithToken(ctx, token),
		&auth.GetMyKeysRequest{},
	)
	if err != nil {
		return "", "", false, err
	}
	return resp.EncryptedPrivateKey, resp.PublicKey, resp.IsE2EeReady, nil
}

// GetPublicKeys retrieves public keys for multiple users from the backend via gRPC.
func (a *AuthClient) GetPublicKeys(ctx context.Context, token string, userIDs []string) (map[string]string, error) {
	resp, err := a.client.GetPublicKeys(
		lib.WithToken(ctx, token),
		&auth.GetPublicKeysRequest{UserIds: userIDs},
	)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(resp.Keys))
	for _, k := range resp.Keys {
		m[k.UserId] = k.PublicKey
	}
	return m, nil
}
