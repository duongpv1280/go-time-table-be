package class

import (
	"context"

	"gosample/internal/domain/base"
)

type IClassRepository interface {
	base.IRepository[*Class, ID]
	FindByTeacherUserID(ctx context.Context, userID string) ([]*Class, error)
	FindByStudentUserID(ctx context.Context, userID string) (*Class, error)
}
