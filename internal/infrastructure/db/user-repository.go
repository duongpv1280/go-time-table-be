package db

import (
	"context"
	"errors"
	"strings"

	"gosample/internal/domain/user"

	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) user.IUserRepository {
	return &gormUserRepository{
		db: db,
	}
}

func (r *gormUserRepository) Create(ctx context.Context, domainUser *user.User) error {
	model := FromDomain(domainUser)
	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key") {
			return user.ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return model.ToDomain()
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("email = ?", email.String()).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return model.ToDomain()
}

func (r *gormUserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	var models []UserModel
	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, err
	}

	domainUsers := make([]*user.User, len(models))
	for i, m := range models {
		du, err := m.ToDomain()
		if err != nil {
			return nil, err
		}
		domainUsers[i] = du
	}

	return domainUsers, nil
}

func (r *gormUserRepository) Delete(ctx context.Context, id user.ID) error {
	result := r.db.WithContext(ctx).Delete(&UserModel{}, "id = ?", id.String())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}
	return nil
}
