package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infraAuth "gosample/internal/infrastructure/auth"
)

func TestPermissionService_GetPermissionsForRole_Admin(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "ADMIN")
	require.NoError(t, err)
	assert.Contains(t, perms, "users")
	assert.Contains(t, perms, "classes")
	assert.Contains(t, perms, "subjects")
	assert.Contains(t, perms, "teachers")
	assert.Contains(t, perms, "slots")
}

func TestPermissionService_GetPermissionsForRole_Admin_TopLevelPermissionsAreStrings(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "ADMIN")
	require.NoError(t, err)

	for _, key := range []string{"users", "classes", "subjects", "teachers", "slots"} {
		val, ok := perms[key]
		require.True(t, ok, "expected key %q in admin perms", key)
		perm, isString := val.(string)
		assert.True(t, isString, "expected string value for %q, got %T", key, val)
		assert.Equal(t, "write", perm, "admin should have write on %q", key)
	}
}

func TestPermissionService_GetPermissionsForRole_Teacher(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "TEACHER")
	require.NoError(t, err)
	assert.Contains(t, perms, "classes")
	assert.Contains(t, perms, "teachers")
	assert.Contains(t, perms, "slots")
}

func TestPermissionService_GetPermissionsForRole_Teacher_ClassesIsTopLevel(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "TEACHER")
	require.NoError(t, err)

	val, ok := perms["classes"]
	require.True(t, ok)
	perm, isString := val.(string)
	assert.True(t, isString, "expected string value for classes, got %T", val)
	assert.Equal(t, "read", perm)
}

func TestPermissionService_GetPermissionsForRole_Student(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "STUDENT")
	require.NoError(t, err)
	assert.Contains(t, perms, "classes")
}

func TestPermissionService_GetPermissionsForRole_Student_ClassesIsTopLevelRead(t *testing.T) {
	// STUDENT has /api/v1/classes (top-level GET) so classes becomes "read" string.
	// The nested /api/v1/classes/:id/subjects and /api/v1/classes/:id/teachers are
	// subsumed because the top-level "classes" key already exists in the result.
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "STUDENT")
	require.NoError(t, err)

	val, ok := perms["classes"]
	require.True(t, ok)
	perm, isString := val.(string)
	assert.True(t, isString, "expected string for student classes (top-level subsumes nested), got %T", val)
	assert.Equal(t, "read", perm)
}

func TestPermissionService_GetPermissionsForRole_UnknownRole_ReturnsEmptyMap(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "JANITOR")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestPermissionService_GetPermissionsForRole_EmptyRole_ReturnsEmptyMap(t *testing.T) {
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestPermissionService_GetPermissionsForRole_Admin_TopLevelSubsumesNested(t *testing.T) {
	// ADMIN has /api/v1/classes (top-level), so classes should be a string "write",
	// NOT a nested map. Top-level subsumes nested per buildPermissions logic.
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "ADMIN")
	require.NoError(t, err)

	val, ok := perms["classes"]
	require.True(t, ok)
	_, isString := val.(string)
	assert.True(t, isString, "ADMIN classes should be string (top-level subsumes nested), got %T", val)
}

func TestPermissionService_GetPermissionsForRole_Teacher_SlotsIsNested(t *testing.T) {
	// TEACHER has /api/v1/slots/:slot_id (nested, with :param) — nonParamSegments returns ["slots"]
	// so it becomes top-level string "write"
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "TEACHER")
	require.NoError(t, err)

	val, ok := perms["slots"]
	require.True(t, ok)
	perm, isString := val.(string)
	assert.True(t, isString, "expected string for slots, got %T", val)
	assert.Equal(t, "write", perm)
}

func TestPermissionService_GetPermissionsForRole_Teacher_TeachersIsTopLevel(t *testing.T) {
	// TEACHER has /api/v1/teachers (GET) and /api/v1/teachers/:teacher_id/subjects (GET, nested)
	// teachers appears as top-level first → subsumes the nested entry
	svc := infraAuth.NewPermissionService()
	perms, err := svc.GetPermissionsForRole(context.Background(), "TEACHER")
	require.NoError(t, err)

	val, ok := perms["teachers"]
	require.True(t, ok)
	_, isString := val.(string)
	assert.True(t, isString, "TEACHER teachers should be string (top-level subsumes nested), got %T", val)
}
