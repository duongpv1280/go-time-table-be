package subject

import (
	"context"
	"errors"

	"gosample/internal/domain/base"
)

var (
	ErrSubjectNotFound      = errors.New("subject not found")
	ErrSubjectAlreadyExists = errors.New("subject already exists")
)

type ISubjectRepository interface {
	base.IRepository[*Subject, ID]
	Delete(ctx context.Context, id ID) error
}
