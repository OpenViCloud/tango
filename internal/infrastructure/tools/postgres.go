package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresExecutable string

const (
	PostgresExecutableDump    PostgresExecutable = "pg_dump"
	PostgresExecutableRestore PostgresExecutable = "pg_restore"
)

var postgresVersionPattern = regexp.MustCompile(`^(\d+)`)

var supportedPostgresVersions = []string{"12", "13", "14", "15", "16", "17", "18"}

type PostgresConnectionConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func DetectPostgresVersion(ctx context.Context, cfg PostgresConnectionConfig) (string, error) {
	slog.Default().Info("detect postgres version start",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"username", cfg.Username,
	)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return "", fmt.Errorf("open postgres connection: %w", err)
	}
	defer db.Close()

	var rawVersion string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&rawVersion); err != nil {
		return "", fmt.Errorf("query postgres version: %w", err)
	}
	normalized, err := NormalizePostgresVersion(rawVersion)
	if err != nil {
		return "", err
	}
	slog.Default().Info("detect postgres version done",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"raw_version", rawVersion,
		"normalized_version", normalized,
	)
	return normalized, nil
}

func NormalizePostgresVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	for _, version := range supportedPostgresVersions {
		if raw == version {
			return version, nil
		}
	}
	matches := postgresVersionPattern.FindStringSubmatch(raw)
	if len(matches) != 2 {
		return "", fmt.Errorf("unsupported postgres version: %s", raw)
	}
	major := matches[1]
	for _, version := range supportedPostgresVersions {
		if major == version {
			return version, nil
		}
	}
	return "", fmt.Errorf("unsupported postgres version: %s", raw)
}

func GetPostgresExecutable(version string, executable PostgresExecutable, installDir string) (string, error) {
	normalized, err := NormalizePostgresVersion(version)
	if err != nil {
		return "", err
	}
	path := filepath.Join(strings.TrimSpace(installDir), normalized, "bin", string(executable))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("postgres executable not found at %s: %w", path, err)
	}
	slog.Default().Info("resolve postgres executable",
		"version", version,
		"normalized_version", normalized,
		"executable", executable,
		"path", path,
	)
	return path, nil
}

func VerifyPostgresInstallation(installDir string) error {
	trimmedInstallDir := strings.TrimSpace(installDir)
	if trimmedInstallDir == "" {
		return fmt.Errorf("postgres install dir is empty")
	}
	var verifyErrs []error
	for _, version := range supportedPostgresVersions {
		for _, executable := range []PostgresExecutable{PostgresExecutableDump, PostgresExecutableRestore} {
			path := filepath.Join(trimmedInstallDir, version, "bin", string(executable))
			if _, err := os.Stat(path); err != nil {
				verifyErrs = append(verifyErrs, fmt.Errorf("%s missing at %s: %w", executable, path, err))
			}
		}
	}
	if len(verifyErrs) > 0 {
		return errors.Join(verifyErrs...)
	}
	return nil
}
