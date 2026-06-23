package class

import "time"

type Class struct {
	id        ID
	name      Name
	grade     Grade
	createdAt time.Time
	updatedAt time.Time
}

func NewClass(name Name, grade Grade) *Class {
	now := time.Now().UTC()
	return &Class{
		id:        NewID(),
		name:      name,
		grade:     grade,
		createdAt: now,
		updatedAt: now,
	}
}

func RestoreClass(id ID, name Name, grade Grade, createdAt, updatedAt time.Time) *Class {
	return &Class{
		id:        id,
		name:      name,
		grade:     grade,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (c *Class) ID() ID {
	return c.id
}

func (c *Class) Name() Name {
	return c.name
}

func (c *Class) Grade() Grade {
	return c.grade
}

func (c *Class) CreatedAt() time.Time {
	return c.createdAt
}

func (c *Class) UpdatedAt() time.Time {
	return c.updatedAt
}

func (c *Class) UpdateName(name Name) {
	c.name = name
	c.updatedAt = time.Now().UTC()
}

func (c *Class) UpdateGrade(grade Grade) {
	c.grade = grade
	c.updatedAt = time.Now().UTC()
}
