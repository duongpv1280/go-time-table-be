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

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	api "gosample/internal/delivery/http/openapi"
	httpMiddleware "gosample/internal/delivery/http/middleware"
	"gosample/internal/delivery/http/handlers"
	"gosample/internal/delivery/http/validator/rules"
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

func (m *mockClassUseCaseIntegration) CreateClass(_ context.Context, _ string, _ int, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

func (m *mockClassUseCaseIntegration) UpdateClass(_ context.Context, _, _ string, _ *int, _ domainAuth.ContextPermission) (*classUseCase.ClassDTO, error) {
	return m.class_, m.err
}

// --- In-memory SQLite DB for integration tests ---

type classDBModel struct {
	ID   string `gorm:"primaryKey;type:varchar(36)"`
	Name string `gorm:"not null;uniqueIndex"`
}

func (classDBModel) TableName() string { return "classes" }

func newInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&classDBModel{})
	require.NoError(t, err)
	return db
}

// --- Router setup ---

func setupIntegrationRouter(jwtSvc domainAuth.IJWTService, classUC classUseCase.IClassUseCase) *echo.Echo {
	db := &gorm.DB{}
	return setupIntegrationRouterWithDB(jwtSvc, classUC, db)
}

func setupIntegrationRouterWithDB(jwtSvc domainAuth.IJWTService, classUC classUseCase.IClassUseCase, db *gorm.DB) *echo.Echo {
	e := echo.New()
	v := rules.NewValidator(db)
	classHandler := handlers.NewClassHandler(classUC, v)
	apiV1 := e.Group("/api/v1")
	apiV1.Use(httpMiddleware.JWTAuth(jwtSvc))
	apiV1.GET("/classes", classHandler.GetClasses)
	apiV1.GET("/classes/:classId", func(c echo.Context) error {
		return classHandler.GetClassById(c, c.Param("classId"))
	})
	apiV1.POST("/classes", classHandler.CreateClass)
	apiV1.PUT("/classes/:classId", func(c echo.Context) error {
		return classHandler.UpdateClass(c, c.Param("classId"))
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

func doRequestWithBody(e *echo.Echo, method, path, authHeader string, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
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

// TC-INT-01: POST duplicate name → 422 with fields.name
func TestIntegration_TCINT01_PostDuplicateName_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	db.Create(&classDBModel{ID: "existing-id", Name: "10A"})

	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "10A", "grade": 10}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "validation_error", *resp.Error)
	_, hasName := (*resp.Fields)["name"]
	assert.True(t, hasName, "expected fields.name in response")
}

// TC-INT-02: PUT duplicate name (other class) → 422 with fields.name
func TestIntegration_TCINT02_PutDuplicateName_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	db.Create(&classDBModel{ID: "class-1", Name: "10A"})
	db.Create(&classDBModel{ID: "class-2", Name: "11B"})

	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "10A"}
	rec := doRequestWithBody(e, http.MethodPut, "/api/v1/classes/class-2", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "validation_error", *resp.Error)
	_, hasName := (*resp.Fields)["name"]
	assert.True(t, hasName, "expected fields.name in response")
}

// TC-INT-03: PUT own name → 200 (not flagged as duplicate because ExcludeIDKey excludes itself)
func TestIntegration_TCINT03_PutOwnName_Returns200(t *testing.T) {
	db := newInMemoryDB(t)
	db.Create(&classDBModel{ID: "class-1", Name: "10A"})

	dto := makeIntegrationClassDTO("10A", 10)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{class_: &dto}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "10A"}
	rec := doRequestWithBody(e, http.MethodPut, "/api/v1/classes/class-1", "Bearer admin-token", body)

	require.Equal(t, http.StatusOK, rec.Code)
}

// TC-INT-04: No Bearer token → 401 from middleware
func TestIntegration_TCINT04_NoBearerToken_Returns401(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "10A", "grade": 10}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "", body)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-060: ADMIN JWT + valid body → POST /api/v1/classes returns 201.
func TestIntegration_TC060_PostValidClass_Returns201(t *testing.T) {
	db := newInMemoryDB(t)
	dto := makeIntegrationClassDTO("ClassA", 5)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{class_: &dto}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "ClassA", "grade": 5}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp api.Class
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ClassA", resp.Name)
}

// TC-061: ADMIN JWT + valid body → PUT /api/v1/classes/{id} returns 200.
func TestIntegration_TC061_PutRenameClass_Returns200(t *testing.T) {
	db := newInMemoryDB(t)
	db.Create(&classDBModel{ID: "class-id-1", Name: "OldName"})
	dto := makeIntegrationClassDTO("NewName", 5)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{class_: &dto}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "NewName", "grade": 5}
	rec := doRequestWithBody(e, http.MethodPut, "/api/v1/classes/class-id-1", "Bearer admin-token", body)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp api.Class
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "NewName", resp.Name)
}

// TC-064: TEACHER JWT → POST /api/v1/classes returns 401 (anti-enumeration).
func TestIntegration_TC064_PostTeacherRole_Returns401(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "teacher-1", Role: "TEACHER"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "ClassA", "grade": 5}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer teacher-token", body)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-065: STUDENT JWT → POST /api/v1/classes returns 401 (anti-enumeration).
func TestIntegration_TC065_PostStudentRole_Returns401(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "student-1", Role: "STUDENT"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "ClassA", "grade": 5}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer student-token", body)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-068: Missing name field → POST /api/v1/classes returns 422 with fields.name.
func TestIntegration_TC068_PostMissingName_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"grade": 5}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, hasName := (*resp.Fields)["name"]
	assert.True(t, hasName, "expected fields.name in response")
}

// TC-069: Missing grade field (omitted = 0, fails required + min=1) → 422.
func TestIntegration_TC069_PostMissingGrade_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "NewClass"}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, hasGrade := (*resp.Fields)["grade"]
	assert.True(t, hasGrade, "expected fields.grade in response")
}

// TC-070: grade=0 → POST /api/v1/classes returns 422 with fields.grade.
func TestIntegration_TC070_PostGradeZero_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "NewClass", "grade": 0}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, hasGrade := (*resp.Fields)["grade"]
	assert.True(t, hasGrade, "expected fields.grade in response")
}

// TC-073: Whitespace-only name → POST /api/v1/classes returns 422 with fields.name.
func TestIntegration_TC073_PostWhitespaceOnlyName_Returns422(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "   ", "grade": 5}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer admin-token", body)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp api.ValidationErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, hasName := (*resp.Fields)["name"]
	assert.True(t, hasName, "expected fields.name in response")
}

// TC-074: ADMIN JWT + usecase returns ErrUnauthorized → PUT /api/v1/classes/{id} returns 401.
func TestIntegration_TC074_PutNonexistentClass_Returns401(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "admin-1", Role: "ADMIN"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "SomeName"}
	rec := doRequestWithBody(e, http.MethodPut, "/api/v1/classes/nonexistent-id", "Bearer admin-token", body)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TC-122: TEACHER JWT, usecase returns ErrUnauthorized → POST /api/v1/classes returns 401.
func TestIntegration_TC122_PostTeacherRole_Returns401(t *testing.T) {
	db := newInMemoryDB(t)
	jwtSvc := &mockJWTServiceIntegration{
		claims: &domainAuth.JWTClaims{UserID: "teacher-1", Role: "TEACHER"},
	}
	classUC := &mockClassUseCaseIntegration{err: domainAuth.ErrUnauthorized}
	e := setupIntegrationRouterWithDB(jwtSvc, classUC, db)

	body := map[string]interface{}{"name": "ClassB", "grade": 3}
	rec := doRequestWithBody(e, http.MethodPost, "/api/v1/classes", "Bearer teacher-token", body)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// suppress unused import for errors package
var _ = errors.New
