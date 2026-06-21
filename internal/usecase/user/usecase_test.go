package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gosample/internal/domain/user"
	usecase "gosample/internal/usecase/user"
)

type mockUserRepository struct {
	users     map[user.ID]*user.User
	createErr error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[user.ID]*user.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, u *user.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[u.ID()] = u
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	list := make([]*user.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func (m *mockUserRepository) FindByEmail(_ context.Context, email user.Email) (*user.User, error) {
	for _, u := range m.users {
		if u.Email().String() == email.String() {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepository) Delete(ctx context.Context, id user.ID) error {
	if _, ok := m.users[id]; !ok {
		return user.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func TestCreateUser(t *testing.T) {
	repo := newMockUserRepository()
	uc := usecase.NewUserUseCase(repo)

	dto, err := uc.CreateUser(context.Background(), usecase.CreateUserParams{
		Email: "john@example.com",
		Name:  "John",
	})

	require.NoError(t, err)
	assert.Equal(t, "john@example.com", dto.Email)
	assert.Equal(t, "John", dto.Name)
	assert.NotEmpty(t, dto.ID)
}

func TestCreateUser_InvalidEmail_ReturnsBadRequest(t *testing.T) {
	repo := newMockUserRepository()
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.CreateUser(context.Background(), usecase.CreateUserParams{
		Email: "not-an-email",
		Name:  "John",
	})

	require.ErrorIs(t, err, user.ErrInvalidEmail)
}

func TestGetUser_NotFound(t *testing.T) {
	repo := newMockUserRepository()
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.GetUser(context.Background(), "123e4567-e89b-12d3-a456-426614174000")

	require.ErrorIs(t, err, user.ErrUserNotFound)
}

func TestDeleteUser_RemovesUser(t *testing.T) {
	repo := newMockUserRepository()
	uc := usecase.NewUserUseCase(repo)

	dto, err := uc.CreateUser(context.Background(), usecase.CreateUserParams{
		Email: "delete@example.com",
		Name:  "ToDelete",
	})
	require.NoError(t, err)

	err = uc.DeleteUser(context.Background(), dto.ID)
	require.NoError(t, err)

	_, err = uc.GetUser(context.Background(), dto.ID)
	require.ErrorIs(t, err, user.ErrUserNotFound)
}
