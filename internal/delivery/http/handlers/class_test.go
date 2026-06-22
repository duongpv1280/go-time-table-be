package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	h := handlers.NewClassHandler(uc)

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
	h := handlers.NewClassHandler(uc)

	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", nil)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClasses_UseCaseUnauthorized_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc)

	perm := domainAuth.ContextPermission{UserID: "x", Role: "UNKNOWN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes", &perm)

	err := h.GetClasses(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClasses_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc)

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
	h := handlers.NewClassHandler(uc)

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
	h := handlers.NewClassHandler(uc)

	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", nil)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClassById_Unauthorized_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: domainAuth.ErrUnauthorized}
	h := handlers.NewClassHandler(uc)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: "STUDENT"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClassHandler_GetClassById_InternalError_Returns500(t *testing.T) {
	uc := &mockClassUseCase{err: assert.AnError}
	h := handlers.NewClassHandler(uc)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestClassHandler_GetClassById_ErrClassNotFound_Returns401(t *testing.T) {
	uc := &mockClassUseCase{err: classDomain.ErrClassNotFound}
	h := handlers.NewClassHandler(uc)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: "ADMIN"}
	c, rec := newContextWithPermission(http.MethodGet, "/api/v1/classes/some-id", &perm)

	err := h.GetClassById(c, "some-id")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
