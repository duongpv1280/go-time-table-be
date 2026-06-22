package rules

import (
	"context"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	httpvalidator "gosample/internal/delivery/http/validator"
)

type Validator struct {
	validate *validator.Validate
	db       *gorm.DB
}

func NewValidator(db *gorm.DB) httpvalidator.IValidator {
	v := &Validator{
		validate: validator.New(),
		db:       db,
	}
	v.registerCustomValidations()
	return v
}

func (v *Validator) registerCustomValidations() {
	v.validate.RegisterValidationCtx("unique_in", v.uniqueInValidator)
	v.validate.RegisterValidation("notblank", notBlankValidator)
}

func (v *Validator) ValidateCtx(ctx context.Context, s interface{}) error {
	return v.validate.StructCtx(ctx, s)
}
