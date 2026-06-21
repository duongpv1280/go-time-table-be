package user

import (
	"time"

	"gosample/internal/domain/user"
)

type CreateUserParams struct {
	Email string
	Name  string
}

type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ToUserDTO(u *user.User) UserDTO {
	return UserDTO{
		ID:        u.ID().String(),
		Email:     u.Email().String(),
		Name:      u.Name().String(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

func ToUserDTOList(users []*user.User) []UserDTO {
	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = ToUserDTO(u)
	}
	return dtos
}
