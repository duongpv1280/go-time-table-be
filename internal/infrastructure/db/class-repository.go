package db

import (
	"context"
	"errors"

	"gorm.io/gorm"

	classDomain "gosample/internal/domain/class"
)

type gormClassRepository struct {
	db *gorm.DB
}

func NewGormClassRepository(db *gorm.DB) classDomain.IClassRepository {
	return &gormClassRepository{db: db}
}

func (r *gormClassRepository) Create(ctx context.Context, c *classDomain.Class) error {
	model := ClassFromDomain(c)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *gormClassRepository) FindByID(ctx context.Context, id classDomain.ID) (*classDomain.Class, error) {
	var model ClassModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, classDomain.ErrClassNotFound
		}
		return nil, err
	}
	return model.ToClassDomain()
}

func (r *gormClassRepository) FindAll(ctx context.Context) ([]*classDomain.Class, error) {
	var models []ClassModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	return toClassDomains(models)
}

func (r *gormClassRepository) FindByTeacherUserID(ctx context.Context, userID string) ([]*classDomain.Class, error) {
	var teacher TeacherModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&teacher).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*classDomain.Class{}, nil
		}
		return nil, err
	}

	var classSubjects []ClassSubjectModel
	if err := r.db.WithContext(ctx).Where("teacher_id = ?", teacher.ID).Find(&classSubjects).Error; err != nil {
		return nil, err
	}
	if len(classSubjects) == 0 {
		return []*classDomain.Class{}, nil
	}

	classIDs := make([]string, len(classSubjects))
	for i, cs := range classSubjects {
		classIDs[i] = cs.ClassID
	}

	var models []ClassModel
	if err := r.db.WithContext(ctx).Where("id IN ?", classIDs).Find(&models).Error; err != nil {
		return nil, err
	}
	return toClassDomains(models)
}

func (r *gormClassRepository) FindByStudentUserID(ctx context.Context, userID string) (*classDomain.Class, error) {
	var student StudentModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&student).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, classDomain.ErrClassNotFound
		}
		return nil, err
	}

	var model ClassModel
	err = r.db.WithContext(ctx).First(&model, "id = ?", student.ClassID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, classDomain.ErrClassNotFound
		}
		return nil, err
	}
	return model.ToClassDomain()
}

func toClassDomains(models []ClassModel) ([]*classDomain.Class, error) {
	classes := make([]*classDomain.Class, len(models))
	for i, m := range models {
		c, err := m.ToClassDomain()
		if err != nil {
			return nil, err
		}
		classes[i] = c
	}
	return classes, nil
}
