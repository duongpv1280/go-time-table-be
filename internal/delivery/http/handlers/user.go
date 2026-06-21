package handlers

import (
	"errors"
	"net/http"

	api "gosample/internal/delivery/http"
	"gosample/internal/domain/user"
	userUseCase "gosample/internal/usecase/user"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type UserHandler struct {
	useCase userUseCase.IUserUseCase
}

func NewUserHandler(useCase userUseCase.IUserUseCase) *UserHandler {
	return &UserHandler{
		useCase: useCase,
	}
}

// CreateUser implements ServerInterface
// (POST /users)
func (h *UserHandler) CreateUser(ctx echo.Context) error {
	var req api.CreateUserJSONRequestBody
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Invalid request body"})
	}

	res, err := h.useCase.CreateUser(ctx.Request().Context(), userUseCase.CreateUserParams{
		Email: string(req.Email),
		Name:  req.Name,
	})
	if err != nil {
		if errors.Is(err, user.ErrInvalidEmail) || errors.Is(err, user.ErrEmptyName) || errors.Is(err, user.ErrUserAlreadyExists) {
			return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Internal server error"})
	}

	return ctx.JSON(http.StatusCreated, api.User{
		Id:        uuid.MustParse(res.ID),
		Email:     openapi_types.Email(res.Email),
		Name:      res.Name,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
	})
}

// ListUsers implements ServerInterface
// (GET /users)
func (h *UserHandler) ListUsers(ctx echo.Context, params api.ListUsersParams) error {
	users, err := h.useCase.ListUsers(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Internal server error"})
	}

	data := make([]api.User, len(users))
	for i, u := range users {
		data[i] = api.User{
			Id:        uuid.MustParse(u.ID),
			Email:     openapi_types.Email(u.Email),
			Name:      u.Name,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}

	return ctx.JSON(http.StatusOK, api.ListUsersResponse{
		Data:       data,
		Pagination: api.Pagination{},
	})
}

// GetUser implements ServerInterface
// (GET /users/{id})
func (h *UserHandler) GetUser(ctx echo.Context, id openapi_types.UUID) error {
	res, err := h.useCase.GetUser(ctx.Request().Context(), id.String())
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) || errors.Is(err, user.ErrInvalidID) {
			return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Message: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Internal server error"})
	}

	return ctx.JSON(http.StatusOK, api.User{
		Id:        uuid.MustParse(res.ID),
		Email:     openapi_types.Email(res.Email),
		Name:      res.Name,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
	})
}

// DeleteUser implements ServerInterface
// (DELETE /users/{id})
func (h *UserHandler) DeleteUser(ctx echo.Context, id openapi_types.UUID) error {
	err := h.useCase.DeleteUser(ctx.Request().Context(), id.String())
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) || errors.Is(err, user.ErrInvalidID) {
			return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Message: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Internal server error"})
	}

	return ctx.NoContent(http.StatusNoContent)
}
