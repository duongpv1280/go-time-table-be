package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	api "gosample/internal/delivery/http"
	httpMiddleware "gosample/internal/delivery/http/middleware"
	domainAuth "gosample/internal/domain/auth"
	classDomain "gosample/internal/domain/class"
	classUseCase "gosample/internal/usecase/class"
)

type ClassHandler struct {
	useCase classUseCase.IClassUseCase
}

func NewClassHandler(uc classUseCase.IClassUseCase) *ClassHandler {
	return &ClassHandler{useCase: uc}
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
