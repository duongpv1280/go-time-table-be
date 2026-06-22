package db

import (
	"fmt"

	"gosample/internal/infrastructure/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newDialector returns the GORM dialector for the configured DB driver.
// Add new drivers here; no other file needs to change.
func newDialector(cfg *config.Config) (gorm.Dialector, error) {
	switch cfg.DBDriver {
	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
		)
		return mysql.Open(dsn), nil

	case "sqlite", "":
		return sqlite.Open(cfg.DBPath), nil

	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q — valid values: sqlite, mysql", cfg.DBDriver)
	}
}
