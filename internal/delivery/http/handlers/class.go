package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	api "gosample/internal/delivery/http"
	httpMiddleware "gosample/internal/delivery/http/middleware"
	httpvalidator "gosample/internal/delivery/http/validator"
	"gosample/internal/delivery/http/validator/rules"
	domainAuth "gosample/internal/domain/auth"
	classDomain "gosample/internal/domain/class"
	classUseCase "gosample/internal/usecase/class"
)

type ClassHandler struct {
	useCase   classUseCase.IClassUseCase
	validator httpvalidator.IValidator
}

func NewClassHandler(uc classUseCase.IClassUseCase, v httpvalidator.IValidator) *ClassHandler {
	return &ClassHandler{useCase: uc, validator: v}
}

func (h *ClassHandler) GetClasses(ctx echo.Context) error {
	perm, ok := httpMiddleware.GetPermission(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing permission context",
		})
	}

	classes, err := h.useCase.GetClasses(ctx.Request().Context(), perm)
	if err != nil {
		if errors.Is(err, domainAuth.ErrUnauthorized) {
			return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
				Error:   "unauthorized",
				Message: "Access denied",
			})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}

	return ctx.JSON(http.StatusOK, api.ListClassesResponse{Data: toAPIClasses(classes)})
}

func (h *ClassHandler) GetClassById(ctx echo.Context, classId string) error {
	perm, ok := httpMiddleware.GetPermission(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing permission context",
		})
	}

	c, err := h.useCase.GetClassByID(ctx.Request().Context(), classId, perm)
	if err != nil {
		if errors.Is(err, domainAuth.ErrUnauthorized) || errors.Is(err, classDomain.ErrClassNotFound) {
			return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
				Error:   "unauthorized",
				Message: "Access denied",
			})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}

	return ctx.JSON(http.StatusOK, toAPIClass(c))
}

func (h *ClassHandler) CreateClass(ctx echo.Context) error {
	perm, ok := httpMiddleware.GetPermission(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing permission context",
		})
	}

	var req api.CreateClassRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if err := h.validator.ValidateCtx(ctx.Request().Context(), &req); err != nil {
		return validationError(ctx, err)
	}

	dto, err := h.useCase.CreateClass(ctx.Request().Context(), req.Name, req.Grade, perm)
	if err != nil {
		if errors.Is(err, domainAuth.ErrUnauthorized) {
			return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
				Error:   "unauthorized",
				Message: "Access denied",
			})
		}
		if errors.Is(err, classDomain.ErrEmptyClassName) || errors.Is(err, classDomain.ErrInvalidGrade) {
			return ctx.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}

	return ctx.JSON(http.StatusCreated, toAPIClass(dto))
}

func (h *ClassHandler) UpdateClass(ctx echo.Context, classId string) error {
	perm, ok := httpMiddleware.GetPermission(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing permission context",
		})
	}

	var req api.UpdateClassRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	reqCtx := context.WithValue(ctx.Request().Context(), rules.ExcludeIDKey, classId)

	if err := h.validator.ValidateCtx(reqCtx, &req); err != nil {
		return validationError(ctx, err)
	}

	dto, err := h.useCase.UpdateClass(reqCtx, classId, req.Name, req.Grade, perm)
	if err != nil {
		if errors.Is(err, domainAuth.ErrUnauthorized) {
			return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
				Error:   "unauthorized",
				Message: "Access denied",
			})
		}
		if errors.Is(err, classDomain.ErrEmptyClassName) || errors.Is(err, classDomain.ErrInvalidGrade) || errors.Is(err, classDomain.ErrInvalidClassID) {
			return ctx.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}

	return ctx.JSON(http.StatusOK, toAPIClass(dto))
}

func validationError(ctx echo.Context, err error) error {
	fields := map[string]string{}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			fields[strings.ToLower(fe.Field())] = fe.Tag()
		}
	}
	return ctx.JSON(http.StatusUnprocessableEntity, api.ValidationErrorResponse{
		Error:   "validation_error",
		Message: "Validation failed",
		Fields:  fields,
	})
}

func toAPIClass(dto *classUseCase.ClassDTO) api.Class {
	return api.Class{
		Id:        dto.ID,
		Name:      dto.Name,
		Grade:     dto.Grade,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}
}

func toAPIClasses(dtos []classUseCase.ClassDTO) []api.Class {
	classes := make([]api.Class, len(dtos))
	for i, dto := range dtos {
		d := dto
		classes[i] = toAPIClass(&d)
	}
	return classes
}
