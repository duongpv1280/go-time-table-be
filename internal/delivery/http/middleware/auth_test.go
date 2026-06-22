package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	httpMiddleware "gosample/internal/delivery/http/middleware"
)

// --- Mock JWT Service ---

type mockJWTService struct {
	claims *domainAuth.JWTClaims
	err    error
}

func (m *mockJWTService) Sign(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (m *mockJWTService) Verify(_ context.Context, _ string) (*domainAuth.JWTClaims, error) {
	return m.claims, m.err
}

// --- Helpers ---

func newEchoContext(method, path, authHeader string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func nextHandler(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// --- Tests ---

func TestJWTAuth_ValidToken_CallsNextAndSetsPermission(t *testing.T) {
	claims := &domainAuth.JWTClaims{UserID: "user-1", Role: "TEACHER"}
	svc := &mockJWTService{claims: claims}

	c, rec := newEchoContext(http.MethodGet, "/", "Bearer valid-token")
	mw := httpMiddleware.JWTAuth(svc)

	var capturedPerm domainAuth.ContextPermission
	var permFound bool
	err := mw(func(ctx echo.Context) error {
		capturedPerm, permFound = httpMiddleware.GetPermission(ctx)
		return ctx.String(http.StatusOK, "ok")
	})(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, permFound)
	assert.Equal(t, "user-1", capturedPerm.UserID)
	assert.Equal(t, "TEACHER", capturedPerm.Role)
}

func TestJWTAuth_MissingAuthHeader_Returns401(t *testing.T) {
	svc := &mockJWTService{}
	c, rec := newEchoContext(http.MethodGet, "/", "")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_NonBearerHeader_Returns401(t *testing.T) {
	svc := &mockJWTService{}
	c, rec := newEchoContext(http.MethodGet, "/", "Basic abc123")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_InvalidToken_Returns401(t *testing.T) {
	svc := &mockJWTService{err: domainAuth.ErrUnauthorized}
	c, rec := newEchoContext(http.MethodGet, "/", "Bearer bad-token")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	svc := &mockJWTService{err: errors.New("token is expired")}
	c, rec := newEchoContext(http.MethodGet, "/", "Bearer expired-token")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_EmptyUserID_Returns401(t *testing.T) {
	svc := &mockJWTService{claims: &domainAuth.JWTClaims{UserID: "", Role: "ADMIN"}}
	c, rec := newEchoContext(http.MethodGet, "/", "Bearer some-token")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_EmptyRole_Returns401(t *testing.T) {
	svc := &mockJWTService{claims: &domainAuth.JWTClaims{UserID: "user-123", Role: ""}}
	c, rec := newEchoContext(http.MethodGet, "/", "Bearer some-token")
	mw := httpMiddleware.JWTAuth(svc)

	err := mw(nextHandler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetPermission_NoPermissionSet_ReturnsFalse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	perm, ok := httpMiddleware.GetPermission(c)
	assert.False(t, ok)
	assert.Equal(t, domainAuth.ContextPermission{}, perm)
}
