package auth

import (
	"context"
	"errors"

	domainAuth "gosample/internal/domain/auth"
	userDomain "gosample/internal/domain/user"
)

type AuthResponseDTO struct {
	Name        string
	Email       string
	Role        string
	Token       string
	Permissions map[string]interface{}
}

type IGoogleAuthUseCase interface {
	Execute(ctx context.Context, idToken string) (AuthResponseDTO, error)
}

type googleAuthUseCase struct {
	verifier    domainAuth.IGoogleVerifier
	userRepo    userDomain.IUserRepository
	roleRepo    userDomain.IRoleRepository
	permissions domainAuth.IPermissionService
	jwtService  domainAuth.IJWTService
}

func NewGoogleAuthUseCase(
	verifier domainAuth.IGoogleVerifier,
	userRepo userDomain.IUserRepository,
	roleRepo userDomain.IRoleRepository,
	permissions domainAuth.IPermissionService,
	jwtService domainAuth.IJWTService,
) IGoogleAuthUseCase {
	return &googleAuthUseCase{
		verifier:    verifier,
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		permissions: permissions,
		jwtService:  jwtService,
	}
}

func (uc *googleAuthUseCase) Execute(ctx context.Context, idToken string) (AuthResponseDTO, error) {
	claims, err := uc.verifier.Verify(ctx, idToken)
	if err != nil {
		return AuthResponseDTO{}, domainAuth.ErrInvalidToken
	}

	email, err := userDomain.NewEmail(claims.Email)
	if err != nil {
		return AuthResponseDTO{}, err
	}

	u, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, userDomain.ErrUserNotFound) {
		return AuthResponseDTO{}, err
	}

	if errors.Is(err, userDomain.ErrUserNotFound) {
		name, nameErr := userDomain.NewName(claims.Name)
		if nameErr != nil {
			name, _ = userDomain.NewName(claims.Email)
		}
		role := userDomain.DefaultRole()
		newUser := userDomain.NewGoogleUser(email, name, role, claims.Sub)

		if createErr := uc.userRepo.Create(ctx, newUser); createErr != nil {
			return AuthResponseDTO{}, createErr
		}

		if roleErr := uc.roleRepo.AddRoleForUser(ctx, newUser.ID().String(), role.String()); roleErr != nil {
			return AuthResponseDTO{}, roleErr
		}

		u = newUser
	}

	role := u.Role()
	if role.IsZero() {
		role = userDomain.DefaultRole()
	}

	perms, err := uc.permissions.GetPermissionsForRole(ctx, role.String())
	if err != nil {
		return AuthResponseDTO{}, err
	}

	token, err := uc.jwtService.Sign(ctx, u.ID().String(), role.String())
	if err != nil {
		return AuthResponseDTO{}, err
	}

	return AuthResponseDTO{
		Name:        u.Name().String(),
		Email:       u.Email().String(),
		Role:        role.String(),
		Token:       token,
		Permissions: perms,
	}, nil
}
