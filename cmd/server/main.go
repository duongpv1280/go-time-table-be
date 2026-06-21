package main

import (
	"log"

	"gosample/internal/delivery/http"
	"gosample/internal/infrastructure/di"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize DI Application container
	app, err := di.InitializeApp()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	// Create Echo instance
	e := echo.New()

	// Register generic middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Register generated openapi handlers
	http.RegisterHandlers(e, app.UserHandler)

	// Start Echo HTTP server
	log.Println("Starting server on :8080...")
	if err := e.Start(":8080"); err != nil {
		log.Fatalf("echo server failed to start: %v", err)
	}
}
