package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mongoRunnerRestoreStrategy struct {
	client appservices.BackupRunnerClient
}

func NewMongoRunnerRestoreStrategy(client appservices.BackupRunnerClient) appservices.RestoreStrategy {
	return &mongoRunnerRestoreStrategy{client: client}
}

func (s *mongoRunnerRestoreStrategy) Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error {
	if target == nil || backup == nil {
		return fmt.Errorf("target and backup are required")
	}
	reader, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open restore artifact: %w", err)
	}
	defer reader.Close()

	return s.client.RunMongoLogicalRestore(ctx, &appservices.MongoLogicalRestoreRequest{
		Host:            target.Host,
		Port:            target.Port,
		Username:        target.Username,
		Password:        target.PasswordEncrypted,
		Database:        target.DatabaseName,
		AuthDatabase:    target.AuthDatabase,
		ConnectionURI:   target.ConnectionURIEncrypted,
		SourceDatabase:  backupMetadataString(backup, "database_name"),
		CompressionType: mongoCompressionTypeFromBackup(backup, localPath),
	}, reader)
}

func mongoCompressionTypeFromBackup(backup *domain.Backup, localPath string) domain.BackupCompressionType {
	if backup != nil && backup.Metadata != nil {
		if value, ok := backup.Metadata["compression_type"].(string); ok {
			if compression := domain.BackupCompressionType(strings.TrimSpace(value)); compression != "" {
				return compression
			}
		}
	}
	if strings.HasSuffix(strings.ToLower(localPath), ".gz") {
		return domain.BackupCompressionGzip
	}
	return domain.BackupCompressionNone
}

func backupMetadataString(backup *domain.Backup, key string) string {
	if backup == nil || backup.Metadata == nil {
		return ""
	}
	value, _ := backup.Metadata[key].(string)
	return strings.TrimSpace(value)
}
