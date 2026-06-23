package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "gosample/internal/delivery/http/openapi"
	"gosample/internal/delivery/http/handlers"
	userDomain "gosample/internal/domain/user"
	userUseCase "gosample/internal/usecase/user"
)

// --- Mock user use case ---

type mockUserUseCase struct {
	createResult userUseCase.UserDTO
	createErr    error
	getResult    userUseCase.UserDTO
	getErr       error
	listResult   []userUseCase.UserDTO
	listErr      error
	deleteErr    error
}

func (m *mockUserUseCase) CreateUser(_ context.Context, _ userUseCase.CreateUserParams) (userUseCase.UserDTO, error) {
	return m.createResult, m.createErr
}

func (m *mockUserUseCase) GetUser(_ context.Context, _ string) (userUseCase.UserDTO, error) {
	return m.getResult, m.getErr
}

func (m *mockUserUseCase) ListUsers(_ context.Context) ([]userUseCase.UserDTO, error) {
	return m.listResult, m.listErr
}

func (m *mockUserUseCase) DeleteUser(_ context.Context, _ string) error {
	return m.deleteErr
}

// --- helpers ---

func makeUserDTO(name, email string) userUseCase.UserDTO {
	now := time.Now().UTC()
	return userUseCase.UserDTO{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newEchoContext(method, path string, body []byte) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func newEchoContextWithParam(method, path, paramName, paramValue string) (echo.Context, *httptest.ResponseRecorder) {
	c, rec := newEchoContext(method, path, nil)
	c.SetParamNames(paramName)
	c.SetParamValues(paramValue)
	return c, rec
}

// --- CreateUser tests ---

func TestUserHandler_CreateUser_Success_Returns201(t *testing.T) {
	dto := makeUserDTO("Alice", "alice@example.com")
	uc := &mockUserUseCase{createResult: dto}
	h := handlers.NewUserHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "name": "Alice"})
	c, rec := newEchoContext(http.MethodPost, "/users", body)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp api.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Alice", resp.Name)
	assert.Equal(t, openapi_types.Email("alice@example.com"), resp.Email)
}

func TestUserHandler_CreateUser_InvalidEmail_Returns400(t *testing.T) {
	uc := &mockUserUseCase{createErr: userDomain.ErrInvalidEmail}
	h := handlers.NewUserHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "bad", "name": "Alice"})
	c, rec := newEchoContext(http.MethodPost, "/users", body)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_CreateUser_EmptyName_Returns400(t *testing.T) {
	uc := &mockUserUseCase{createErr: userDomain.ErrEmptyName}
	h := handlers.NewUserHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "name": ""})
	c, rec := newEchoContext(http.MethodPost, "/users", body)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_CreateUser_DuplicateEmail_Returns400(t *testing.T) {
	uc := &mockUserUseCase{createErr: userDomain.ErrUserAlreadyExists}
	h := handlers.NewUserHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "name": "Alice"})
	c, rec := newEchoContext(http.MethodPost, "/users", body)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_CreateUser_InternalError_Returns500(t *testing.T) {
	uc := &mockUserUseCase{createErr: errors.New("unexpected db error")}
	h := handlers.NewUserHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "name": "Alice"})
	c, rec := newEchoContext(http.MethodPost, "/users", body)

	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUserHandler_CreateUser_BadRequestBody_Returns400(t *testing.T) {
	uc := &mockUserUseCase{}
	h := handlers.NewUserHandler(uc)

	c, rec := newEchoContext(http.MethodPost, "/users", []byte("not-json{{{"))
	err := h.CreateUser(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListUsers tests ---

func TestUserHandler_ListUsers_Success_Returns200(t *testing.T) {
	dto1 := makeUserDTO("Alice", "alice@example.com")
	dto2 := makeUserDTO("Bob", "bob@example.com")
	uc := &mockUserUseCase{listResult: []userUseCase.UserDTO{dto1, dto2}}
	h := handlers.NewUserHandler(uc)

	c, rec := newEchoContext(http.MethodGet, "/users", nil)

	err := h.ListUsers(c, api.ListUsersParams{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.ListUsersResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 2)
}

func TestUserHandler_ListUsers_Empty_Returns200WithEmptySlice(t *testing.T) {
	uc := &mockUserUseCase{listResult: []userUseCase.UserDTO{}}
	h := handlers.NewUserHandler(uc)

	c, rec := newEchoContext(http.MethodGet, "/users", nil)

	err := h.ListUsers(c, api.ListUsersParams{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.ListUsersResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.Data)
}

func TestUserHandler_ListUsers_InternalError_Returns500(t *testing.T) {
	uc := &mockUserUseCase{listErr: errors.New("db error")}
	h := handlers.NewUserHandler(uc)

	c, rec := newEchoContext(http.MethodGet, "/users", nil)

	err := h.ListUsers(c, api.ListUsersParams{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- GetUser tests ---

func TestUserHandler_GetUser_Success_Returns200(t *testing.T) {
	dto := makeUserDTO("Alice", "alice@example.com")
	uc := &mockUserUseCase{getResult: dto}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.MustParse(dto.ID))
	c, rec := newEchoContext(http.MethodGet, "/users/"+dto.ID, nil)

	err := h.GetUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Alice", resp.Name)
}

func TestUserHandler_GetUser_NotFound_Returns404(t *testing.T) {
	uc := &mockUserUseCase{getErr: userDomain.ErrUserNotFound}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodGet, "/users/"+id.String(), nil)

	err := h.GetUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_GetUser_InvalidID_Returns404(t *testing.T) {
	uc := &mockUserUseCase{getErr: userDomain.ErrInvalidID}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodGet, "/users/"+id.String(), nil)

	err := h.GetUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_GetUser_InternalError_Returns500(t *testing.T) {
	uc := &mockUserUseCase{getErr: errors.New("db error")}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodGet, "/users/"+id.String(), nil)

	err := h.GetUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- DeleteUser tests ---

func TestUserHandler_DeleteUser_Success_Returns204(t *testing.T) {
	uc := &mockUserUseCase{deleteErr: nil}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodDelete, "/users/"+id.String(), nil)

	err := h.DeleteUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUserHandler_DeleteUser_NotFound_Returns404(t *testing.T) {
	uc := &mockUserUseCase{deleteErr: userDomain.ErrUserNotFound}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodDelete, "/users/"+id.String(), nil)

	err := h.DeleteUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_DeleteUser_InvalidID_Returns404(t *testing.T) {
	uc := &mockUserUseCase{deleteErr: userDomain.ErrInvalidID}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodDelete, "/users/"+id.String(), nil)

	err := h.DeleteUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_DeleteUser_InternalError_Returns500(t *testing.T) {
	uc := &mockUserUseCase{deleteErr: errors.New("db error")}
	h := handlers.NewUserHandler(uc)

	id := openapi_types.UUID(uuid.New())
	c, rec := newEchoContext(http.MethodDelete, "/users/"+id.String(), nil)

	err := h.DeleteUser(c, id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// suppress unused import
var _ = errors.New
