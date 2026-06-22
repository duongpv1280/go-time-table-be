package db

import (
	"time"

	classDomain "gosample/internal/domain/class"
)

type ClassModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	Name      string    `gorm:"not null;uniqueIndex"`
	Grade     int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (ClassModel) TableName() string { return "classes" }

func ClassFromDomain(c *classDomain.Class) *ClassModel {
	return &ClassModel{
		ID:        c.ID().String(),
		Name:      c.Name().String(),
		Grade:     c.Grade().Value(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

func (m *ClassModel) ToClassDomain() (*classDomain.Class, error) {
	id, err := classDomain.ParseID(m.ID)
	if err != nil {
		return nil, err
	}
	name, err := classDomain.NewName(m.Name)
	if err != nil {
		return nil, err
	}
	grade, err := classDomain.NewGrade(m.Grade)
	if err != nil {
		return nil, err
	}
	return classDomain.RestoreClass(id, name, grade, m.CreatedAt, m.UpdatedAt), nil
}

type TeacherModel struct {
	ID     string `gorm:"primaryKey;type:varchar(36)"`
	UserID string `gorm:"not null;index"`
}

func (TeacherModel) TableName() string { return "teachers" }

type StudentModel struct {
	ID      string `gorm:"primaryKey;type:varchar(36)"`
	UserID  string `gorm:"not null;uniqueIndex"`
	ClassID string `gorm:"not null;index"`
}

func (StudentModel) TableName() string { return "students" }

type ClassSubjectModel struct {
	ClassID   string `gorm:"primaryKey;type:varchar(36)"`
	TeacherID string `gorm:"primaryKey;type:varchar(36)"`
}

func (ClassSubjectModel) TableName() string { return "class_subjects" }
