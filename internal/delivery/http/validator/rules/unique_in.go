package rules

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
)

type contextKey string

const ExcludeIDKey = contextKey("unique_in_exclude_id")

func (v *Validator) uniqueInValidator(ctx context.Context, fl validator.FieldLevel) bool {
	parts := strings.SplitN(fl.Param(), ":", 2)
	if len(parts) != 2 {
		return false
	}
	table := parts[0]
	column := parts[1]

	var count int64
	q := v.db.WithContext(ctx).Table(table).Where(column+" = ?", fl.Field().String())
	if excludeID, ok := ctx.Value(ExcludeIDKey).(string); ok && excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	result := q.Count(&count)
	if result.Error != nil {
		return false
	}
	return count == 0
}
