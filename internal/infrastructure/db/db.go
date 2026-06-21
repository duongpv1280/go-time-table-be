package db

import (
	"fmt"
	"log"
	"time"

	"gosample/internal/infrastructure/config"

	"gorm.io/gorm"
)

const (
	mysqlMaxOpenConns    = 25
	mysqlMaxIdleConns    = 10
	mysqlConnMaxLifetime = 5 * time.Minute

	// SQLite is single-writer; limit to one connection to avoid "database is locked".
	sqliteMaxOpenConns = 1
)

// NewDatabase opens a GORM connection for the configured driver and runs
// AutoMigrate so the schema is always in sync during development.
func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	dialector, err := newDialector(cfg)
	if err != nil {
		return nil, err
	}

	db, err := openWithRetry(dialector, cfg.DBDriver)
	if err != nil {
		return nil, err
	}

	if err := configurePool(db, cfg.DBDriver); err != nil {
		return nil, err
	}

	log.Println("Running database auto-migrations...")
	if err := db.AutoMigrate(&UserModel{}, &SubjectModel{}, &CasbinRuleModel{}, &ClassModel{}, &TeacherModel{}, &StudentModel{}, &ClassSubjectModel{}); err != nil {
		return nil, err
	}

	return db, nil
}

// openWithRetry retries the connection a few times to handle race conditions
// when a database container (e.g. MySQL in Docker) is still starting up.
func openWithRetry(dialector gorm.Dialector, driver string) (*gorm.DB, error) {
	const maxAttempts = 5
	const retryDelay = 2 * time.Second

	var (
		db  *gorm.DB
		err error
	)
	for i := 0; i < maxAttempts; i++ {
		db, err = gorm.Open(dialector, &gorm.Config{})
		if err == nil {
			return db, nil
		}
		if i < maxAttempts-1 {
			log.Printf("[db] connection attempt %d/%d failed (%s): %v — retrying in %s",
				i+1, maxAttempts, driver, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return nil, fmt.Errorf("could not connect to %s after %d attempts: %w", driver, maxAttempts, err)
}

// configurePool sets connection-pool limits appropriate for each driver.
func configurePool(db *gorm.DB, driver string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	switch driver {
	case "mysql":
		sqlDB.SetMaxOpenConns(mysqlMaxOpenConns)
		sqlDB.SetMaxIdleConns(mysqlMaxIdleConns)
		sqlDB.SetConnMaxLifetime(mysqlConnMaxLifetime)
	default: // sqlite
		sqlDB.SetMaxOpenConns(sqliteMaxOpenConns)
	}
	return nil
}
