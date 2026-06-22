package auth

import (
	"context"
	"errors"
)

var (
	ErrInvalidToken  = errors.New("invalid_token")
	ErrUnauthorized  = errors.New("unauthorized")
)

type GoogleClaims struct {
	Sub   string
	Email string
	Name  string
}

type IGoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleClaims, error)
}

type IPermissionService interface {
	GetPermissionsForRole(ctx context.Context, role string) (map[string]interface{}, error)
}

type JWTClaims struct {
	UserID string
	Role   string
}

type IJWTService interface {
	Sign(ctx context.Context, userID, role string) (string, error)
	Verify(ctx context.Context, token string) (*JWTClaims, error)
}

type ContextPermission struct {
	UserID string
	Role   string
}
