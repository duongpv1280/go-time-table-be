package db

import (
	"context"

	"gosample/internal/domain/user"

	"gorm.io/gorm"
)

type GormCasbinRepository struct {
	db *gorm.DB
}

func NewGormCasbinRepository(db *gorm.DB) user.IRoleRepository {
	return &GormCasbinRepository{db: db}
}

func (r *GormCasbinRepository) AddRoleForUser(ctx context.Context, userID, role string) error {
	rule := &CasbinRuleModel{
		Ptype: "g",
		V0:    userID,
		V1:    role,
	}
	return r.db.WithContext(ctx).Create(rule).Error
}
