package main

import (
	"fmt"
	"log"

	httpMiddleware "gosample/internal/delivery/http/middleware"
	openapi "gosample/internal/delivery/http/openapi"
	"gosample/internal/infrastructure/config"
	"gosample/internal/infrastructure/di"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := di.InitializeApp()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	jwtMiddleware := httpMiddleware.JWTAuth(app.JWTService)
	openapi.RegisterHandlersWithOptions(e, app.Handler, openapi.RegisterHandlersOptions{
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"getClasses":   {jwtMiddleware},
			"getClassById": {jwtMiddleware},
			"createClass":  {jwtMiddleware},
			"updateClass":  {jwtMiddleware},
		},
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s...", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("echo server failed to start: %v", err)
	}
}
