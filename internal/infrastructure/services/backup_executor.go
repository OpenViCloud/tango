package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type backupExecutor struct {
	backupRepo       domain.BackupRepository
	sourceRepo       domain.DatabaseSourceRepository
	configRepo       domain.BackupConfigRepository
	storageRepo      domain.StorageRepository
	cipher           appservices.SecretCipher
	strategyResolver appservices.BackupStrategyResolver
	storageResolver  appservices.StorageDriverResolver
}

func NewBackupExecutor(
	backupRepo domain.BackupRepository,
	sourceRepo domain.DatabaseSourceRepository,
	configRepo domain.BackupConfigRepository,
	storageRepo domain.StorageRepository,
	cipher appservices.SecretCipher,
	strategyResolver appservices.BackupStrategyResolver,
	storageResolver appservices.StorageDriverResolver,
) appservices.BackupExecutor {
	return &backupExecutor{
		backupRepo:       backupRepo,
		sourceRepo:       sourceRepo,
		configRepo:       configRepo,
		storageRepo:      storageRepo,
		cipher:           cipher,
		strategyResolver: strategyResolver,
		storageResolver:  storageResolver,
	}
}

func (e *backupExecutor) ExecuteBackup(ctx context.Context, backupID string) error {
	slog.Default().Info("backup execute start", "backup_id", backupID)
	backup, err := e.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return err
	}
	startedAt := time.Now().UTC()
	backup, err = e.backupRepo.Update(ctx, backup.ID, domain.UpdateBackupInput{
		Status:         domain.BackupStatusInProgress,
		FileName:       backup.FileName,
		FilePath:       backup.FilePath,
		FileSizeBytes:  backup.FileSizeBytes,
		ChecksumSHA256: backup.ChecksumSHA256,
		StartedAt:      &startedAt,
		CompletedAt:    backup.CompletedAt,
		DurationMs:     backup.DurationMs,
		FailMessage:    "",
		Metadata:       backup.Metadata,
	})
	if err != nil {
		return err
	}

	fail := func(runErr error) error {
		slog.Default().Error("backup execute failed", "backup_id", backup.ID, "err", runErr)
		completedAt := time.Now().UTC()
		_, updateErr := e.backupRepo.Update(ctx, backup.ID, domain.UpdateBackupInput{
			Status:         domain.BackupStatusFailed,
			FileName:       backup.FileName,
			FilePath:       backup.FilePath,
			FileSizeBytes:  backup.FileSizeBytes,
			ChecksumSHA256: backup.ChecksumSHA256,
			StartedAt:      backup.StartedAt,
			CompletedAt:    &completedAt,
			DurationMs:     completedAt.Sub(startedAt).Milliseconds(),
			FailMessage:    runErr.Error(),
			Metadata:       backup.Metadata,
		})
		if updateErr != nil {
			return fmt.Errorf("backup failed: %v; update failed: %w", runErr, updateErr)
		}
		return runErr
	}

	source, err := e.sourceRepo.GetByID(ctx, backup.DatabaseSourceID)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("backup source loaded",
		"backup_id", backup.ID,
		"source_id", source.ID,
		"db_type", source.DBType,
		"version", source.Version,
		"database", source.DatabaseName,
		"host", source.Host,
		"port", source.Port,
	)
	if e.cipher != nil {
		source, err = decryptDatabaseSource(ctx, e.cipher, source)
		if err != nil {
			return fail(err)
		}
	}
	config, err := e.configRepo.GetByID(ctx, backup.BackupConfigID)
	if err != nil {
		return fail(err)
	}
	storage, err := e.storageRepo.GetByID(ctx, backup.StorageID)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("backup dependencies resolved",
		"backup_id", backup.ID,
		"config_id", config.ID,
		"storage_id", storage.ID,
		"storage_type", storage.Type,
		"compression", config.CompressionType,
		"method", backup.BackupMethod,
	)
	strategy, err := e.strategyResolver.Resolve(source.DBType, backup.BackupMethod)
	if err != nil {
		return fail(err)
	}
	driver, err := e.storageResolver.Resolve(storage.Type)
	if err != nil {
		return fail(err)
	}

	artifact, err := strategy.Execute(ctx, source, config)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("backup artifact created",
		"backup_id", backup.ID,
		"file_name", artifact.FileName,
		"local_path", artifact.LocalPath,
	)
	defer os.Remove(artifact.LocalPath)

	stored, err := driver.StoreFile(ctx, storage, buildBackupStorageKey(source, backup.ID, artifact.FileName, startedAt), artifact.LocalPath)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("backup artifact stored",
		"backup_id", backup.ID,
		"stored_path", stored.Path,
		"size_bytes", stored.Size,
	)
	checksum, err := fileSHA256(artifact.LocalPath)
	if err != nil {
		return fail(err)
	}

	completedAt := time.Now().UTC()
	mergedMetadata := make(map[string]any, len(backup.Metadata)+len(artifact.Metadata))
	for k, v := range backup.Metadata {
		mergedMetadata[k] = v
	}
	for k, v := range artifact.Metadata {
		mergedMetadata[k] = v
	}
	_, err = e.backupRepo.Update(ctx, backup.ID, domain.UpdateBackupInput{
		Status:         domain.BackupStatusCompleted,
		FileName:       artifact.FileName,
		FilePath:       stored.Path,
		FileSizeBytes:  stored.Size,
		ChecksumSHA256: checksum,
		StartedAt:      &startedAt,
		CompletedAt:    &completedAt,
		DurationMs:     completedAt.Sub(startedAt).Milliseconds(),
		FailMessage:    "",
		Metadata:       mergedMetadata,
	})
	if err == nil {
		slog.Default().Info("backup execute completed",
			"backup_id", backup.ID,
			"duration_ms", completedAt.Sub(startedAt).Milliseconds(),
			"checksum_sha256", checksum,
		)
	}
	return err
}

func decryptDatabaseSource(ctx context.Context, cipher appservices.SecretCipher, source *domain.DatabaseSource) (*domain.DatabaseSource, error) {
	copyValue := *source
	if strings.TrimSpace(copyValue.PasswordEncrypted) != "" {
		password, err := cipher.Decrypt(ctx, copyValue.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt database password: %w", err)
		}
		copyValue.PasswordEncrypted = password
	}
	if strings.TrimSpace(copyValue.ConnectionURIEncrypted) != "" {
		uri, err := cipher.Decrypt(ctx, copyValue.ConnectionURIEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt connection uri: %w", err)
		}
		copyValue.ConnectionURIEncrypted = uri
	}
	return &copyValue, nil
}

func buildBackupStorageKey(source *domain.DatabaseSource, backupID string, fileName string, ts time.Time) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = backupID
	}
	return filepath.Join(string(source.DBType), source.ID, ts.Format("2006"), ts.Format("01"), ts.Format("02"), backupID, name)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
