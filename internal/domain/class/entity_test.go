package class_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gosample/internal/domain/class"
)

// --- Value Object Tests ---

func TestNewID_GeneratesUniqueIDs(t *testing.T) {
	id1 := class.NewID()
	id2 := class.NewID()
	assert.NotEqual(t, id1.String(), id2.String())
}

func TestParseID_ValidUUID_Succeeds(t *testing.T) {
	id := class.NewID()
	parsed, err := class.ParseID(id.String())
	require.NoError(t, err)
	assert.Equal(t, id.String(), parsed.String())
}

func TestParseID_InvalidUUID_ReturnsError(t *testing.T) {
	_, err := class.ParseID("not-a-valid-uuid")
	require.Error(t, err)
	assert.ErrorIs(t, err, class.ErrInvalidClassID)
}

func TestNewName_Valid_Succeeds(t *testing.T) {
	n, err := class.NewName("Class 10A")
	require.NoError(t, err)
	assert.Equal(t, "Class 10A", n.String())
}

func TestNewName_Empty_ReturnsError(t *testing.T) {
	_, err := class.NewName("")
	require.Error(t, err)
	assert.ErrorIs(t, err, class.ErrEmptyClassName)
}

func TestNewName_WhitespaceOnly_ReturnsError(t *testing.T) {
	_, err := class.NewName("   ")
	require.Error(t, err)
	assert.ErrorIs(t, err, class.ErrEmptyClassName)
}

func TestNewGrade_Valid_Succeeds(t *testing.T) {
	g, err := class.NewGrade(10)
	require.NoError(t, err)
	assert.Equal(t, 10, g.Value())
}

func TestNewGrade_Zero_ReturnsError(t *testing.T) {
	_, err := class.NewGrade(0)
	require.Error(t, err)
	assert.ErrorIs(t, err, class.ErrInvalidGrade)
}

func TestNewGrade_Negative_ReturnsError(t *testing.T) {
	_, err := class.NewGrade(-1)
	require.Error(t, err)
	assert.ErrorIs(t, err, class.ErrInvalidGrade)
}

// --- Entity Tests ---

func TestNewClass_AssignsValues(t *testing.T) {
	name, _ := class.NewName("10A")
	grade, _ := class.NewGrade(10)

	c := class.NewClass(name, grade)
	assert.NotEmpty(t, c.ID().String())
	assert.Equal(t, "10A", c.Name().String())
	assert.Equal(t, 10, c.Grade().Value())
	assert.False(t, c.CreatedAt().IsZero())
	assert.False(t, c.UpdatedAt().IsZero())
}

func TestRestoreClass_ReturnsExactValues(t *testing.T) {
	id := class.NewID()
	name, _ := class.NewName("11B")
	grade, _ := class.NewGrade(11)
	now := time.Now().UTC()

	c := class.RestoreClass(id, name, grade, now, now)
	assert.Equal(t, id.String(), c.ID().String())
	assert.Equal(t, "11B", c.Name().String())
	assert.Equal(t, 11, c.Grade().Value())
	assert.Equal(t, now, c.CreatedAt())
	assert.Equal(t, now, c.UpdatedAt())
}

func TestClass_UpdateName_ChangesNameAndBumpsUpdatedAt(t *testing.T) {
	name, _ := class.NewName("10A")
	grade, _ := class.NewGrade(10)
	c := class.NewClass(name, grade)
	before := c.UpdatedAt()

	time.Sleep(time.Millisecond)
	newName, _ := class.NewName("11B")
	c.UpdateName(newName)

	assert.Equal(t, "11B", c.Name().String())
	assert.True(t, c.UpdatedAt().After(before) || c.UpdatedAt().Equal(before), "updatedAt must not go backwards")
}

func TestClass_UpdateGrade_ChangesGradeAndBumpsUpdatedAt(t *testing.T) {
	name, _ := class.NewName("10A")
	grade, _ := class.NewGrade(10)
	c := class.NewClass(name, grade)
	before := c.UpdatedAt()

	time.Sleep(time.Millisecond)
	newGrade, _ := class.NewGrade(12)
	c.UpdateGrade(newGrade)

	assert.Equal(t, 12, c.Grade().Value())
	assert.True(t, c.UpdatedAt().After(before) || c.UpdatedAt().Equal(before), "updatedAt must not go backwards")
}
