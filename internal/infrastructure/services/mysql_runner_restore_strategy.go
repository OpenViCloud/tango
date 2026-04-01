package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mySQLRunnerRestoreStrategy struct {
	client appservices.BackupRunnerClient
}

func NewMySQLRunnerRestoreStrategy(client appservices.BackupRunnerClient) appservices.RestoreStrategy {
	return &mySQLRunnerRestoreStrategy{client: client}
}

func (s *mySQLRunnerRestoreStrategy) Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error {
	if target == nil || backup == nil {
		return fmt.Errorf("target and backup are required")
	}
	reader, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open restore artifact: %w", err)
	}
	defer reader.Close()

	return s.client.RunMySQLLogicalRestore(ctx, &appservices.MySQLLogicalRestoreRequest{
		Version:         target.Version,
		Host:            target.Host,
		Port:            target.Port,
		Username:        target.Username,
		Password:        target.PasswordEncrypted,
		Database:        target.DatabaseName,
		CompressionType: compressionTypeFromBackup(backup, localPath),
	}, reader)
}

func compressionTypeFromBackup(backup *domain.Backup, localPath string) domain.BackupCompressionType {
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
