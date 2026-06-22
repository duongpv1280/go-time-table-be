package handlers

import (
	"errors"
	"net/http"

	api "gosample/internal/delivery/http"
	domainAuth "gosample/internal/domain/auth"
	authUseCase "gosample/internal/usecase/auth"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	useCase authUseCase.IGoogleAuthUseCase
}

func NewAuthHandler(useCase authUseCase.IGoogleAuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}

// GoogleAuth implements ServerInterface.
// (POST /auth/google)
func (h *AuthHandler) GoogleAuth(ctx echo.Context) error {
	var req api.GoogleAuthJSONRequestBody
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Invalid request body"})
	}

	result, err := h.useCase.Execute(ctx.Request().Context(), req.IdToken)
	if err != nil {
		if errors.Is(err, domainAuth.ErrInvalidToken) {
			return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{
				Error:   "invalid_token",
				Message: "Google ID token is invalid or expired",
			})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}

	return ctx.JSON(http.StatusOK, api.AuthResponse{
		Name:        result.Name,
		Email:       result.Email,
		Role:        result.Role,
		Token:       result.Token,
		Permissions: result.Permissions,
	})
}
