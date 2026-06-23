package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	httpMiddleware "gosample/internal/delivery/http/middleware"
	"gosample/internal/delivery/http/handlers"
	classDomain "gosample/internal/domain/class"
	classUseCase "gosample/internal/usecase/class"
)

// --- Mock Class Use Case ---

type mockClassUseCase struct {
	classes []classUseCase.ClassDTO
	class_  *classUseCase.ClassDTO
	err     error
}

func (m *mockClassUseCase) GetClasses(_ context.Context, _ domainAuth.ContextPermission) ([]classUseCase.ClassDTO, error) {
	return m.classes, m.err
}

func (m *mockClassUseCase) GetClassByID(_ context.Context, _ string, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

func (m *mockClassUseCase) CreateClass(_ context.Context, _ string, _ int, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

func (m *mockClassUseCase) UpdateClass(_ context.Context, _, _ string, _ *int, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

// --- Mock Validator ---

type mockValidator struct {
	err error
}

func (m *mockValidator) ValidateCtx(_ context.Context, _ interface{}) error {
	return m.err
}

// --- Helpers ---

func newContextWithPermission(method, path string, perm *domainAuth.ContextPermission) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if perm != nil {
		c.Set(httpMiddleware.PermissionContextKey, *perm)
	}
	return c, rec
}

func makeClassDTO(name string, grade int) classUseCase.ClassDTO {
	now := time.Now().UTC()
	return classUseCase.ClassDTO{
		ID:        "11111111-1111-1111-1111-111111111111",
		Name:      name,
		Grade:     grade,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// --- GetClasses handler tests ---

func TestClassHandler_GetClasses_WithPermission_Returns200(t *testing.T) {
	dto := makeClassDTO("10A", 10)
	uc := &mockClassUseCase{classes: []classUseCase.ClassDTO{dto}}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", &perm)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, ok := body["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 1)
}

func TestClassHandler_GetClasses_NoPermission_Returns401(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", nil)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClasses_UseCaseUnauthorized_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "x", Role: "UNKNOWN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", &perm)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClasses_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", &perm)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- GetClassById handler tests ---

func TestClassHandler_GetClassById_ValidAccess_Returns200(t *testing.T) {
	dto := makeClassDTO("10A", 10)
	uc := &mockClassUseCase{class_: &dto}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "10A", body["name"])
}

func TestClassHandler_GetClassById_NoPermission_Returns401(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", nil)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClassById_Unauthorized_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: "STUDENT"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClassById_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestClassHandler_GetClassById_ErrClassNotFound_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: classDomain.ErrClassNotFound}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Helper: echo context with JSON body ---

func newContextWithJSONBody(method, path string, body string, perm *domainAuth.ContextPermission) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if perm != nil {
		c.Set(httpMiddleware.PermissionContextKey, *perm)
	}
	return c, rec
}

// --- CreateClass handler unit tests ---

func TestClassHandler_CreateClass_NoPermission_Returns401(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	c, rec := newContextWithPermission(http.MethodPost, "/api/v1/classes", nil)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_CreateClass_BadRequestBody_Returns400(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{invalid-json}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestClassHandler_CreateClass_ValidationError_Returns422(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{err: assert.AnError})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{"name":"X","grade":5}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestClassHandler_CreateClass_ErrUnauthorized_Returns401(t *testing.T) {
	dto := makeClassDTO("10A", 10)
	uc := &mockClassUseCase{class_: &dto, err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: "TEACHER"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{"name":"10A","grade":10}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_CreateClass_DomainError_Returns422(t *testing.T) {
	uc := &mockClassUseCase{err: classDomain.ErrEmptyClassName}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{"name":"","grade":5}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestClassHandler_CreateClass_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{"name":"10A","grade":10}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestClassHandler_CreateClass_Success_Returns201(t *testing.T) {
	dto := makeClassDTO("10A", 10)
	uc := &mockClassUseCase{class_: &dto}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPost, "/api/v1/classes", `{"name":"10A","grade":10}`, &perm)

	err := h.CreateClass(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// --- UpdateClass handler unit tests ---

func TestClassHandler_UpdateClass_NoPermission_Returns401(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	c, rec := newContextWithPermission(http.MethodPut, "/api/v1/classes/some-id", nil)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_UpdateClass_BadRequestBody_Returns400(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/some-id", `{bad-json}`, &perm)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestClassHandler_UpdateClass_ValidationError_Returns422(t *testing.T) {
	uc := &mockClassUseCase{}
	h := handlers.NewClassHandler(uc, &mockValidator{err: assert.AnError})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/some-id", `{"name":"10A"}`, &perm)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestClassHandler_UpdateClass_ErrUnauthorized_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/nonexistent", `{"name":"10A"}`, &perm)

	err := h.UpdateClass(c, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_UpdateClass_DomainError_Returns422(t *testing.T) {
	uc := &mockClassUseCase{err: classDomain.ErrEmptyClassName}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/some-id", `{"name":""}`, &perm)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestClassHandler_UpdateClass_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/some-id", `{"name":"10A"}`, &perm)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestClassHandler_UpdateClass_Success_Returns200(t *testing.T) {
	dto := makeClassDTO("10A-updated", 10)
	uc := &mockClassUseCase{class_: &dto}
	h := handlers.NewClassHandler(uc, &mockValidator{})

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithJSONBody(http.MethodPut, "/api/v1/classes/some-id", `{"name":"10A-updated"}`, &perm)

	err := h.UpdateClass(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- CombinedHandler ---

func TestNewCombinedHandler_ReturnsNonNil(t *testing.T) {
	userH := handlers.NewUserHandler(&mockUserUseCase{})
	authH := handlers.NewAuthHandler(&mockGoogleAuthUseCase{})
	classH := handlers.NewClassHandler(&mockClassUseCase{}, &mockValidator{})
	combined := handlers.NewCombinedHandler(userH, authH, classH)
	assert.NotNil(t, combined)
}
