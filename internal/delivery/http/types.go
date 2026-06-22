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
	Error   string `json:"error,omitempty"`
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

type AuthResponse struct {
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	Role        string                 `json:"role"`
	Token       string                 `json:"token"`
	Permissions map[string]interface{} `json:"permissions"`
}

type Class struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Grade     int       `json:"grade"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ListClassesResponse struct {
	Data []Class `json:"data"`
}
