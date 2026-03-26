package onboarding

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	appconfig "tango/internal/config"
)

type Config struct {
	DBDriver    string `json:"db_driver"`
	DatabaseURL string `json:"database_url"`
	ChatChannel string `json:"chat_channel"`
	ChatModel   string `json:"chat_model"`
	APIKey      string `json:"api_key"`
}

func saveConfig(cfg Config) (string, error) {
	if err := prepareDatabase(cfg); err != nil {
		return "", err
	}
	return appconfig.SaveFile(&appconfig.Config{
		DBDriver:    cfg.DBDriver,
		DBUrl:       cfg.DatabaseURL,
		APIKey:      cfg.APIKey,
		ChatChannel: cfg.ChatChannel,
		ChatModel:   cfg.ChatModel,
	})
}

func configPath() (string, error) {
	return appconfig.Path()
}

func defaultDatabaseURL(driver string) string {
	switch driver {
	case "sqlite":
		return "file:tango.db?_foreign_keys=on"
	default:
		return "postgres://postgres:postgres@localhost:5432/tango?sslmode=disable"
	}
}

func maskSecret(value string) string {
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 6 {
		return strings.Repeat("•", len(value))
	}
	return value[:3] + strings.Repeat("•", len(value)-6) + value[len(value)-3:]
}

func prepareDatabase(cfg Config) error {
	switch cfg.DBDriver {
	case "sqlite":
		return ensureSQLiteDBFile(cfg.DatabaseURL)
	case "postgres":
		return validatePostgresURL(cfg.DatabaseURL)
	default:
		return fmt.Errorf("unsupported db driver %q", cfg.DBDriver)
	}
}

func validatePostgresURL(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("invalid postgres url: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("invalid postgres url: scheme must be postgres or postgresql")
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid postgres url: missing host")
	}
	if strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("invalid postgres url: missing database name")
	}
	return nil
}

func ensureSQLiteDBFile(databaseURL string) error {
	path := sqliteFilePath(databaseURL)
	if path == "" || path == ":memory:" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sqlite directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("create sqlite database file: %w", err)
	}
	return file.Close()
}

func sqliteFilePath(databaseURL string) string {
	raw := strings.TrimSpace(databaseURL)
	if strings.HasPrefix(raw, "file:") {
		raw = strings.TrimPrefix(raw, "file:")
	}
	if idx := strings.Index(raw, "?"); idx >= 0 {
		raw = raw[:idx]
	}
	return raw
}
