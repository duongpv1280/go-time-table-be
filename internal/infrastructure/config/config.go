package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string

	// DBDriver selects the database backend: "sqlite" (default) or "mysql".
	DBDriver string

	// SQLite — used when DBDriver is "sqlite".
	DBPath string

	// MySQL — used when DBDriver is "mysql".
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// JWTSecret is used to sign and verify JWT tokens.
	JWTSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("[WARNING] JWT_SECRET is not set — using insecure default for development only")
		jwtSecret = "dev-secret-change-in-production"
	}

	return &Config{
		ServerPort: envOr("SERVER_PORT", "8080"),
		DBDriver:   envOr("DB_DRIVER", "sqlite"),
		DBPath:     envOr("DB_PATH", "gorm.db"),
		DBHost:     envOr("DB_HOST", "localhost"),
		DBPort:     envOr("DB_PORT", "3306"),
		DBName:     envOr("DB_NAME", "timetable"),
		DBUser:     envOr("DB_USER", "root"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		JWTSecret:  jwtSecret,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
