package db

import (
	"time"

	"gosample/internal/domain/subject"
)

type SubjectModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (SubjectModel) TableName() string {
	return "subjects"
}

func SubjectFromDomain(s *subject.Subject) *SubjectModel {
	return &SubjectModel{
		ID:        s.ID().String(),
		Name:      s.Name().String(),
		CreatedAt: s.CreatedAt(),
		UpdatedAt: s.UpdatedAt(),
	}
}

func (m *SubjectModel) ToSubjectDomain() (*subject.Subject, error) {
	id, err := subject.ParseID(m.ID)
	if err != nil {
		return nil, err
	}
	name, err := subject.NewName(m.Name)
	if err != nil {
		return nil, err
	}
	return subject.RestoreSubject(id, name, m.CreatedAt, m.UpdatedAt), nil
}
