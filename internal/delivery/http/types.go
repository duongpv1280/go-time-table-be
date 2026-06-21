package http

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type User struct {
	Id        openapi_types.UUID  `json:"id"`
	Email     openapi_types.Email `json:"email"`
	Name      string              `json:"name"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"perPage"`
	Total   int `json:"total"`
}

type ListUsersResponse struct {
	Data       []User     `json:"data"`
	Pagination Pagination `json:"pagination"`
}
