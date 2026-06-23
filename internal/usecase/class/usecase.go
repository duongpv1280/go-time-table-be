package class

import (
	"context"
	"errors"

	domainAuth "gosample/internal/domain/auth"
	classDomain "gosample/internal/domain/class"
	userDomain "gosample/internal/domain/user"
)

type IClassUseCase interface {
	GetClasses(ctx context.Context, permission domainAuth.ContextPermission) ([]ClassDTO, error)
	GetClassByID(ctx context.Context, classID string, permission domainAuth.ContextPermission) (*ClassDTO, error)
	CreateClass(ctx context.Context, name string, grade int, permission domainAuth.ContextPermission) (*ClassDTO, error)
	UpdateClass(ctx context.Context, classID, name string, grade *int, permission domainAuth.ContextPermission) (*ClassDTO, error)
}

type classUseCase struct {
	repo classDomain.IClassRepository
}

func NewClassUseCase(repo classDomain.IClassRepository) IClassUseCase {
	return &classUseCase{repo: repo}
}

func (uc *classUseCase) GetClasses(ctx context.Context, permission domainAuth.ContextPermission) ([]ClassDTO, error) {
	switch permission.Role {
	case userDomain.RoleAdmin:
		classes, err := uc.repo.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		return toClassDTOs(classes), nil

	case userDomain.RoleTeacher:
		classes, err := uc.repo.FindByTeacherUserID(ctx, permission.UserID)
		if err != nil {
			return nil, err
		}
		return toClassDTOs(classes), nil

	case userDomain.RoleStudent:
		c, err := uc.repo.FindByStudentUserID(ctx, permission.UserID)
		if err != nil {
			if errors.Is(err, classDomain.ErrClassNotFound) {
				return nil, domainAuth.ErrUnauthorized
			}
			return nil, err
		}
		return []ClassDTO{toClassDTO(c)}, nil

	default:
		return nil, domainAuth.ErrUnauthorized
	}
}

func (uc *classUseCase) GetClassByID(ctx context.Context, classID string, permission domainAuth.ContextPermission) (*ClassDTO, error) {
	id, err := classDomain.ParseID(classID)
	if err != nil {
		return nil, domainAuth.ErrUnauthorized
	}

	switch permission.Role {
	case userDomain.RoleAdmin:
		c, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			if errors.Is(err, classDomain.ErrClassNotFound) {
				return nil, domainAuth.ErrUnauthorized
			}
			return nil, err
		}
		dto := toClassDTO(c)
		return &dto, nil

	case userDomain.RoleTeacher:
		classes, err := uc.repo.FindByTeacherUserID(ctx, permission.UserID)
		if err != nil {
			return nil, err
		}
		for _, c := range classes {
			if c.ID().String() == id.String() {
				dto := toClassDTO(c)
				return &dto, nil
			}
		}
		return nil, domainAuth.ErrUnauthorized

	case userDomain.RoleStudent:
		c, err := uc.repo.FindByStudentUserID(ctx, permission.UserID)
		if err != nil {
			return nil, domainAuth.ErrUnauthorized
		}
		if c.ID().String() != id.String() {
			return nil, domainAuth.ErrUnauthorized
		}
		dto := toClassDTO(c)
		return &dto, nil

	default:
		return nil, domainAuth.ErrUnauthorized
	}
}

func (uc *classUseCase) CreateClass(ctx context.Context, name string, grade int, permission domainAuth.ContextPermission) (*ClassDTO, error) {
	if permission.Role != userDomain.RoleAdmin {
		return nil, domainAuth.ErrUnauthorized
	}
	n, err := classDomain.NewName(name)
	if err != nil {
		return nil, err
	}
	g, err := classDomain.NewGrade(grade)
	if err != nil {
		return nil, err
	}
	c := classDomain.NewClass(n, g)
	if err := uc.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	dto := toClassDTO(c)
	return &dto, nil
}

func (uc *classUseCase) UpdateClass(ctx context.Context, classID, name string, grade *int, permission domainAuth.ContextPermission) (*ClassDTO, error) {
	if permission.Role != userDomain.RoleAdmin {
		return nil, domainAuth.ErrUnauthorized
	}
	id, err := classDomain.ParseID(classID)
	if err != nil {
		return nil, domainAuth.ErrUnauthorized
	}
	c, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, classDomain.ErrClassNotFound) {
			return nil, domainAuth.ErrUnauthorized
		}
		return nil, err
	}
	if name != "" {
		n, err := classDomain.NewName(name)
		if err != nil {
			return nil, err
		}
		c.UpdateName(n)
	}
	if grade != nil {
		g, err := classDomain.NewGrade(*grade)
		if err != nil {
			return nil, err
		}
		c.UpdateGrade(g)
	}
	if err := uc.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	dto := toClassDTO(c)
	return &dto, nil
}

func toClassDTO(c *classDomain.Class) ClassDTO {
	return ClassDTO{
		ID:        c.ID().String(),
		Name:      c.Name().String(),
		Grade:     c.Grade().Value(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

func toClassDTOs(classes []*classDomain.Class) []ClassDTO {
	dtos := make([]ClassDTO, len(classes))
	for i, c := range classes {
		dtos[i] = toClassDTO(c)
	}
	return dtos
}
