package class_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	classDomain "gosample/internal/domain/class"
	userDomain "gosample/internal/domain/user"
	classUseCase "gosample/internal/usecase/class"
)

// --- Mock ---

type mockClassRepository struct {
	classes        []*classDomain.Class
	findByIDErr    error
	teacherClasses []*classDomain.Class
	teacherErr     error
	studentClass   *classDomain.Class
	studentErr     error
	createErr      error
	updateErr      error
}

func (m *mockClassRepository) Create(_ context.Context, _ *classDomain.Class) error {
	return m.createErr
}

func (m *mockClassRepository) FindByID(_ context.Context, id classDomain.ID) (*classDomain.Class, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	for _, c := range m.classes {
		if c.ID().String() == id.String() {
			return c, nil
		}
	}
	return nil, classDomain.ErrClassNotFound
}

func (m *mockClassRepository) FindAll(_ context.Context) ([]*classDomain.Class, error) {
	return m.classes, nil
}

func (m *mockClassRepository) FindByTeacherUserID(_ context.Context, _ string) ([]*classDomain.Class, error) {
	return m.teacherClasses, m.teacherErr
}

func (m *mockClassRepository) FindByStudentUserID(_ context.Context, _ string) (*classDomain.Class, error) {
	return m.studentClass, m.studentErr
}

func (m *mockClassRepository) Update(_ context.Context, _ *classDomain.Class) error {
	return m.updateErr
}

// --- Helpers ---

func makeClass(name string, grade int) *classDomain.Class {
	n, _ := classDomain.NewName(name)
	g, _ := classDomain.NewGrade(grade)
	return classDomain.NewClass(n, g)
}

func restoreClass(id, name string, grade int) *classDomain.Class {
	parsedID, _ := classDomain.ParseID(id)
	n, _ := classDomain.NewName(name)
	g, _ := classDomain.NewGrade(grade)
	now := time.Now().UTC()
	return classDomain.RestoreClass(parsedID, n, g, now, now)
}

// --- GetClasses tests ---

func TestGetClasses_Admin_ReturnsAll(t *testing.T) {
	class1 := makeClass("10A", 10)
	class2 := makeClass("11B", 11)
	repo := &mockClassRepository{classes: []*classDomain.Class{class1, class2}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	result, err := uc.GetClasses(context.Background(), perm)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetClasses_Teacher_ReturnsScopedClasses(t *testing.T) {
	class1 := makeClass("10A", 10)
	repo := &mockClassRepository{teacherClasses: []*classDomain.Class{class1}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	result, err := uc.GetClasses(context.Background(), perm)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "10A", result[0].Name)
}

func TestGetClasses_Student_ReturnsSingleClassInSlice(t *testing.T) {
	class1 := makeClass("10A", 10)
	repo := &mockClassRepository{studentClass: class1}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	result, err := uc.GetClasses(context.Background(), perm)

	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGetClasses_UnknownRole_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "x", Role: "UNKNOWN"}
	_, err := uc.GetClasses(context.Background(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClasses_Student_NoClass_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{studentErr: classDomain.ErrClassNotFound}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	_, err := uc.GetClasses(context.Background(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

// --- GetClassByID tests ---

func TestGetClassByID_Admin_ValidClass_ReturnsDTO(t *testing.T) {
	class1 := makeClass("10A", 10)
	repo := &mockClassRepository{classes: []*classDomain.Class{class1}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	result, err := uc.GetClassByID(context.Background(), class1.ID().String(), perm)

	require.NoError(t, err)
	assert.Equal(t, "10A", result.Name)
}

func TestGetClassByID_Admin_NotFound_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{classes: []*classDomain.Class{}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	id := classDomain.NewID()
	_, err := uc.GetClassByID(context.Background(), id.String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClassByID_InvalidFormat_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	_, err := uc.GetClassByID(context.Background(), "not-a-uuid", perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClassByID_Teacher_HasAccess_ReturnsDTO(t *testing.T) {
	class1 := makeClass("10A", 10)
	repo := &mockClassRepository{teacherClasses: []*classDomain.Class{class1}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	result, err := uc.GetClassByID(context.Background(), class1.ID().String(), perm)

	require.NoError(t, err)
	assert.Equal(t, "10A", result.Name)
}

func TestGetClassByID_Teacher_NoAccess_ReturnsUnauthorized(t *testing.T) {
	class1 := makeClass("10A", 10)
	class2 := makeClass("11B", 11)
	repo := &mockClassRepository{teacherClasses: []*classDomain.Class{class1}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	_, err := uc.GetClassByID(context.Background(), class2.ID().String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClassByID_Student_HasAccess_ReturnsDTO(t *testing.T) {
	class1 := makeClass("10A", 10)
	repo := &mockClassRepository{studentClass: class1}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	result, err := uc.GetClassByID(context.Background(), class1.ID().String(), perm)

	require.NoError(t, err)
	assert.Equal(t, "10A", result.Name)
}

func TestGetClassByID_Student_WrongClass_ReturnsUnauthorized(t *testing.T) {
	class1 := makeClass("10A", 10)
	class2 := makeClass("11B", 11)
	repo := &mockClassRepository{studentClass: class1}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	_, err := uc.GetClassByID(context.Background(), class2.ID().String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClassByID_Student_NotInAnyClass_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{studentErr: classDomain.ErrClassNotFound}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	id := classDomain.NewID()
	_, err := uc.GetClassByID(context.Background(), id.String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClassByID_UnknownRole_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "x", Role: "UNKNOWN"}
	id := classDomain.NewID()
	_, err := uc.GetClassByID(context.Background(), id.String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestGetClasses_Student_DBError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db connection failed")
	repo := &mockClassRepository{studentErr: dbErr}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	_, err := uc.GetClasses(context.Background(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, domainAuth.ErrUnauthorized))
}

func TestGetClassByID_Teacher_FindByTeacherUserIDError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db timeout")
	repo := &mockClassRepository{teacherErr: dbErr}
	uc := classUseCase.NewClassUseCase(repo)

	id := classDomain.NewID()
	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	_, err := uc.GetClassByID(context.Background(), id.String(), perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

// --- CreateClass tests ---

func TestCreateClass_Admin_Success(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	result, err := uc.CreateClass(context.Background(), "10A", 10, perm)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "10A", result.Name)
	assert.Equal(t, 10, result.Grade)
}

func TestCreateClass_Teacher_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	_, err := uc.CreateClass(context.Background(), "10A", 10, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestCreateClass_Student_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "student-1", Role: userDomain.RoleStudent}
	_, err := uc.CreateClass(context.Background(), "10A", 10, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestCreateClass_EmptyName_ReturnsDomainError(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	_, err := uc.CreateClass(context.Background(), "", 10, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, classDomain.ErrEmptyClassName)
}

func TestCreateClass_InvalidGrade_ReturnsDomainError(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	_, err := uc.CreateClass(context.Background(), "10A", 0, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, classDomain.ErrInvalidGrade)
}

// --- UpdateClass tests ---

func TestUpdateClass_Admin_Success(t *testing.T) {
	existing := restoreClass("11111111-1111-1111-1111-111111111111", "10A", 10)
	repo := &mockClassRepository{classes: []*classDomain.Class{existing}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	newGrade := 11
	result, err := uc.UpdateClass(context.Background(), "11111111-1111-1111-1111-111111111111", "11B", &newGrade, perm)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "11B", result.Name)
	assert.Equal(t, 11, result.Grade)
}

func TestUpdateClass_Teacher_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "teacher-1", Role: userDomain.RoleTeacher}
	_, err := uc.UpdateClass(context.Background(), "11111111-1111-1111-1111-111111111111", "11B", nil, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestUpdateClass_ClassNotFound_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{classes: []*classDomain.Class{}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	id := classDomain.NewID()
	_, err := uc.UpdateClass(context.Background(), id.String(), "11B", nil, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestUpdateClass_InvalidClassID_ReturnsUnauthorized(t *testing.T) {
	repo := &mockClassRepository{}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	_, err := uc.UpdateClass(context.Background(), "not-a-uuid", "11B", nil, perm)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrUnauthorized)
}

func TestUpdateClass_OnlyName_UpdatesName(t *testing.T) {
	existing := restoreClass("11111111-1111-1111-1111-111111111111", "10A", 10)
	repo := &mockClassRepository{classes: []*classDomain.Class{existing}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	result, err := uc.UpdateClass(context.Background(), "11111111-1111-1111-1111-111111111111", "NewName", nil, perm)

	require.NoError(t, err)
	assert.Equal(t, "NewName", result.Name)
	assert.Equal(t, 10, result.Grade)
}

func TestUpdateClass_OnlyGrade_UpdatesGrade(t *testing.T) {
	existing := restoreClass("11111111-1111-1111-1111-111111111111", "10A", 10)
	repo := &mockClassRepository{classes: []*classDomain.Class{existing}}
	uc := classUseCase.NewClassUseCase(repo)

	perm := domainAuth.ContextPermission{UserID: "admin-1", Role: userDomain.RoleAdmin}
	newGrade := 12
	result, err := uc.UpdateClass(context.Background(), "11111111-1111-1111-1111-111111111111", "", &newGrade, perm)

	require.NoError(t, err)
	assert.Equal(t, "10A", result.Name)
	assert.Equal(t, 12, result.Grade)
}

// suppress unused import
var _ = errors.New
