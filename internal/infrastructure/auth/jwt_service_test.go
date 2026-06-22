package auth_test

import (
	"context"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	infraAuth "gosample/internal/infrastructure/auth"
	"gosample/internal/infrastructure/config"
)

func newTestConfig(secret string) *config.Config {
	return &config.Config{JWTSecret: secret}
}

func TestJWTService_Sign_ReturnsNonEmptyToken(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	token, err := svc.Sign(context.Background(), "user-123", "STUDENT")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_Verify_ValidToken_ReturnsClaims(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	token, err := svc.Sign(context.Background(), "user-456", "TEACHER")
	require.NoError(t, err)

	claims, err := svc.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "user-456", claims.UserID)
	assert.Equal(t, "TEACHER", claims.Role)
}

func TestJWTService_Verify_TamperedToken_ReturnsUnauthorized(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	token, err := svc.Sign(context.Background(), "user-789", "ADMIN")
	require.NoError(t, err)

	tampered := token + "tamper"
	_, err = svc.Verify(context.Background(), tampered)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestJWTService_Verify_WrongSecret_ReturnsUnauthorized(t *testing.T) {
	signer := infraAuth.NewJWTService(newTestConfig("secret-A"))
	verifier := infraAuth.NewJWTService(newTestConfig("secret-B"))

	token, err := signer.Sign(context.Background(), "user-111", "STUDENT")
	require.NoError(t, err)

	_, err = verifier.Verify(context.Background(), token)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestJWTService_Verify_EmptyToken_ReturnsUnauthorized(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	_, err := svc.Verify(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestJWTService_Verify_MalformedToken_ReturnsUnauthorized(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	_, err := svc.Verify(context.Background(), "not.a.jwt")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

// TestJWTService_Sign_TokenIsValidImmediately checks that a freshly-signed token is valid now.
func TestJWTService_Sign_TokenIsValidImmediately(t *testing.T) {
	svc := infraAuth.NewJWTService(newTestConfig("test-secret"))
	token, _ := svc.Sign(context.Background(), "user-222", "ADMIN")
	claims, err := svc.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "user-222", claims.UserID)
	_ = time.Now() // suppress unused import warning
}

func TestJWTService_Sign_TokenExpiry_Is24Hours(t *testing.T) {
	cfg := newTestConfig("test-secret")
	svc := infraAuth.NewJWTService(cfg)
	before := time.Now()
	token, err := svc.Sign(context.Background(), "user-exp", "ADMIN")
	require.NoError(t, err)
	after := time.Now()

	type rawClaims struct {
		jwtLib.RegisteredClaims
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	var claims rawClaims
	_, err = jwtLib.ParseWithClaims(token, &claims, func(t *jwtLib.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	require.NoError(t, err)
	require.NotNil(t, claims.ExpiresAt)

	minExpiry := before.Add(23*time.Hour + 59*time.Minute)
	maxExpiry := after.Add(24*time.Hour + 1*time.Minute)
	assert.True(t, claims.ExpiresAt.Time.After(minExpiry),
		"exp %v should be after %v", claims.ExpiresAt.Time, minExpiry)
	assert.True(t, claims.ExpiresAt.Time.Before(maxExpiry),
		"exp %v should be before %v", claims.ExpiresAt.Time, maxExpiry)
}
