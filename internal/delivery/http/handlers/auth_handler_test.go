package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "gosample/internal/delivery/http"
	"gosample/internal/delivery/http/handlers"
	domainAuth "gosample/internal/domain/auth"
	authUseCase "gosample/internal/usecase/auth"
)

// --- Mock google auth use case ---

type mockGoogleAuthUseCase struct {
	result authUseCase.AuthResponseDTO
	err    error
}

func (m *mockGoogleAuthUseCase) Execute(_ context.Context, _ string) (authUseCase.AuthResponseDTO, error) {
	return m.result, m.err
}

// --- helpers for auth handler ---

func newAuthEchoContext(body []byte) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(http.MethodPost, "/auth/google", nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// --- GoogleAuth handler tests ---

func TestAuthHandler_GoogleAuth_Success_Returns200(t *testing.T) {
	dto := authUseCase.AuthResponseDTO{
		Name:        "Alice",
		Email:       "alice@example.com",
		Role:        "ADMIN",
		Token:       "jwt.token.here",
		Permissions: map[string]interface{}{"classes": "read"},
	}
	uc := &mockGoogleAuthUseCase{result: dto}
	h := handlers.NewAuthHandler(uc)

	body, _ := json.Marshal(map[string]string{"idToken": "google-id-token"})
	c, rec := newAuthEchoContext(body)

	err := h.GoogleAuth(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Alice", resp.Name)
	assert.Equal(t, "alice@example.com", resp.Email)
	assert.Equal(t, "ADMIN", resp.Role)
	assert.Equal(t, "jwt.token.here", resp.Token)
}

func TestAuthHandler_GoogleAuth_InvalidToken_Returns401(t *testing.T) {
	uc := &mockGoogleAuthUseCase{err: domainAuth.ErrInvalidToken}
	h := handlers.NewAuthHandler(uc)

	body, _ := json.Marshal(map[string]string{"idToken": "bad-token"})
	c, rec := newAuthEchoContext(body)

	err := h.GoogleAuth(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp api.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_token", resp.Error)
}

func TestAuthHandler_GoogleAuth_InternalError_Returns500(t *testing.T) {
	uc := &mockGoogleAuthUseCase{err: errors.New("unexpected error")}
	h := handlers.NewAuthHandler(uc)

	body, _ := json.Marshal(map[string]string{"idToken": "token"})
	c, rec := newAuthEchoContext(body)

	err := h.GoogleAuth(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp api.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "internal_error", resp.Error)
}

func TestAuthHandler_GoogleAuth_BadRequestBody_Returns400(t *testing.T) {
	uc := &mockGoogleAuthUseCase{}
	h := handlers.NewAuthHandler(uc)

	// Send malformed JSON — Bind will fail
	c, rec := newAuthEchoContext([]byte("not-json{{{"))
	err := h.GoogleAuth(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// suppress unused import
var _ = errors.New
