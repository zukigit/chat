package lib

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims holds the data embedded inside a JWT.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	LoginID  string `json:"login_id"`
	jwt.RegisteredClaims
}

// jwtSecret returns the signing secret, falling back to a dev-only default.
func jwtSecret() []byte {
	secret := Getenv("JWT_SECRET", "")
	if secret == "" {
		WarnLog.Println("JWT_SECRET env var is not set — using insecure dev default. Do NOT use this in production.")
		secret = "dev-secret-change-me"
	}
	return []byte(secret)
}

// GenerateToken signs a JWT for the given userID and username with a 24-hour expiry.
// A fresh login_id UUID is embedded so the gateway can use it as a stable durable
// consumer name for the lifetime of this token.
func GenerateToken(userID, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		LoginID:  uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ValidateToken parses and validates a JWT string, returning the embedded Claims.
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ParseTokenUnverified decodes the JWT claims without checking the signature.
// Safe for use in the gateway, which has no JWT secret — signature verification
// is the backend's responsibility via the gRPC interceptor.
func ParseTokenUnverified(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	p := jwt.NewParser()
	_, _, err := p.ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
