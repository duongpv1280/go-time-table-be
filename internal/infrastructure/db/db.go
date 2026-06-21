package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDatabase initializes a GORM SQLite connection and runs auto-migrations.
func NewDatabase() (*gorm.DB, error) {
	// Using in-memory sqlite for testing/development or a local file
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Running database migrations...")
	if err := db.AutoMigrate(&UserModel{}, &SubjectModel{}); err != nil {
		return nil, err
	}

	return db, nil
}
