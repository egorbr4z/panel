// Package database opens the GORM connection (SQLite by default, PostgreSQL
// optional) and runs auto-migration. SQLite is configured in WAL mode with a
// bounded cache to stay friendly on a 1 GB host.
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egorbr4z/panel/internal/config"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/glebarez/sqlite" // pure-Go sqlite (modernc) -> no CGO, static binary
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates the GORM DB handle for the configured driver and migrates schema.
func Open(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Warn),
		PrepareStmt: true,
	}

	var db *gorm.DB
	var err error

	switch cfg.DBDriver {
	case "sqlite":
		if dir := filepath.Dir(cfg.DBDSN); dir != "" {
			if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
				return nil, fmt.Errorf("create db dir: %w", mkErr)
			}
		}
		// WAL + busy_timeout keeps the single-writer batched-flush model smooth.
		dsn := cfg.DBDSN + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
		db, err = gorm.Open(sqlite.Open(dsn), gormCfg)
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DBDSN), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported db driver %q", cfg.DBDriver)
	}
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	// SQLite is happiest with a single writer; keep the pool small everywhere
	// to bound memory on the tiny target host.
	if cfg.DBDriver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(10)
	}
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	return db, nil
}
