package services

import (
	"context"
	"fmt"
	"os"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mariaDBRunnerRestoreStrategy struct {
	client appservices.BackupRunnerClient
}

func NewMariaDBRunnerRestoreStrategy(client appservices.BackupRunnerClient) appservices.RestoreStrategy {
	return &mariaDBRunnerRestoreStrategy{client: client}
}

func (s *mariaDBRunnerRestoreStrategy) Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error {
	if target == nil || backup == nil {
		return fmt.Errorf("target and backup are required")
	}
	reader, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open restore artifact: %w", err)
	}
	defer reader.Close()

	return s.client.RunMariaDBLogicalRestore(ctx, &appservices.MariaDBLogicalRestoreRequest{
		Version:         target.Version,
		Host:            target.Host,
		Port:            target.Port,
		Username:        target.Username,
		Password:        target.PasswordEncrypted,
		Database:        target.DatabaseName,
		CompressionType: compressionTypeFromBackup(backup, localPath),
	}, reader)
}
