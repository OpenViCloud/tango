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

	_ "github.com/go-sql-driver/mysql"
)

type MariaDBExecutable string

const (
	MariaDBExecutableDump   MariaDBExecutable = "mariadb-dump"
	MariaDBExecutableClient MariaDBExecutable = "mariadb"
)

var mariaDBVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

var supportedMariaDBVersions = []string{"10.6", "12.1"}

type MariaDBConnectionConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func DetectMariaDBVersion(ctx context.Context, cfg MariaDBConnectionConfig) (string, error) {
	slog.Default().Info("detect mariadb version start",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"username", cfg.Username,
	)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Errorf("open mariadb connection: %w", err)
	}
	defer db.Close()

	var rawVersion string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&rawVersion); err != nil {
		return "", fmt.Errorf("query mariadb version: %w", err)
	}
	normalized, err := NormalizeMariaDBVersion(rawVersion)
	if err != nil {
		return "", err
	}
	slog.Default().Info("detect mariadb version done",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"raw_version", rawVersion,
		"normalized_version", normalized,
	)
	return normalized, nil
}

func NormalizeMariaDBVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	for _, version := range supportedMariaDBVersions {
		if raw == version {
			return raw, nil
		}
	}
	matches := mariaDBVersionPattern.FindStringSubmatch(raw)
	if len(matches) != 3 {
		return "", fmt.Errorf("unsupported mariadb version: %s", raw)
	}
	normalized := matches[1] + "." + matches[2]
	for _, version := range supportedMariaDBVersions {
		if normalized == version {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("unsupported mariadb version: %s", raw)
}

func GetMariaDBExecutable(version string, executable MariaDBExecutable, installDir string) (string, error) {
	normalized, err := NormalizeMariaDBVersion(version)
	if err != nil {
		return "", err
	}
	path := filepath.Join(strings.TrimSpace(installDir), "mariadb-"+normalized, "bin", string(executable))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("mariadb executable not found at %s: %w", path, err)
	}
	slog.Default().Info("resolve mariadb executable",
		"version", version,
		"normalized_version", normalized,
		"executable", executable,
		"path", path,
	)
	return path, nil
}

func VerifyMariaDBInstallation(installDir string) error {
	trimmedInstallDir := strings.TrimSpace(installDir)
	if trimmedInstallDir == "" {
		return fmt.Errorf("mariadb install dir is empty")
	}

	var verifyErrs []error
	for _, version := range supportedMariaDBVersions {
		for _, executable := range []MariaDBExecutable{MariaDBExecutableClient, MariaDBExecutableDump} {
			path := filepath.Join(trimmedInstallDir, "mariadb-"+version, "bin", string(executable))
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
