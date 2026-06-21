package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	DBPath     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "gorm.db"
	}

	return &Config{
		ServerPort: port,
		DBPath:     dbPath,
	}, nil
}
