package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	userDomain "gosample/internal/domain/user"
	authUseCase "gosample/internal/usecase/auth"
)

// --- Mocks ---

type mockGoogleVerifier struct {
	claims *domainAuth.GoogleClaims
	err    error
}

func (m *mockGoogleVerifier) Verify(_ context.Context, _ string) (*domainAuth.GoogleClaims, error) {
	return m.claims, m.err
}

type mockUserRepository struct {
	users     map[string]*userDomain.User
	createErr error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{users: make(map[string]*userDomain.User)}
}

func (m *mockUserRepository) Create(_ context.Context, u *userDomain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[u.Email().String()] = u
	return nil
}

func (m *mockUserRepository) FindByID(_ context.Context, id userDomain.ID) (*userDomain.User, error) {
	for _, u := range m.users {
		if u.ID().String() == id.String() {
			return u, nil
		}
	}
	return nil, userDomain.ErrUserNotFound
}

func (m *mockUserRepository) FindByEmail(_ context.Context, email userDomain.Email) (*userDomain.User, error) {
	u, ok := m.users[email.String()]
	if !ok {
		return nil, userDomain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepository) FindAll(_ context.Context) ([]*userDomain.User, error) {
	list := make([]*userDomain.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func (m *mockUserRepository) Delete(_ context.Context, id userDomain.ID) error {
	for email, u := range m.users {
		if u.ID().String() == id.String() {
			delete(m.users, email)
			return nil
		}
	}
	return userDomain.ErrUserNotFound
}

type mockRoleRepository struct {
	err error
}

func (m *mockRoleRepository) AddRoleForUser(_ context.Context, _, _ string) error {
	return m.err
}

type mockPermissionService struct{}

func (m *mockPermissionService) GetPermissionsForRole(_ context.Context, role string) (map[string]interface{}, error) {
	switch role {
	case userDomain.RoleStudent:
		return map[string]interface{}{"classes": map[string]interface{}{"subjects": "read", "teachers": "read"}}, nil
	case userDomain.RoleTeacher:
		return map[string]interface{}{"teachers": "read", "slots": "write"}, nil
	default:
		return map[string]interface{}{"users": "write"}, nil
	}
}

type mockJWTService struct {
	token string
	err   error
}

func (m *mockJWTService) Sign(_ context.Context, _, _ string) (string, error) {
	return m.token, m.err
}

func (m *mockJWTService) Verify(_ context.Context, _ string) (*domainAuth.JWTClaims, error) {
	return nil, nil
}

// --- Tests ---

func TestGoogleAuth_SignUp(t *testing.T) {
	verifier := &mockGoogleVerifier{
		claims: &domainAuth.GoogleClaims{Sub: "google-sub-1", Email: "new@example.com", Name: "New User"},
	}
	userRepo := newMockUserRepository()
	roleRepo := &mockRoleRepository{}
	permSvc := &mockPermissionService{}
	jwtSvc := &mockJWTService{token: "signed-token"}

	uc := authUseCase.NewGoogleAuthUseCase(verifier, userRepo, roleRepo, permSvc, jwtSvc)

	dto, err := uc.Execute(context.Background(), "valid-token")

	require.NoError(t, err)
	assert.Equal(t, "New User", dto.Name)
	assert.Equal(t, "new@example.com", dto.Email)
	assert.Equal(t, userDomain.RoleStudent, dto.Role)
	assert.Equal(t, "signed-token", dto.Token)
	assert.NotEmpty(t, dto.Permissions)

	// New user should be persisted
	_, found := userRepo.users["new@example.com"]
	assert.True(t, found)
}

func TestGoogleAuth_SignIn(t *testing.T) {
	email, _ := userDomain.NewEmail("existing@example.com")
	name, _ := userDomain.NewName("Existing User")
	role := userDomain.DefaultRole()
	existingUser := userDomain.NewGoogleUser(email, name, role, "sub-existing")

	verifier := &mockGoogleVerifier{
		claims: &domainAuth.GoogleClaims{Sub: "sub-existing", Email: "existing@example.com", Name: "Existing User"},
	}
	userRepo := newMockUserRepository()
	userRepo.users["existing@example.com"] = existingUser
	roleRepo := &mockRoleRepository{}
	permSvc := &mockPermissionService{}
	jwtSvc := &mockJWTService{token: "signed-token"}

	uc := authUseCase.NewGoogleAuthUseCase(verifier, userRepo, roleRepo, permSvc, jwtSvc)

	dto, err := uc.Execute(context.Background(), "valid-token")

	require.NoError(t, err)
	assert.Equal(t, "Existing User", dto.Name)
	assert.Equal(t, "existing@example.com", dto.Email)
	assert.Equal(t, userDomain.RoleStudent, dto.Role)
}

func TestGoogleAuth_InvalidToken(t *testing.T) {
	verifier := &mockGoogleVerifier{err: domainAuth.ErrInvalidToken}
	userRepo := newMockUserRepository()
	roleRepo := &mockRoleRepository{}
	permSvc := &mockPermissionService{}
	jwtSvc := &mockJWTService{}

	uc := authUseCase.NewGoogleAuthUseCase(verifier, userRepo, roleRepo, permSvc, jwtSvc)

	_, err := uc.Execute(context.Background(), "bad-token")

	require.Error(t, err)
	assert.True(t, errors.Is(err, domainAuth.ErrInvalidToken))
}

func TestGoogleAuth_DBError(t *testing.T) {
	verifier := &mockGoogleVerifier{
		claims: &domainAuth.GoogleClaims{Sub: "sub-1", Email: "user@example.com", Name: "User"},
	}
	dbErr := errors.New("database connection failed")
	userRepo := newMockUserRepository()
	userRepo.createErr = dbErr
	roleRepo := &mockRoleRepository{}
	permSvc := &mockPermissionService{}
	jwtSvc := &mockJWTService{}

	uc := authUseCase.NewGoogleAuthUseCase(verifier, userRepo, roleRepo, permSvc, jwtSvc)

	_, err := uc.Execute(context.Background(), "valid-token")

	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

func TestGoogleAuth_JWTSignError_ReturnsError(t *testing.T) {
	verifier := &mockGoogleVerifier{
		claims: &domainAuth.GoogleClaims{Sub: "sub-1", Email: "user@example.com", Name: "User"},
	}
	jwtErr := errors.New("jwt sign failed")
	userRepo := newMockUserRepository()
	roleRepo := &mockRoleRepository{}
	permSvc := &mockPermissionService{}
	jwtSvc := &mockJWTService{err: jwtErr}

	uc := authUseCase.NewGoogleAuthUseCase(verifier, userRepo, roleRepo, permSvc, jwtSvc)

	_, err := uc.Execute(context.Background(), "valid-token")

	require.Error(t, err)
	assert.ErrorIs(t, err, jwtErr)
}
