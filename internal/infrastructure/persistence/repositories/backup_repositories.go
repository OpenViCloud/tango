package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tango/internal/domain"
	"tango/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

type DatabaseSourceRepository struct{ db *gorm.DB }
type StorageRepository struct{ db *gorm.DB }
type BackupConfigRepository struct{ db *gorm.DB }
type BackupRepository struct{ db *gorm.DB }
type RestoreRepository struct{ db *gorm.DB }

func NewDatabaseSourceRepository(db *gorm.DB) *DatabaseSourceRepository {
	return &DatabaseSourceRepository{db: db}
}
func NewStorageRepository(db *gorm.DB) *StorageRepository { return &StorageRepository{db: db} }
func NewBackupConfigRepository(db *gorm.DB) *BackupConfigRepository {
	return &BackupConfigRepository{db: db}
}
func NewBackupRepository(db *gorm.DB) *BackupRepository   { return &BackupRepository{db: db} }
func NewRestoreRepository(db *gorm.DB) *RestoreRepository { return &RestoreRepository{db: db} }

func (r *DatabaseSourceRepository) Create(ctx context.Context, input domain.CreateDatabaseSourceInput) (*domain.DatabaseSource, error) {
	now := time.Now().UTC()
	record := models.DatabaseSourceRecord{
		ID:                     input.ID,
		Name:                   input.Name,
		DBType:                 string(input.DBType),
		Host:                   input.Host,
		Port:                   input.Port,
		Username:               input.Username,
		PasswordEncrypted:      input.PasswordEncrypted,
		DatabaseName:           input.DatabaseName,
		Version:                input.Version,
		IsTLSEnabled:           input.IsTLSEnabled,
		AuthDatabase:           input.AuthDatabase,
		ConnectionURIEncrypted: input.ConnectionURIEncrypted,
		ResourceID:             input.ResourceID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create database source: %w", err)
	}
	return toDomainDatabaseSource(&record), nil
}

func (r *DatabaseSourceRepository) GetByID(ctx context.Context, id string) (*domain.DatabaseSource, error) {
	var record models.DatabaseSourceRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDatabaseSourceNotFound
		}
		return nil, fmt.Errorf("get database source: %w", err)
	}
	return toDomainDatabaseSource(&record), nil
}

func (r *DatabaseSourceRepository) List(ctx context.Context) ([]*domain.DatabaseSource, error) {
	var records []models.DatabaseSourceRecord
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list database sources: %w", err)
	}
	items := make([]*domain.DatabaseSource, 0, len(records))
	for i := range records {
		items = append(items, toDomainDatabaseSource(&records[i]))
	}
	return items, nil
}

func (r *DatabaseSourceRepository) ListByResourceID(ctx context.Context, resourceID string) ([]*domain.DatabaseSource, error) {
	var records []models.DatabaseSourceRecord
	if err := r.db.WithContext(ctx).Where("resource_id = ?", resourceID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list database sources by resource: %w", err)
	}
	items := make([]*domain.DatabaseSource, 0, len(records))
	for i := range records {
		items = append(items, toDomainDatabaseSource(&records[i]))
	}
	return items, nil
}

func (r *DatabaseSourceRepository) Update(ctx context.Context, id string, input domain.UpdateDatabaseSourceInput) (*domain.DatabaseSource, error) {
	result := r.db.WithContext(ctx).Model(&models.DatabaseSourceRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"name":                     input.Name,
			"host":                     input.Host,
			"port":                     input.Port,
			"username":                 input.Username,
			"password_encrypted":       input.PasswordEncrypted,
			"database_name":            input.DatabaseName,
			"version":                  input.Version,
			"is_tls_enabled":           input.IsTLSEnabled,
			"auth_database":            input.AuthDatabase,
			"connection_uri_encrypted": input.ConnectionURIEncrypted,
			"resource_id":              input.ResourceID,
			"updated_at":               time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update database source: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrDatabaseSourceNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *DatabaseSourceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source models.DatabaseSourceRecord
		if err := tx.Where("id = ?", id).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrDatabaseSourceNotFound
			}
			return fmt.Errorf("get database source for delete: %w", err)
		}

		var backupIDs []string
		if err := tx.Model(&models.BackupRecord{}).
			Where("database_source_id = ?", id).
			Pluck("id", &backupIDs).Error; err != nil {
			return fmt.Errorf("list source backup ids: %w", err)
		}

		if len(backupIDs) > 0 {
			if err := tx.Where("backup_id IN ?", backupIDs).Delete(&models.RestoreRecord{}).Error; err != nil {
				return fmt.Errorf("delete restore records by backup ids: %w", err)
			}
		}
		if err := tx.Where("database_source_id = ?", id).Delete(&models.RestoreRecord{}).Error; err != nil {
			return fmt.Errorf("delete restore records by source id: %w", err)
		}
		if err := tx.Where("database_source_id = ?", id).Delete(&models.BackupRecord{}).Error; err != nil {
			return fmt.Errorf("delete backup records: %w", err)
		}
		if err := tx.Where("database_source_id = ?", id).Delete(&models.BackupConfigRecord{}).Error; err != nil {
			return fmt.Errorf("delete backup config records: %w", err)
		}
		if err := tx.Where("id = ?", id).Delete(&models.DatabaseSourceRecord{}).Error; err != nil {
			return fmt.Errorf("delete database source: %w", err)
		}
		return nil
	})
}

func (r *StorageRepository) Create(ctx context.Context, input domain.CreateStorageInput) (*domain.Storage, error) {
	configJSON, err := marshalJSONMap(input.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal storage config: %w", err)
	}
	now := time.Now().UTC()
	record := models.StorageRecord{
		ID:                   input.ID,
		Name:                 input.Name,
		Type:                 string(input.Type),
		ConfigJSON:           configJSON,
		CredentialsEncrypted: input.CredentialsEncrypted,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create storage: %w", err)
	}
	return toDomainStorage(&record)
}

func (r *StorageRepository) GetByID(ctx context.Context, id string) (*domain.Storage, error) {
	var record models.StorageRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrStorageNotFound
		}
		return nil, fmt.Errorf("get storage: %w", err)
	}
	return toDomainStorage(&record)
}

func (r *StorageRepository) List(ctx context.Context) ([]*domain.Storage, error) {
	var records []models.StorageRecord
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list storages: %w", err)
	}
	items := make([]*domain.Storage, 0, len(records))
	for i := range records {
		item, err := toDomainStorage(&records[i])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *StorageRepository) Update(ctx context.Context, id string, input domain.UpdateStorageInput) (*domain.Storage, error) {
	configJSON, err := marshalJSONMap(input.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal storage config: %w", err)
	}
	result := r.db.WithContext(ctx).Model(&models.StorageRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"name":                  input.Name,
			"type":                  string(input.Type),
			"config_json":           configJSON,
			"credentials_encrypted": input.CredentialsEncrypted,
			"updated_at":            time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update storage: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrStorageNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *StorageRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var storage models.StorageRecord
		if err := tx.Where("id = ?", id).First(&storage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrStorageNotFound
			}
			return fmt.Errorf("get storage for delete: %w", err)
		}

		var configCount int64
		if err := tx.Model(&models.BackupConfigRecord{}).Where("storage_id = ?", id).Count(&configCount).Error; err != nil {
			return fmt.Errorf("count backup configs by storage: %w", err)
		}
		if configCount > 0 {
			return domain.ErrStorageInUse
		}

		var backupCount int64
		if err := tx.Model(&models.BackupRecord{}).Where("storage_id = ?", id).Count(&backupCount).Error; err != nil {
			return fmt.Errorf("count backups by storage: %w", err)
		}
		if backupCount > 0 {
			return domain.ErrStorageInUse
		}

		if err := tx.Where("id = ?", id).Delete(&models.StorageRecord{}).Error; err != nil {
			return fmt.Errorf("delete storage: %w", err)
		}
		return nil
	})
}

func (r *BackupConfigRepository) Create(ctx context.Context, input domain.CreateBackupConfigInput) (*domain.BackupConfig, error) {
	now := time.Now().UTC()
	record := models.BackupConfigRecord{
		ID:               input.ID,
		DatabaseSourceID: input.DatabaseSourceID,
		StorageID:        input.StorageID,
		IsEnabled:        input.IsEnabled,
		ScheduleType:     string(input.ScheduleType),
		TimeOfDay:        input.TimeOfDay,
		IntervalHours:    input.IntervalHours,
		RetentionType:    string(input.RetentionType),
		RetentionDays:    input.RetentionDays,
		RetentionCount:   input.RetentionCount,
		IsRetryIfFailed:  input.IsRetryIfFailed,
		MaxRetryCount:    input.MaxRetryCount,
		EncryptionType:   string(input.EncryptionType),
		CompressionType:  string(input.CompressionType),
		BackupMethod:     string(input.BackupMethod),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create backup config: %w", err)
	}
	return toDomainBackupConfig(&record), nil
}

func (r *BackupConfigRepository) GetByID(ctx context.Context, id string) (*domain.BackupConfig, error) {
	var record models.BackupConfigRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBackupConfigNotFound
		}
		return nil, fmt.Errorf("get backup config: %w", err)
	}
	return toDomainBackupConfig(&record), nil
}

func (r *BackupConfigRepository) GetByDatabaseSourceID(ctx context.Context, databaseSourceID string) (*domain.BackupConfig, error) {
	var record models.BackupConfigRecord
	if err := r.db.WithContext(ctx).Where("database_source_id = ?", databaseSourceID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBackupConfigNotFound
		}
		return nil, fmt.Errorf("get backup config by database source: %w", err)
	}
	return toDomainBackupConfig(&record), nil
}

func (r *BackupConfigRepository) Update(ctx context.Context, id string, input domain.UpdateBackupConfigInput) (*domain.BackupConfig, error) {
	result := r.db.WithContext(ctx).Model(&models.BackupConfigRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"storage_id":         input.StorageID,
			"is_enabled":         input.IsEnabled,
			"schedule_type":      string(input.ScheduleType),
			"time_of_day":        input.TimeOfDay,
			"interval_hours":     input.IntervalHours,
			"retention_type":     string(input.RetentionType),
			"retention_days":     input.RetentionDays,
			"retention_count":    input.RetentionCount,
			"is_retry_if_failed": input.IsRetryIfFailed,
			"max_retry_count":    input.MaxRetryCount,
			"encryption_type":    string(input.EncryptionType),
			"compression_type":   string(input.CompressionType),
			"backup_method":      string(input.BackupMethod),
			"updated_at":         time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update backup config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrBackupConfigNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *BackupRepository) Create(ctx context.Context, input domain.CreateBackupInput) (*domain.Backup, error) {
	metadataJSON, err := marshalJSONMap(input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata: %w", err)
	}
	record := models.BackupRecord{
		ID:               input.ID,
		DatabaseSourceID: input.DatabaseSourceID,
		BackupConfigID:   input.BackupConfigID,
		StorageID:        input.StorageID,
		Status:           string(input.Status),
		BackupMethod:     string(input.BackupMethod),
		FileName:         input.FileName,
		FilePath:         input.FilePath,
		FileSizeBytes:    input.FileSizeBytes,
		ChecksumSHA256:   input.ChecksumSHA256,
		StartedAt:        input.StartedAt,
		CompletedAt:      input.CompletedAt,
		DurationMs:       input.DurationMs,
		FailMessage:      input.FailMessage,
		EncryptionType:   string(input.EncryptionType),
		MetadataJSON:     metadataJSON,
		CreatedAt:        time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}
	return r.GetByID(ctx, input.ID)
}

func (r *BackupRepository) GetByID(ctx context.Context, id string) (*domain.Backup, error) {
	var record models.BackupRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBackupNotFound
		}
		return nil, fmt.Errorf("get backup: %w", err)
	}
	return toDomainBackup(&record)
}

func (r *BackupRepository) ListByDatabaseSourceID(ctx context.Context, databaseSourceID string) ([]*domain.Backup, error) {
	var records []models.BackupRecord
	if err := r.db.WithContext(ctx).Where("database_source_id = ?", databaseSourceID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	items := make([]*domain.Backup, 0, len(records))
	for i := range records {
		item, err := toDomainBackup(&records[i])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *BackupRepository) Update(ctx context.Context, id string, input domain.UpdateBackupInput) (*domain.Backup, error) {
	metadataJSON, err := marshalJSONMap(input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata: %w", err)
	}
	result := r.db.WithContext(ctx).Model(&models.BackupRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":          string(input.Status),
			"file_name":       input.FileName,
			"file_path":       input.FilePath,
			"file_size_bytes": input.FileSizeBytes,
			"checksum_sha256": input.ChecksumSHA256,
			"started_at":      input.StartedAt,
			"completed_at":    input.CompletedAt,
			"duration_ms":     input.DurationMs,
			"fail_message":    input.FailMessage,
			"metadata_json":   metadataJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update backup: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrBackupNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *RestoreRepository) Create(ctx context.Context, input domain.CreateRestoreInput) (*domain.Restore, error) {
	metadataJSON, err := marshalJSONMap(input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal restore metadata: %w", err)
	}
	record := models.RestoreRecord{
		ID:                      input.ID,
		BackupID:                input.BackupID,
		DatabaseSourceID:        input.DatabaseSourceID,
		Status:                  string(input.Status),
		TargetHost:              input.TargetHost,
		TargetPort:              input.TargetPort,
		TargetUsername:          input.TargetUsername,
		TargetPasswordEncrypted: input.TargetPasswordEncrypted,
		TargetDatabaseName:      input.TargetDatabaseName,
		TargetAuthDatabase:      input.TargetAuthDatabase,
		TargetURIEncrypted:      input.TargetURIEncrypted,
		StartedAt:               input.StartedAt,
		CompletedAt:             input.CompletedAt,
		DurationMs:              input.DurationMs,
		FailMessage:             input.FailMessage,
		MetadataJSON:            metadataJSON,
		CreatedAt:               time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create restore: %w", err)
	}
	return r.GetByID(ctx, input.ID)
}

func (r *RestoreRepository) GetByID(ctx context.Context, id string) (*domain.Restore, error) {
	var record models.RestoreRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRestoreNotFound
		}
		return nil, fmt.Errorf("get restore: %w", err)
	}
	return toDomainRestore(&record)
}

func (r *RestoreRepository) Update(ctx context.Context, id string, input domain.UpdateRestoreInput) (*domain.Restore, error) {
	metadataJSON, err := marshalJSONMap(input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal restore metadata: %w", err)
	}
	result := r.db.WithContext(ctx).Model(&models.RestoreRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":                    string(input.Status),
			"target_host":               input.TargetHost,
			"target_port":               input.TargetPort,
			"target_username":           input.TargetUsername,
			"target_password_encrypted": input.TargetPasswordEncrypted,
			"target_database_name":      input.TargetDatabaseName,
			"target_auth_database":      input.TargetAuthDatabase,
			"target_uri_encrypted":      input.TargetURIEncrypted,
			"started_at":                input.StartedAt,
			"completed_at":              input.CompletedAt,
			"duration_ms":               input.DurationMs,
			"fail_message":              input.FailMessage,
			"metadata_json":             metadataJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("update restore: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrRestoreNotFound
	}
	return r.GetByID(ctx, id)
}

func marshalJSONMap(value map[string]any) (string, error) {
	if len(value) == 0 {
		return "{}", nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func unmarshalJSONMap(value string) (map[string]any, error) {
	if value == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func toDomainDatabaseSource(record *models.DatabaseSourceRecord) *domain.DatabaseSource {
	return &domain.DatabaseSource{
		ID:                     record.ID,
		Name:                   record.Name,
		DBType:                 domain.DatabaseType(record.DBType),
		Host:                   record.Host,
		Port:                   record.Port,
		Username:               record.Username,
		PasswordEncrypted:      record.PasswordEncrypted,
		DatabaseName:           record.DatabaseName,
		Version:                record.Version,
		IsTLSEnabled:           record.IsTLSEnabled,
		AuthDatabase:           record.AuthDatabase,
		ConnectionURIEncrypted: record.ConnectionURIEncrypted,
		ResourceID:             record.ResourceID,
		CreatedAt:              record.CreatedAt,
		UpdatedAt:              record.UpdatedAt,
	}
}

func toDomainStorage(record *models.StorageRecord) (*domain.Storage, error) {
	config, err := unmarshalJSONMap(record.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal storage config: %w", err)
	}
	return &domain.Storage{
		ID:                   record.ID,
		Name:                 record.Name,
		Type:                 domain.StorageType(record.Type),
		Config:               config,
		CredentialsEncrypted: record.CredentialsEncrypted,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}, nil
}

func toDomainBackupConfig(record *models.BackupConfigRecord) *domain.BackupConfig {
	return &domain.BackupConfig{
		ID:               record.ID,
		DatabaseSourceID: record.DatabaseSourceID,
		StorageID:        record.StorageID,
		IsEnabled:        record.IsEnabled,
		ScheduleType:     domain.BackupScheduleType(record.ScheduleType),
		TimeOfDay:        record.TimeOfDay,
		IntervalHours:    record.IntervalHours,
		RetentionType:    domain.BackupRetentionType(record.RetentionType),
		RetentionDays:    record.RetentionDays,
		RetentionCount:   record.RetentionCount,
		IsRetryIfFailed:  record.IsRetryIfFailed,
		MaxRetryCount:    record.MaxRetryCount,
		EncryptionType:   domain.BackupEncryptionType(record.EncryptionType),
		CompressionType:  domain.BackupCompressionType(record.CompressionType),
		BackupMethod:     domain.BackupMethod(record.BackupMethod),
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func toDomainBackup(record *models.BackupRecord) (*domain.Backup, error) {
	metadata, err := unmarshalJSONMap(record.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal backup metadata: %w", err)
	}
	return &domain.Backup{
		ID:               record.ID,
		DatabaseSourceID: record.DatabaseSourceID,
		BackupConfigID:   record.BackupConfigID,
		StorageID:        record.StorageID,
		Status:           domain.BackupStatus(record.Status),
		BackupMethod:     domain.BackupMethod(record.BackupMethod),
		FileName:         record.FileName,
		FilePath:         record.FilePath,
		FileSizeBytes:    record.FileSizeBytes,
		ChecksumSHA256:   record.ChecksumSHA256,
		StartedAt:        record.StartedAt,
		CompletedAt:      record.CompletedAt,
		DurationMs:       record.DurationMs,
		FailMessage:      record.FailMessage,
		EncryptionType:   domain.BackupEncryptionType(record.EncryptionType),
		Metadata:         metadata,
		CreatedAt:        record.CreatedAt,
	}, nil
}

func toDomainRestore(record *models.RestoreRecord) (*domain.Restore, error) {
	metadata, err := unmarshalJSONMap(record.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal restore metadata: %w", err)
	}
	return &domain.Restore{
		ID:                      record.ID,
		BackupID:                record.BackupID,
		DatabaseSourceID:        record.DatabaseSourceID,
		Status:                  domain.RestoreStatus(record.Status),
		TargetHost:              record.TargetHost,
		TargetPort:              record.TargetPort,
		TargetUsername:          record.TargetUsername,
		TargetPasswordEncrypted: record.TargetPasswordEncrypted,
		TargetDatabaseName:      record.TargetDatabaseName,
		TargetAuthDatabase:      record.TargetAuthDatabase,
		TargetURIEncrypted:      record.TargetURIEncrypted,
		StartedAt:               record.StartedAt,
		CompletedAt:             record.CompletedAt,
		DurationMs:              record.DurationMs,
		FailMessage:             record.FailMessage,
		Metadata:                metadata,
		CreatedAt:               record.CreatedAt,
	}, nil
}
