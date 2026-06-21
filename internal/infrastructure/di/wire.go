//go:build wireinject
// +build wireinject

package di

import (
	"gosample/internal/delivery/http/handlers"
	"gosample/internal/infrastructure/db"
	usecase "gosample/internal/usecase/user"

	"github.com/google/wire"
	"gorm.io/gorm"
)

type Application struct {
	DB          *gorm.DB
	UserHandler *handlers.UserHandler
}

func NewApplication(db *gorm.DB, handler *handlers.UserHandler) *Application {
	return &Application{
		DB:          db,
		UserHandler: handler,
	}
}

// UserSet bundles all providers for the User component.
var UserSet = wire.NewSet(
	db.NewGormUserRepository,
	usecase.NewUserUseCase,
	handlers.NewUserHandler,
)

// InitializeApp resolves database connection, repository, usecase, and handler.
func InitializeApp() (*Application, error) {
	wire.Build(
		db.NewDatabase,
		UserSet,
		NewApplication,
	)
	return nil, nil
}
