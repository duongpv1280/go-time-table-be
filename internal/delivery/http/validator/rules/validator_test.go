package rules_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gosample/internal/delivery/http/validator"
	"gosample/internal/delivery/http/validator/rules"
)

type validStruct struct {
	Name  string `validate:"required"`
	Grade int    `validate:"required,min=1"`
}

type invalidStruct struct {
	Name  string `validate:"required"`
	Grade int    `validate:"required,min=10"`
}

type malformedUniqueInStruct struct {
	Name string `validate:"unique_in=nocolon"`
}

func TestNewValidator_ReturnsNonNilIValidator(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	assert.NotNil(t, v)
	_, ok := v.(validator.IValidator)
	assert.True(t, ok)
}

func TestValidateCtx_ValidStruct_ReturnsNoError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := validStruct{Name: "10A", Grade: 5}
	err := v.ValidateCtx(context.Background(), &input)

	assert.NoError(t, err)
}

func TestValidateCtx_InvalidStruct_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := invalidStruct{Name: "10A", Grade: 5}
	err := v.ValidateCtx(context.Background(), &input)

	require.Error(t, err)
}

func TestValidateCtx_UniqueInTag_IsRegisteredAndFires(t *testing.T) {
	db := newTestDB(t)
	db.Create(&testClass{ID: "id-1", Name: "ExistingName"})
	v := rules.NewValidator(db)

	input := createClassInput{Name: "ExistingName"}
	err := v.ValidateCtx(context.Background(), &input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestValidateCtx_MalformedUniqueInParam_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	v := rules.NewValidator(db)

	input := malformedUniqueInStruct{Name: "anything"}
	err := v.ValidateCtx(context.Background(), &input)

	require.Error(t, err)
}
