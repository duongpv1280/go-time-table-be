package db

import (
	"time"

	"gosample/internal/domain/user"
)

type UserModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	Role      string    `gorm:"type:varchar(20);not null;default:''"`
	GoogleID  string    `gorm:"type:varchar(255)"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName defines the table name for UserModel.
func (UserModel) TableName() string {
	return "users"
}

func FromDomain(u *user.User) *UserModel {
	return &UserModel{
		ID:        u.ID().String(),
		Email:     u.Email().String(),
		Name:      u.Name().String(),
		Role:      u.Role().String(),
		GoogleID:  u.GoogleID(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

func (m *UserModel) ToDomain() (*user.User, error) {
	id, err := user.ParseID(m.ID)
	if err != nil {
		return nil, err
	}

	email, err := user.NewEmail(m.Email)
	if err != nil {
		return nil, err
	}

	name, err := user.NewName(m.Name)
	if err != nil {
		return nil, err
	}

	var role user.Role
	if m.Role != "" {
		role, err = user.NewRole(m.Role)
		if err != nil {
			return nil, err
		}
	}

	return user.RestoreUser(id, email, name, role, m.GoogleID, m.CreatedAt, m.UpdatedAt), nil
}
