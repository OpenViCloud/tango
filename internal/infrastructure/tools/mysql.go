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

type MySQLExecutable string

const (
	MySQLExecutableDump   MySQLExecutable = "mysqldump"
	MySQLExecutableClient MySQLExecutable = "mysql"
)

var mysqlVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

var supportedMySQLVersions = []string{"5.7", "8.0", "8.4", "9"}

type MySQLConnectionConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func DetectMySQLVersion(ctx context.Context, cfg MySQLConnectionConfig) (string, error) {
	slog.Default().Info("detect mysql version start",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"username", cfg.Username,
	)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Errorf("open mysql connection: %w", err)
	}
	defer db.Close()

	var rawVersion string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&rawVersion); err != nil {
		return "", fmt.Errorf("query mysql version: %w", err)
	}
	normalized, err := NormalizeMySQLVersion(rawVersion)
	if err != nil {
		return "", err
	}
	slog.Default().Info("detect mysql version done",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"raw_version", rawVersion,
		"normalized_version", normalized,
	)
	return normalized, nil
}

func NormalizeMySQLVersion(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "5.7", "8.0", "8.4", "9":
		return raw, nil
	}
	matches := mysqlVersionPattern.FindStringSubmatch(raw)
	if len(matches) != 3 {
		return "", fmt.Errorf("unsupported mysql version: %s", raw)
	}
	major := matches[1]
	minor := matches[2]
	switch {
	case major == "5" && minor == "7":
		return "5.7", nil
	case major == "8" && minor == "0":
		return "8.0", nil
	case major == "8" && minor == "4":
		return "8.4", nil
	case major == "9":
		return "9", nil
	default:
		return "", fmt.Errorf("unsupported mysql version: %s", raw)
	}
}

func GetMySQLExecutable(version string, executable MySQLExecutable, installDir string) (string, error) {
	normalized, err := NormalizeMySQLVersion(version)
	if err != nil {
		return "", err
	}
	path := filepath.Join(strings.TrimSpace(installDir), "mysql-"+normalized, "bin", string(executable))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("mysql executable not found at %s: %w", path, err)
	}
	slog.Default().Info("resolve mysql executable",
		"version", version,
		"normalized_version", normalized,
		"executable", executable,
		"path", path,
	)
	return path, nil
}

func VerifyMySQLInstallation(installDir string) error {
	trimmedInstallDir := strings.TrimSpace(installDir)
	if trimmedInstallDir == "" {
		return fmt.Errorf("mysql install dir is empty")
	}

	var verifyErrs []error
	for _, version := range supportedMySQLVersions {
		for _, executable := range []MySQLExecutable{MySQLExecutableClient, MySQLExecutableDump} {
			path := filepath.Join(trimmedInstallDir, "mysql-"+version, "bin", string(executable))
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
