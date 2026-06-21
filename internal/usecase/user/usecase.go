package user

import (
	"context"

	"gosample/internal/domain/user"
)

type IUserUseCase interface {
	CreateUser(ctx context.Context, params CreateUserParams) (UserDTO, error)
	GetUser(ctx context.Context, idStr string) (UserDTO, error)
	ListUsers(ctx context.Context) ([]UserDTO, error)
	DeleteUser(ctx context.Context, idStr string) error
}

type userUseCase struct {
	repo user.IUserRepository
}

func NewUserUseCase(repo user.IUserRepository) IUserUseCase {
	return &userUseCase{
		repo: repo,
	}
}

func (uc *userUseCase) CreateUser(ctx context.Context, params CreateUserParams) (UserDTO, error) {
	email, err := user.NewEmail(params.Email)
	if err != nil {
		return UserDTO{}, err
	}

	name, err := user.NewName(params.Name)
	if err != nil {
		return UserDTO{}, err
	}

	// We can check duplicate emails if required by domain rules, but let's delegate to repository constraints or a domain service.
	// In database implementations, we'll return ErrUserAlreadyExists on unique index violation.
	newUser := user.NewUser(email, name)

	if err := uc.repo.Create(ctx, newUser); err != nil {
		return UserDTO{}, err
	}

	return ToUserDTO(newUser), nil
}

func (uc *userUseCase) GetUser(ctx context.Context, idStr string) (UserDTO, error) {
	id, err := user.ParseID(idStr)
	if err != nil {
		return UserDTO{}, err
	}

	u, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return UserDTO{}, err
	}

	return ToUserDTO(u), nil
}

func (uc *userUseCase) ListUsers(ctx context.Context) ([]UserDTO, error) {
	users, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	return ToUserDTOList(users), nil
}

func (uc *userUseCase) DeleteUser(ctx context.Context, idStr string) error {
	id, err := user.ParseID(idStr)
	if err != nil {
		return err
	}

	return uc.repo.Delete(ctx, id)
}
