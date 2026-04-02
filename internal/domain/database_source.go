package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypeMariaDB  DatabaseType = "mariadb"
	DatabaseTypeMongoDB  DatabaseType = "mongodb"
)

var (
	ErrDatabaseSourceNotFound = errors.New("database source not found")
	ErrStorageNotFound        = errors.New("storage not found")
	ErrStorageInUse           = errors.New("storage is in use")
	ErrBackupConfigNotFound   = errors.New("backup config not found")
	ErrBackupNotFound         = errors.New("backup not found")
	ErrRestoreNotFound        = errors.New("restore not found")
	ErrInvalidBackupConfig    = errors.New("invalid backup config")
)

type DatabaseSource struct {
	ID                     string
	Name                   string
	DBType                 DatabaseType
	Host                   string
	Port                   int
	Username               string
	PasswordEncrypted      string
	DatabaseName           string
	Version                string
	IsTLSEnabled           bool
	AuthDatabase           string
	ConnectionURIEncrypted string
	ResourceID             string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type CreateDatabaseSourceInput struct {
	ID                     string
	Name                   string
	DBType                 DatabaseType
	Host                   string
	Port                   int
	Username               string
	PasswordEncrypted      string
	DatabaseName           string
	Version                string
	IsTLSEnabled           bool
	AuthDatabase           string
	ConnectionURIEncrypted string
	ResourceID             string
}

type UpdateDatabaseSourceInput struct {
	Name                   string
	Host                   string
	Port                   int
	Username               string
	PasswordEncrypted      string
	DatabaseName           string
	Version                string
	IsTLSEnabled           bool
	AuthDatabase           string
	ConnectionURIEncrypted string
	ResourceID             string
}

func ValidateDatabaseType(value string) (DatabaseType, error) {
	switch DatabaseType(strings.TrimSpace(value)) {
	case DatabaseTypePostgres, DatabaseTypeMySQL, DatabaseTypeMariaDB, DatabaseTypeMongoDB:
		return DatabaseType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

type DatabaseSourceRepository interface {
	Create(ctx context.Context, input CreateDatabaseSourceInput) (*DatabaseSource, error)
	GetByID(ctx context.Context, id string) (*DatabaseSource, error)
	List(ctx context.Context) ([]*DatabaseSource, error)
	ListByResourceID(ctx context.Context, resourceID string) ([]*DatabaseSource, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, input UpdateDatabaseSourceInput) (*DatabaseSource, error)
}
