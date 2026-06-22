package main

import (
	"fmt"
	"log"

	"gosample/internal/delivery/http"
	httpMiddleware "gosample/internal/delivery/http/middleware"
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

	http.RegisterHandlers(e, app.Handler)

	apiV1 := e.Group("/api/v1")
	apiV1.Use(httpMiddleware.JWTAuth(app.JWTService))
	apiV1.GET("/classes", app.ClassHandler.GetClasses)
	apiV1.GET("/classes/:classId", func(c echo.Context) error {
		return app.ClassHandler.GetClassById(c, c.Param("classId"))
	})
	apiV1.POST("/classes", app.ClassHandler.CreateClass)
	apiV1.PUT("/classes/:classId", func(c echo.Context) error {
		return app.ClassHandler.UpdateClass(c, c.Param("classId"))
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s...", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("echo server failed to start: %v", err)
	}
}
