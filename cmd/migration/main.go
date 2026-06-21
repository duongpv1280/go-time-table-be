package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"gosample/internal/infrastructure/config"
)

func main() {
	cmd := flag.String("cmd", "up", "Command: up, down, version")
	steps := flag.Int("steps", 0, "Number of migration steps (0 = all)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbURL, err := buildDatabaseURL(cfg)
	if err != nil {
		log.Fatalf("failed to build database URL: %v", err)
	}

	m, err := migrate.New("file://cmd/migration/sqls", dbURL)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("source close error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("db close error: %v", dbErr)
		}
	}()

	var migrateErr error
	switch *cmd {
	case "up":
		if *steps > 0 {
			migrateErr = m.Steps(*steps)
		} else {
			migrateErr = m.Up()
		}
	case "down":
		if *steps > 0 {
			migrateErr = m.Steps(-(*steps))
		} else {
			migrateErr = m.Down()
		}
	case "version":
		version, dirty, verErr := m.Version()
		if verErr != nil {
			log.Fatalf("failed to get version: %v", verErr)
		}
		log.Printf("version: %d, dirty: %v", version, dirty)
		return
	default:
		log.Fatalf("unknown command %q — use: up, down, version", *cmd)
	}

	if migrateErr != nil && migrateErr != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", migrateErr)
	}
	log.Printf("[%s] migration completed", cfg.DBDriver)
}

// buildDatabaseURL constructs the golang-migrate URL for the configured driver.
// Adding a new driver here is the only change required in this file.
func buildDatabaseURL(cfg *config.Config) (string, error) {
	switch cfg.DBDriver {
	case "mysql":
		u := &url.URL{
			Scheme: "mysql",
			User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
			Host:   fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
			Path:   "/" + cfg.DBName,
			RawQuery: url.Values{
				"charset":   {"utf8mb4"},
				"parseTime": {"true"},
				"loc":       {"UTC"},
			}.Encode(),
		}
		return u.String(), nil

	case "sqlite", "":
		return fmt.Sprintf("sqlite3://%s", cfg.DBPath), nil

	default:
		return "", fmt.Errorf("unsupported DB_DRIVER %q — valid values: sqlite, mysql", cfg.DBDriver)
	}
}
