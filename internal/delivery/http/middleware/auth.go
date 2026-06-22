package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	api "gosample/internal/delivery/http"
	domainAuth "gosample/internal/domain/auth"
)

const PermissionContextKey = "permission"

func JWTAuth(jwtService domainAuth.IJWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, api.ErrorResponse{
					Error:   "unauthorized",
					Message: "Missing or invalid authorization header",
				})
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtService.Verify(c.Request().Context(), token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, api.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid or expired token",
				})
			}
			if claims.UserID == "" || claims.Role == "" {
				return c.JSON(http.StatusUnauthorized, api.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid or expired token",
				})
			}
			c.Set(PermissionContextKey, domainAuth.ContextPermission{
				UserID: claims.UserID,
				Role:   claims.Role,
			})
			return next(c)
		}
	}
}

func GetPermission(c echo.Context) (domainAuth.ContextPermission, bool) {
	v := c.Get(PermissionContextKey)
	if v == nil {
		return domainAuth.ContextPermission{}, false
	}
	perm, ok := v.(domainAuth.ContextPermission)
	return perm, ok
}
