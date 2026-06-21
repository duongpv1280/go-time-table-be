package db

import (
	"context"
	"errors"

	"gosample/internal/domain/subject"

	"gorm.io/gorm"
)

type gormSubjectRepository struct {
	db *gorm.DB
}

func NewGormSubjectRepository(db *gorm.DB) subject.ISubjectRepository {
	return &gormSubjectRepository{db: db}
}

func (r *gormSubjectRepository) Create(ctx context.Context, s *subject.Subject) error {
	model := SubjectFromDomain(s)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *gormSubjectRepository) FindByID(ctx context.Context, id subject.ID) (*subject.Subject, error) {
	var model SubjectModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, subject.ErrSubjectNotFound
		}
		return nil, err
	}
	return model.ToSubjectDomain()
}

func (r *gormSubjectRepository) FindAll(ctx context.Context) ([]*subject.Subject, error) {
	var models []SubjectModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	subjects := make([]*subject.Subject, len(models))
	for i, m := range models {
		s, err := m.ToSubjectDomain()
		if err != nil {
			return nil, err
		}
		subjects[i] = s
	}
	return subjects, nil
}

func (r *gormSubjectRepository) Delete(ctx context.Context, id subject.ID) error {
	result := r.db.WithContext(ctx).Delete(&SubjectModel{}, "id = ?", id.String())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return subject.ErrSubjectNotFound
	}
	return nil
}
