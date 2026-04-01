package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type postgresRunnerBackupStrategy struct {
	client appservices.BackupRunnerClient
}

func NewPostgresRunnerBackupStrategy(client appservices.BackupRunnerClient) appservices.BackupStrategy {
	return &postgresRunnerBackupStrategy{client: client}
}

func (s *postgresRunnerBackupStrategy) Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if source == nil || config == nil {
		return nil, fmt.Errorf("source and config are required")
	}
	fileName := buildPostgresArtifactName(source, config)
	outputPath := filepath.Join(os.TempDir(), "tango-runner-"+fileName)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("create runner output file: %w", err)
	}

	artifact, runErr := s.client.RunPostgresLogicalDump(ctx, &appservices.PostgresLogicalDumpRequest{
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
		"postgres_version": source.Version,
		"tool":             "pg_dump",
		"dump_format":      "custom",
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

func buildPostgresArtifactName(source *domain.DatabaseSource, config *domain.BackupConfig) string {
	base := source.DatabaseName
	if base == "" {
		base = "postgres-backup"
	}
	if config.CompressionType == domain.BackupCompressionGzip {
		return base + ".dump.gz"
	}
	return base + ".dump"
}
