package subject

import "time"

type Subject struct {
	id        ID
	name      Name
	createdAt time.Time
	updatedAt time.Time
}

func NewSubject(name Name) *Subject {
	now := time.Now().UTC()
	return &Subject{
		id:        NewID(),
		name:      name,
		createdAt: now,
		updatedAt: now,
	}
}

func RestoreSubject(id ID, name Name, createdAt, updatedAt time.Time) *Subject {
	return &Subject{
		id:        id,
		name:      name,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (s *Subject) ID() ID               { return s.id }
func (s *Subject) Name() Name           { return s.name }
func (s *Subject) CreatedAt() time.Time { return s.createdAt }
func (s *Subject) UpdatedAt() time.Time { return s.updatedAt }

func (s *Subject) UpdateName(newName Name) {
	s.name = newName
	s.updatedAt = time.Now().UTC()
}
