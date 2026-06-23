//go:build wireinject
// +build wireinject

package di

import (
	domainAuth "gosample/internal/domain/auth"
	"gosample/internal/delivery/http/handlers"
	"gosample/internal/delivery/http/validator/rules"
	"gosample/internal/infrastructure/config"
	"gosample/internal/infrastructure/db"
	infraAuth "gosample/internal/infrastructure/auth"
	usecaseAuth "gosample/internal/usecase/auth"
	classUseCase "gosample/internal/usecase/class"
	usecase "gosample/internal/usecase/user"

	"github.com/google/wire"
	"gorm.io/gorm"
)

type Application struct {
	DB         *gorm.DB
	Handler    *handlers.CombinedHandler
	JWTService domainAuth.IJWTService
}

func NewApplication(db *gorm.DB, handler *handlers.CombinedHandler, jwtService domainAuth.IJWTService) *Application {
	return &Application{
		DB:         db,
		Handler:    handler,
		JWTService: jwtService,
	}
}

// UserSet bundles all providers for the User component.
var UserSet = wire.NewSet(
	db.NewGormUserRepository,
	usecase.NewUserUseCase,
	handlers.NewUserHandler,
)

// AuthSet bundles all providers for the Google Auth component.
var AuthSet = wire.NewSet(
	infraAuth.NewGoogleVerifier,
	infraAuth.NewPermissionService,
	db.NewGormCasbinRepository,
	usecaseAuth.NewGoogleAuthUseCase,
	handlers.NewAuthHandler,
)

// JWTSet bundles the JWT service provider.
var JWTSet = wire.NewSet(
	infraAuth.NewJWTService,
)

// ValidatorSet bundles the validator provider.
var ValidatorSet = wire.NewSet(
	rules.NewValidator,
)

// ClassSet bundles all providers for the Class component.
var ClassSet = wire.NewSet(
	db.NewGormClassRepository,
	classUseCase.NewClassUseCase,
	ValidatorSet,
	handlers.NewClassHandler,
)

// InitializeApp resolves all dependencies via Wire.
// config.Load → *config.Config → db.NewDatabase → *gorm.DB → repositories → usecases → handlers.
func InitializeApp() (*Application, error) {
	wire.Build(
		config.Load,
		db.NewDatabase,
		UserSet,
		AuthSet,
		JWTSet,
		ClassSet,
		handlers.NewCombinedHandler,
		NewApplication,
	)
	return nil, nil
}
