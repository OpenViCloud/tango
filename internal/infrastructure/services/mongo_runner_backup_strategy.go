package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mongoRunnerBackupStrategy struct {
	client appservices.BackupRunnerClient
}

func NewMongoRunnerBackupStrategy(client appservices.BackupRunnerClient) appservices.BackupStrategy {
	return &mongoRunnerBackupStrategy{client: client}
}

func (s *mongoRunnerBackupStrategy) Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if source == nil || config == nil {
		return nil, fmt.Errorf("source and config are required")
	}
	fileName := buildMongoArtifactName(source, config)
	outputPath := filepath.Join(os.TempDir(), "tango-runner-"+fileName)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("create runner output file: %w", err)
	}

	artifact, runErr := s.client.RunMongoLogicalDump(ctx, &appservices.MongoLogicalDumpRequest{
		Host:            source.Host,
		Port:            source.Port,
		Username:        source.Username,
		Password:        source.PasswordEncrypted,
		Database:        source.DatabaseName,
		AuthDatabase:    source.AuthDatabase,
		ConnectionURI:   source.ConnectionURIEncrypted,
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
		"tool":             "mongodump",
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

func buildMongoArtifactName(source *domain.DatabaseSource, config *domain.BackupConfig) string {
	base := source.DatabaseName
	if base == "" {
		base = "mongodb-backup"
	}
	if config.CompressionType == domain.BackupCompressionGzip {
		return base + ".archive.gz"
	}
	return base + ".archive"
}
