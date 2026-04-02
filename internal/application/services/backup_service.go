package services

import (
	"context"
	"io"

	"tango/internal/domain"
)

type BackupStrategy interface {
	Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*BackupArtifact, error)
}

type RestoreStrategy interface {
	Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error
}

type BackupStrategyResolver interface {
	Resolve(dbType domain.DatabaseType, method domain.BackupMethod) (BackupStrategy, error)
}

type RestoreStrategyResolver interface {
	Resolve(dbType domain.DatabaseType, method domain.BackupMethod) (RestoreStrategy, error)
}

type StorageDriver interface {
	StoreFile(ctx context.Context, storage *domain.Storage, key string, localPath string) (*StoredObject, error)
	LoadFile(ctx context.Context, storage *domain.Storage, path string) (*LocalObject, error)
}

type StorageDriverResolver interface {
	Resolve(storageType domain.StorageType) (StorageDriver, error)
}

type BackupExecutor interface {
	ExecuteBackup(ctx context.Context, backupID string) error
}

type RestoreExecutor interface {
	ExecuteRestore(ctx context.Context, restoreID string) error
}

type BackupRunnerClient interface {
	RunMySQLLogicalDump(ctx context.Context, req *MySQLLogicalDumpRequest, writer io.Writer) (*BackupRunnerArtifact, error)
	RunMySQLLogicalRestore(ctx context.Context, req *MySQLLogicalRestoreRequest, reader io.Reader) error
	RunMariaDBLogicalDump(ctx context.Context, req *MariaDBLogicalDumpRequest, writer io.Writer) (*BackupRunnerArtifact, error)
	RunMariaDBLogicalRestore(ctx context.Context, req *MariaDBLogicalRestoreRequest, reader io.Reader) error
	RunPostgresLogicalDump(ctx context.Context, req *PostgresLogicalDumpRequest, writer io.Writer) (*BackupRunnerArtifact, error)
	RunPostgresLogicalRestore(ctx context.Context, req *PostgresLogicalRestoreRequest, reader io.Reader) error
	RunMongoLogicalDump(ctx context.Context, req *MongoLogicalDumpRequest, writer io.Writer) (*BackupRunnerArtifact, error)
	RunMongoLogicalRestore(ctx context.Context, req *MongoLogicalRestoreRequest, reader io.Reader) error
}

type BackupArtifact struct {
	FileName  string
	LocalPath string
	Metadata  map[string]any
}

type StoredObject struct {
	Path string
	Size int64
}

type LocalObject struct {
	Path    string
	Cleanup func() error
}

type MySQLLogicalDumpRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type MySQLLogicalRestoreRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type MariaDBLogicalDumpRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type MariaDBLogicalRestoreRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type MongoLogicalDumpRequest struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	AuthDatabase    string
	ConnectionURI   string
	CompressionType domain.BackupCompressionType
}

type MongoLogicalRestoreRequest struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	AuthDatabase    string
	ConnectionURI   string
	SourceDatabase  string
	CompressionType domain.BackupCompressionType
}

type PostgresLogicalDumpRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type PostgresLogicalRestoreRequest struct {
	Version         string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	CompressionType domain.BackupCompressionType
}

type BackupRunnerArtifact struct {
	FileName string
	Metadata map[string]any
}
