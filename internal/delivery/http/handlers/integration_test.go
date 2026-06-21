package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "gosample/internal/delivery/http"
	httpMiddleware "gosample/internal/delivery/http/middleware"
	"gosample/internal/delivery/http/handlers"
	domainAuth "gosample/internal/domain/auth"
	classUseCase "gosample/internal/usecase/class"
)

// --- Integration mock: JWT service ---

type mockJWTServiceIntegration struct {
	claims *domainAuth.JWTClaims
	err    error
}

func (m *mockJWTServiceIntegration) Sign(_ context.Context, _, _ string) (string, error) {
	return "mock-token", nil
}

func (m *mockJWTServiceIntegration) Verify(_ context.Context, _ string) (*domainAuth.JWTClaims, error) {
	return m.claims, m.err
}

// --- Integration mock: class usecase ---

type mockClassUseCaseIntegration struct {
	classes []classUseCase.ClassDTO
	class_  *classUseCase.ClassDTO
	err     error
}

func (m *mockClassUseCaseIntegration) GetClasses(_ context.Context, _ domainAuth.ContextPermission) ([]classUseCase.ClassDTO, error) {
	return m.classes, m.err
}

func (m *mockClassUseCaseIntegration) GetClassByID(_ context.Context, _ string, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

// --- Router setup ---

func setupIntegrationRouter(jwtSvc domainAuth.IJWTService, classUC classUseCase.IClassUseCase) *echo.Echo {
	e := echo.New()
	classHandler := handlers.NewClassHandler(classUC)
	apiV1 := e.Group("/api/v1")
	apiV1.Use(httpMiddleware.JWTAuth(jwtSvc))
	apiV1.GET("/classes", classHandler.GetClasses)
	apiV1.GET("/classes/:classId", func(c echo.Context) error {
		return classHandler.GetClassById(c, c.Param("classId"))
	})
	return e
}

func makeIntegrationClassDTO(name string, grade int) classUseCase.ClassDTO {
	now := time.Now().UTC()
	return classUseCase.ClassDTO{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Name:      name,
		Grade:     grade,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func doRequest(e *echo.Echo, method, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TC-058: Middleware applies — request without Authorization header returns 401.
func TestIntegration_TC058_NoAuthHeader_Returns401(t *testing.T) {
	jwtSvc := &mockJWTServiceIntegration{}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes", "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-050: Full class flow — valid JWT with ADMIN role → GET /api/v1/classes returns 200 with data array.
func TestIntegration_TC050_AdminValidJWT_GetClasses_Returns200(t *testing.T) {
	dto := makeIntegrationClassDTO("10A", 10)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{classes: []classUseCase.ClassDTO{dto}}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes", "Bearer valid-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body api.ListClassesResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, "10A", body.Data[0].Name)
}

// TC-051: Teacher scope — JWT with TEACHER role → GET /api/v1/classes returns classes for that teacher.
func TestIntegration_TC051_TeacherValidJWT_GetClasses_Returns200(t *testing.T) {
	dto := makeIntegrationClassDTO("11B", 11)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "teacher-1", Role: "TEACHER"},
	}
	classUC := &mockClassUseCaseIntegration{classes: []classUseCase.ClassDTO{dto}}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes", "Bearer teacher-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body api.ListClassesResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, "11B", body.Data[0].Name)
}

// TC-052: Student scope — JWT with STUDENT role → GET /api/v1/classes returns single-class list.
func TestIntegration_TC052_StudentValidJWT_GetClasses_ReturnsSingleClass(t *testing.T) {
	dto := makeIntegrationClassDTO("12C", 12)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "student-1", Role: "STUDENT"},
	}
	classUC := &mockClassUseCaseIntegration{classes: []classUseCase.ClassDTO{dto}}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes", "Bearer student-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body api.ListClassesResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
}

// TC-053: Admin gets class by ID → GET /api/v1/classes/{id} returns 200.
func TestIntegration_TC053_AdminGetClassByID_Returns200(t *testing.T) {
	dto := makeIntegrationClassDTO("10A", 10)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{class_: &dto}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Bearer admin-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body api.Class
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "10A", body.Name)
}

// TC-054: Class not found returns 401 (not 404) → GET /api/v1/classes/{nonexistent-id} returns 401.
func TestIntegration_TC054_ClassNotFound_Returns401(t *testing.T) {
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Bearer admin-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-055: Teacher 401 for class they don't teach → GET /api/v1/classes/{id} returns 401.
func TestIntegration_TC055_Teacher_ClassTheyDontTeach_Returns401(t *testing.T) {
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "teacher-1", Role: "TEACHER"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes/cccccccc-cccc-cccc-cccc-cccccccccccc", "Bearer teacher-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-056: Student 401 for wrong class → GET /api/v1/classes/{id} returns 401.
func TestIntegration_TC056_Student_WrongClass_Returns401(t *testing.T) {
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "student-1", Role: "STUDENT"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes/dddddddd-dddd-dddd-dddd-dddddddddddd", "Bearer student-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-057: Student 200 for own class → GET /api/v1/classes/{id} returns 200.
func TestIntegration_TC057_Student_OwnClass_Returns200(t *testing.T) {
	dto := makeIntegrationClassDTO("10A", 10)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "student-1", Role: "STUDENT"},
	}
	classUC := &mockClassUseCaseIntegration{class_: &dto}
	e := setupIntegrationRouter(jwtSvc, classUC)

	rec := doRequest(e, http.MethodGet, "/api/v1/classes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Bearer student-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body api.Class
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "10A", body.Name)
}

// suppress unused import for errors package
var _ = errors.New
