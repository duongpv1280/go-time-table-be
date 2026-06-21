package base

import "context"

type IRepository[T any, ID any] interface {
	Create(ctx context.Context, entity T) error
	FindByID(ctx context.Context, id ID) (T, error)
	FindAll(ctx context.Context) ([]T, error)
}
