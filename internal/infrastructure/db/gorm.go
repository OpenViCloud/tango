package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(driver, databaseURL string) (*gorm.DB, error) {
	switch driver {
	case "", "postgres":
		db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return db, nil
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(databaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported db driver %q", driver)
	}
}

func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	return sqlDB.Close()
}

func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	return nil
}

func EnsureDatabase(ctx context.Context, driver, databaseURL string) error {
	switch driver {
	case "", "postgres":
		return ensurePostgresDatabase(ctx, databaseURL)
	case "sqlite":
		return ensureSQLitePath(databaseURL)
	default:
		return fmt.Errorf("unsupported db driver %q", driver)
	}
}

func Migrate(ctx context.Context, db *gorm.DB, models ...any) error {
	if err := db.WithContext(ctx).AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

func ensurePostgresDatabase(ctx context.Context, databaseURL string) error {
	targetURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}

	databaseName := strings.TrimPrefix(targetURL.Path, "/")
	if databaseName == "" {
		return fmt.Errorf("database url missing database name")
	}

	adminURL := *targetURL
	adminURL.Path = "/postgres"

	adminDB, err := sql.Open("pgx", adminURL.String())
	if err != nil {
		return fmt.Errorf("open admin database: %w", err)
	}
	defer adminDB.Close()

	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping admin database: %w", err)
	}

	var exists bool
	if err := adminDB.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`,
		databaseName,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check database %s: %w", databaseName, err)
	}
	if exists {
		return nil
	}

	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(databaseName)); err != nil {
		return fmt.Errorf("create database %s: %w", databaseName, err)
	}
	return nil
}

func ensureSQLitePath(databaseURL string) error {
	path := sqliteFilePath(databaseURL)
	if path == "" || path == ":memory:" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sqlite directory: %w", err)
	}
	return nil
}

func sqliteFilePath(databaseURL string) string {
	if databaseURL == "" {
		return ""
	}

	raw := databaseURL
	if strings.HasPrefix(raw, "file:") {
		raw = strings.TrimPrefix(raw, "file:")
	}
	if idx := strings.Index(raw, "?"); idx >= 0 {
		raw = raw[:idx]
	}
	return raw
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
