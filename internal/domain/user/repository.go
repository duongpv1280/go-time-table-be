package user

import (
	"context"
	"errors"

	"gosample/internal/domain/base"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists with this email")
)

type IUserRepository interface {
	base.IRepository[*User, ID]
	Delete(ctx context.Context, id ID) error
}
