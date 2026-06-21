package main

import (
	"fmt"
	"log"

	"gosample/internal/delivery/http"
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

	http.RegisterHandlers(e, app.UserHandler)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s...", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("echo server failed to start: %v", err)
	}
}
