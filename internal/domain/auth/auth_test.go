package auth_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	domainAuth "gosample/internal/domain/auth"
)

func TestErrInvalidToken_IsDistinctError(t *testing.T) {
	assert.NotEqual(t, domainAuth.ErrInvalidToken, domainAuth.ErrUnauthorized)
	assert.True(t, errors.Is(domainAuth.ErrInvalidToken, domainAuth.ErrInvalidToken))
}

func TestErrUnauthorized_IsDistinctError(t *testing.T) {
	assert.True(t, errors.Is(domainAuth.ErrUnauthorized, domainAuth.ErrUnauthorized))
	assert.False(t, errors.Is(domainAuth.ErrUnauthorized, domainAuth.ErrInvalidToken))
}

func TestErrInvalidToken_DoesNotMatchErrUnauthorized(t *testing.T) {
	assert.False(t, errors.Is(domainAuth.ErrInvalidToken, domainAuth.ErrUnauthorized))
}

func TestContextPermission_ZeroValue(t *testing.T) {
	var p domainAuth.ContextPermission
	assert.Empty(t, p.UserID)
	assert.Empty(t, p.Role)
}

func TestContextPermission_FieldAssignment(t *testing.T) {
	p := domainAuth.ContextPermission{UserID: "user-123", Role: "ADMIN"}
	assert.Equal(t, "user-123", p.UserID)
	assert.Equal(t, "ADMIN", p.Role)
}

func TestJWTClaims_ZeroValue(t *testing.T) {
	var c domainAuth.JWTClaims
	assert.Empty(t, c.UserID)
	assert.Empty(t, c.Role)
}

func TestJWTClaims_FieldAssignment(t *testing.T) {
	c := domainAuth.JWTClaims{UserID: "abc", Role: "TEACHER"}
	assert.Equal(t, "abc", c.UserID)
	assert.Equal(t, "TEACHER", c.Role)
}

func TestGoogleClaims_ZeroValue(t *testing.T) {
	var g domainAuth.GoogleClaims
	assert.Empty(t, g.Sub)
	assert.Empty(t, g.Email)
	assert.Empty(t, g.Name)
}

func TestGoogleClaims_FieldAssignment(t *testing.T) {
	g := domainAuth.GoogleClaims{Sub: "sub-1", Email: "a@b.com", Name: "Alice"}
	assert.Equal(t, "sub-1", g.Sub)
	assert.Equal(t, "a@b.com", g.Email)
	assert.Equal(t, "Alice", g.Name)
}
