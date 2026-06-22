package validator

import "context"

type IValidator interface {
	ValidateCtx(ctx context.Context, s interface{}) error
}
