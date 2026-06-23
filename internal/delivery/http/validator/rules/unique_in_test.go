package rules_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gosample/internal/delivery/http/validator/rules"
)

type testClass struct {
	ID   string `gorm:"primaryKey;type:varchar(36)"`
	Name string `gorm:"not null;uniqueIndex"`
}

func (testClass) TableName() string { return "classes" }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&testClass{})
	require.NoError(t, err)
	return db
}

type createClassInput struct {
	Name string `validate:"required,unique_in=classes:name"`
}

type updateClassInput struct {
	Name string `validate:"omitempty,unique_in=classes:name"`
}

func TestUniqueInValidator_ValueNotInDB_ReturnsNoError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := createClassInput{Name: "10A"}
	err := v.ValidateCtx(context.Background(), &input)

	assert.NoError(t, err)
}

func TestUniqueInValidator_ValueInDB_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "10A"})
	v := rules.NewValidator(db)

	input := createClassInput{Name: "10A"}
	err := v.ValidateCtx(context.Background(), &input)

	assert.Error(t, err)
}

func TestUniqueInValidator_ValueInDBButExcludedByExcludeIDKey_ReturnsNoError(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "10A"})
	v := rules.NewValidator(db)

	ctx := context.WithValue(context.Background(), rules.ExcludeIDKey, "id-1")
	input := updateClassInput{Name: "10A"}
	err := v.ValidateCtx(ctx, &input)

	assert.NoError(t, err)
}

func TestUniqueInValidator_ValueInDBExcludeIDKeyDifferentID_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "10A"})
	v := rules.NewValidator(db)

	ctx := context.WithValue(context.Background(), rules.ExcludeIDKey, "id-2")
	input := updateClassInput{Name: "10A"}
	err := v.ValidateCtx(ctx, &input)

	assert.Error(t, err)
}

func TestUniqueInValidator_EmptyNameWithOmitempty_ReturnsNoError(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "10A"})
	v := rules.NewValidator(db)

	input := updateClassInput{Name: ""}
	err := v.ValidateCtx(context.Background(), &input)

	assert.NoError(t, err)
}

func TestUniqueInValidator_ContextWithoutExcludeIDKey_StillWorks(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "10A"})
	v := rules.NewValidator(db)

	input := createClassInput{Name: "10A"}
	err := v.ValidateCtx(context.Background(), &input)

	assert.Error(t, err)
}

// notblank validator tests

type notBlankInput struct {
	Name string `validate:"notblank"`
}

func TestNotBlankValidator_NonEmptyString_ReturnsNoError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := notBlankInput{Name: "hello"}
	err := v.ValidateCtx(context.Background(), &input)

	assert.NoError(t, err)
}

func TestNotBlankValidator_WhitespaceOnly_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := notBlankInput{Name: "   "}
	err := v.ValidateCtx(context.Background(), &input)

	assert.Error(t, err)
}

func TestNotBlankValidator_EmptyString_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := notBlankInput{Name: ""}
	err := v.ValidateCtx(context.Background(), &input)

	assert.Error(t, err)
}

// TC-009: DB error (table does not exist) causes the validator to return false → validation error.
func TestUniqueInValidator_DBError_ReturnsFalse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// intentionally skip AutoMigrate so "nonexistent_table" does not exist

	type inputWithNonexistentTable struct {
		Name string `validate:"unique_in=nonexistent_table:name"`
	}

	v := rules.NewValidator(db)
	input := inputWithNonexistentTable{Name: "anything"}
	validationErr := v.ValidateCtx(context.Background(), &input)

	require.Error(t, validationErr)
}
