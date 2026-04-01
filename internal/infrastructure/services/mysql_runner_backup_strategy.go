package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mySQLRunnerBackupStrategy struct {
	client appservices.BackupRunnerClient
}

func NewMySQLRunnerBackupStrategy(client appservices.BackupRunnerClient) appservices.BackupStrategy {
	return &mySQLRunnerBackupStrategy{client: client}
}

func (s *mySQLRunnerBackupStrategy) Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if source == nil || config == nil {
		return nil, fmt.Errorf("source and config are required")
	}
	fileName := buildMySQLArtifactName(source, config)
	outputPath := filepath.Join(os.TempDir(), "tango-runner-"+fileName)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("create runner output file: %w", err)
	}

	artifact, runErr := s.client.RunMySQLLogicalDump(ctx, &appservices.MySQLLogicalDumpRequest{
		Version:         source.Version,
		Host:            source.Host,
		Port:            source.Port,
		Username:        source.Username,
		Password:        source.PasswordEncrypted,
		Database:        source.DatabaseName,
		CompressionType: config.CompressionType,
	}, outFile)
	closeErr := outFile.Close()
	if runErr != nil {
		_ = os.Remove(outputPath)
		return nil, runErr
	}
	if closeErr != nil {
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("close runner output file: %w", closeErr)
	}
	if artifact == nil {
		artifact = &appservices.BackupRunnerArtifact{}
	}
	resolvedName := artifact.FileName
	if resolvedName == "" {
		resolvedName = fileName
	}
	metadata := map[string]any{
		"db_type":          source.DBType,
		"database_name":    source.DatabaseName,
		"backup_method":    config.BackupMethod,
		"compression_type": config.CompressionType,
		"mysql_version":    source.Version,
		"tool":             "mysqldump",
	}
	for k, v := range artifact.Metadata {
		metadata[k] = v
	}
	return &appservices.BackupArtifact{
		FileName:  resolvedName,
		LocalPath: outputPath,
		Metadata:  metadata,
	}, nil
}
