package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
	infradb "tango/internal/infrastructure/db"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"

	"gorm.io/gorm"
)

type fakeRestoreStrategyResolver struct {
	strategy appservices.RestoreStrategy
	err      error
}

func (r *fakeRestoreStrategyResolver) Resolve(_ domain.DatabaseType, _ domain.BackupMethod) (appservices.RestoreStrategy, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.strategy, nil
}

type fakeRestoreStrategy struct{ err error }

func (s *fakeRestoreStrategy) Execute(_ context.Context, _ *domain.DatabaseSource, _ *domain.Backup, _ string) error {
	return s.err
}

type fakeLoadStorageDriver struct {
	path string
	err  error
}

func (d *fakeLoadStorageDriver) StoreFile(_ context.Context, _ *domain.Storage, _ string, _ string) (*appservices.StoredObject, error) {
	return nil, errors.New("not implemented")
}
func (d *fakeLoadStorageDriver) LoadFile(_ context.Context, _ *domain.Storage, _ string) (*appservices.LocalObject, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &appservices.LocalObject{Path: d.path, Cleanup: func() error { return nil }}, nil
}

type fakeLoadStorageResolver struct {
	driver appservices.StorageDriver
	err    error
}

func (r *fakeLoadStorageResolver) Resolve(_ domain.StorageType) (appservices.StorageDriver, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.driver, nil
}

func TestRestoreExecutorExecuteRestoreSuccess(t *testing.T) {
	db := openRestoreTestDB(t)
	sourceRepo := persistrepo.NewDatabaseSourceRepository(db)
	storageRepo := persistrepo.NewStorageRepository(db)
	configRepo := persistrepo.NewBackupConfigRepository(db)
	backupRepo := persistrepo.NewBackupRepository(db)
	restoreRepo := persistrepo.NewRestoreRepository(db)
	ctx := context.Background()

	source, _ := sourceRepo.Create(ctx, domain.CreateDatabaseSourceInput{
		ID:                "src_1",
		Name:              "mysql",
		DBType:            domain.DatabaseTypeMySQL,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app",
	})
	storage, _ := storageRepo.Create(ctx, domain.CreateStorageInput{
		ID:     "stg_1",
		Name:   "local",
		Type:   domain.StorageTypeLocal,
		Config: map[string]any{"base_path": t.TempDir()},
	})
	config, _ := configRepo.Create(ctx, domain.CreateBackupConfigInput{
		ID:               "cfg_1",
		DatabaseSourceID: source.ID,
		StorageID:        storage.ID,
		IsEnabled:        true,
		ScheduleType:     domain.BackupScheduleManualOnly,
		RetentionType:    domain.BackupRetentionNone,
		EncryptionType:   domain.BackupEncryptionNone,
		CompressionType:  domain.BackupCompressionGzip,
		BackupMethod:     domain.BackupMethodLogicalDump,
	})
	backup, _ := backupRepo.Create(ctx, domain.CreateBackupInput{
		ID:               "bkp_1",
		DatabaseSourceID: source.ID,
		BackupConfigID:   config.ID,
		StorageID:        storage.ID,
		Status:           domain.BackupStatusCompleted,
		BackupMethod:     domain.BackupMethodLogicalDump,
		EncryptionType:   domain.BackupEncryptionNone,
		FilePath:         "/stored/app.sql.gz",
	})
	restore, _ := restoreRepo.Create(ctx, domain.CreateRestoreInput{
		ID:               "rst_1",
		BackupID:         backup.ID,
		DatabaseSourceID: source.ID,
		Status:           domain.RestoreStatusPending,
	})
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := os.WriteFile(localPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}

	executor := NewRestoreExecutor(
		restoreRepo,
		backupRepo,
		sourceRepo,
		storageRepo,
		nil,
		&fakeRestoreStrategyResolver{strategy: &fakeRestoreStrategy{}},
		&fakeLoadStorageResolver{driver: &fakeLoadStorageDriver{path: localPath}},
	)
	if err := executor.ExecuteRestore(ctx, restore.ID); err != nil {
		t.Fatalf("ExecuteRestore() error = %v", err)
	}
	saved, err := restoreRepo.GetByID(ctx, restore.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.RestoreStatusCompleted {
		t.Fatalf("status = %s, want completed", saved.Status)
	}
}

func TestRestoreExecutorExecuteRestoreFailure(t *testing.T) {
	db := openRestoreTestDB(t)
	sourceRepo := persistrepo.NewDatabaseSourceRepository(db)
	storageRepo := persistrepo.NewStorageRepository(db)
	configRepo := persistrepo.NewBackupConfigRepository(db)
	backupRepo := persistrepo.NewBackupRepository(db)
	restoreRepo := persistrepo.NewRestoreRepository(db)
	ctx := context.Background()

	source, _ := sourceRepo.Create(ctx, domain.CreateDatabaseSourceInput{
		ID:                "src_1",
		Name:              "mysql",
		DBType:            domain.DatabaseTypeMySQL,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app",
	})
	storage, _ := storageRepo.Create(ctx, domain.CreateStorageInput{
		ID:     "stg_1",
		Name:   "local",
		Type:   domain.StorageTypeLocal,
		Config: map[string]any{"base_path": t.TempDir()},
	})
	config, _ := configRepo.Create(ctx, domain.CreateBackupConfigInput{
		ID:               "cfg_1",
		DatabaseSourceID: source.ID,
		StorageID:        storage.ID,
		IsEnabled:        true,
		ScheduleType:     domain.BackupScheduleManualOnly,
		RetentionType:    domain.BackupRetentionNone,
		EncryptionType:   domain.BackupEncryptionNone,
		CompressionType:  domain.BackupCompressionGzip,
		BackupMethod:     domain.BackupMethodLogicalDump,
	})
	backup, _ := backupRepo.Create(ctx, domain.CreateBackupInput{
		ID:               "bkp_1",
		DatabaseSourceID: source.ID,
		BackupConfigID:   config.ID,
		StorageID:        storage.ID,
		Status:           domain.BackupStatusCompleted,
		BackupMethod:     domain.BackupMethodLogicalDump,
		EncryptionType:   domain.BackupEncryptionNone,
		FilePath:         "/stored/app.sql.gz",
	})
	restore, _ := restoreRepo.Create(ctx, domain.CreateRestoreInput{
		ID:               "rst_1",
		BackupID:         backup.ID,
		DatabaseSourceID: source.ID,
		Status:           domain.RestoreStatusPending,
	})
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := os.WriteFile(localPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}

	executor := NewRestoreExecutor(
		restoreRepo,
		backupRepo,
		sourceRepo,
		storageRepo,
		nil,
		&fakeRestoreStrategyResolver{strategy: &fakeRestoreStrategy{err: errors.New("restore boom")}},
		&fakeLoadStorageResolver{driver: &fakeLoadStorageDriver{path: localPath}},
	)
	if err := executor.ExecuteRestore(ctx, restore.ID); err == nil {
		t.Fatal("expected error")
	}
	saved, _ := restoreRepo.GetByID(ctx, restore.ID)
	if saved.Status != domain.RestoreStatusFailed {
		t.Fatalf("status = %s, want failed", saved.Status)
	}
}

func openRestoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := infradb.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = infradb.Close(db) })
	if err := infradb.Migrate(context.Background(), db, models.All()...); err != nil {
		t.Fatal(err)
	}
	return db
}
