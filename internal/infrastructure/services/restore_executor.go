package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type restoreExecutor struct {
	restoreRepo     domain.RestoreRepository
	backupRepo      domain.BackupRepository
	sourceRepo      domain.DatabaseSourceRepository
	storageRepo     domain.StorageRepository
	cipher          appservices.SecretCipher
	restoreResolver appservices.RestoreStrategyResolver
	storageResolver appservices.StorageDriverResolver
}

func NewRestoreExecutor(
	restoreRepo domain.RestoreRepository,
	backupRepo domain.BackupRepository,
	sourceRepo domain.DatabaseSourceRepository,
	storageRepo domain.StorageRepository,
	cipher appservices.SecretCipher,
	restoreResolver appservices.RestoreStrategyResolver,
	storageResolver appservices.StorageDriverResolver,
) appservices.RestoreExecutor {
	return &restoreExecutor{
		restoreRepo:     restoreRepo,
		backupRepo:      backupRepo,
		sourceRepo:      sourceRepo,
		storageRepo:     storageRepo,
		cipher:          cipher,
		restoreResolver: restoreResolver,
		storageResolver: storageResolver,
	}
}

func (e *restoreExecutor) ExecuteRestore(ctx context.Context, restoreID string) error {
	slog.Default().Info("restore execute start", "restore_id", restoreID)
	restore, err := e.restoreRepo.GetByID(ctx, restoreID)
	if err != nil {
		return err
	}
	startedAt := time.Now().UTC()
	restore, err = e.restoreRepo.Update(ctx, restore.ID, domain.UpdateRestoreInput{
		Status:                  domain.RestoreStatusInProgress,
		TargetHost:              restore.TargetHost,
		TargetPort:              restore.TargetPort,
		TargetUsername:          restore.TargetUsername,
		TargetPasswordEncrypted: restore.TargetPasswordEncrypted,
		TargetDatabaseName:      restore.TargetDatabaseName,
		TargetAuthDatabase:      restore.TargetAuthDatabase,
		TargetURIEncrypted:      restore.TargetURIEncrypted,
		StartedAt:               &startedAt,
		CompletedAt:             restore.CompletedAt,
		DurationMs:              restore.DurationMs,
		FailMessage:             "",
		Metadata:                restore.Metadata,
	})
	if err != nil {
		return err
	}
	fail := func(runErr error) error {
		slog.Default().Error("restore execute failed", "restore_id", restore.ID, "err", runErr)
		completedAt := time.Now().UTC()
		_, updateErr := e.restoreRepo.Update(ctx, restore.ID, domain.UpdateRestoreInput{
			Status:                  domain.RestoreStatusFailed,
			TargetHost:              restore.TargetHost,
			TargetPort:              restore.TargetPort,
			TargetUsername:          restore.TargetUsername,
			TargetPasswordEncrypted: restore.TargetPasswordEncrypted,
			TargetDatabaseName:      restore.TargetDatabaseName,
			TargetAuthDatabase:      restore.TargetAuthDatabase,
			TargetURIEncrypted:      restore.TargetURIEncrypted,
			StartedAt:               restore.StartedAt,
			CompletedAt:             &completedAt,
			DurationMs:              completedAt.Sub(startedAt).Milliseconds(),
			FailMessage:             runErr.Error(),
			Metadata:                restore.Metadata,
		})
		if updateErr != nil {
			return fmt.Errorf("restore failed: %v; update failed: %w", runErr, updateErr)
		}
		return runErr
	}

	backup, err := e.backupRepo.GetByID(ctx, restore.BackupID)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("restore backup loaded",
		"restore_id", restore.ID,
		"backup_id", backup.ID,
		"storage_id", backup.StorageID,
		"file_path", backup.FilePath,
		"method", backup.BackupMethod,
	)
	storage, err := e.storageRepo.GetByID(ctx, backup.StorageID)
	if err != nil {
		return fail(err)
	}
	driver, err := e.storageResolver.Resolve(storage.Type)
	if err != nil {
		return fail(err)
	}
	localObject, err := driver.LoadFile(ctx, storage, backup.FilePath)
	if err != nil {
		return fail(err)
	}
	if localObject.Cleanup != nil {
		defer localObject.Cleanup()
	}

	target, err := e.resolveTargetSource(ctx, restore, backup)
	if err != nil {
		return fail(err)
	}
	slog.Default().Info("restore target resolved",
		"restore_id", restore.ID,
		"db_type", target.DBType,
		"version", target.Version,
		"database", target.DatabaseName,
		"host", target.Host,
		"port", target.Port,
	)
	restoreStrategy, err := e.restoreResolver.Resolve(target.DBType, backup.BackupMethod)
	if err != nil {
		return fail(err)
	}
	if err := restoreStrategy.Execute(ctx, target, backup, localObject.Path); err != nil {
		return fail(err)
	}

	completedAt := time.Now().UTC()
	_, err = e.restoreRepo.Update(ctx, restore.ID, domain.UpdateRestoreInput{
		Status:                  domain.RestoreStatusCompleted,
		TargetHost:              restore.TargetHost,
		TargetPort:              restore.TargetPort,
		TargetUsername:          restore.TargetUsername,
		TargetPasswordEncrypted: restore.TargetPasswordEncrypted,
		TargetDatabaseName:      restore.TargetDatabaseName,
		TargetAuthDatabase:      restore.TargetAuthDatabase,
		TargetURIEncrypted:      restore.TargetURIEncrypted,
		StartedAt:               restore.StartedAt,
		CompletedAt:             &completedAt,
		DurationMs:              completedAt.Sub(startedAt).Milliseconds(),
		FailMessage:             "",
		Metadata:                restore.Metadata,
	})
	if err == nil {
		slog.Default().Info("restore execute completed",
			"restore_id", restore.ID,
			"backup_id", backup.ID,
			"duration_ms", completedAt.Sub(startedAt).Milliseconds(),
		)
	}
	return err
}

func (e *restoreExecutor) resolveTargetSource(ctx context.Context, restore *domain.Restore, backup *domain.Backup) (*domain.DatabaseSource, error) {
	if strings.TrimSpace(restore.TargetHost) == "" && strings.TrimSpace(restore.DatabaseSourceID) == "" {
		restore.DatabaseSourceID = backup.DatabaseSourceID
	}
	if strings.TrimSpace(restore.DatabaseSourceID) != "" {
		source, err := e.sourceRepo.GetByID(ctx, restore.DatabaseSourceID)
		if err != nil {
			return nil, err
		}
		if e.cipher != nil {
			return decryptDatabaseSource(ctx, e.cipher, source)
		}
		return source, nil
	}
	copyValue := &domain.DatabaseSource{
		DBType:            databaseTypeFromBackup(backup),
		Version:           metadataString(backup.Metadata, "mysql_version"),
		Host:              restore.TargetHost,
		Port:              restore.TargetPort,
		Username:          restore.TargetUsername,
		PasswordEncrypted: restore.TargetPasswordEncrypted,
		DatabaseName:      restore.TargetDatabaseName,
		AuthDatabase:      restore.TargetAuthDatabase,
	}
	if e.cipher != nil && strings.TrimSpace(copyValue.PasswordEncrypted) != "" {
		password, err := e.cipher.Decrypt(ctx, copyValue.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt restore password: %w", err)
		}
		copyValue.PasswordEncrypted = password
	}
	return copyValue, nil
}

func databaseTypeFromBackup(backup *domain.Backup) domain.DatabaseType {
	value := metadataString(backup.Metadata, "db_type")
	dbType, err := domain.ValidateDatabaseType(value)
	if err == nil {
		return dbType
	}
	return domain.DatabaseTypeMySQL
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return value
}
