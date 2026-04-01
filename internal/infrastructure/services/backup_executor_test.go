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

type fakeStrategyResolver struct {
	strategy appservices.BackupStrategy
	err      error
}

func (r *fakeStrategyResolver) Resolve(_ domain.DatabaseType, _ domain.BackupMethod) (appservices.BackupStrategy, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.strategy, nil
}

type fakeStorageResolver struct {
	driver appservices.StorageDriver
	err    error
}

func (r *fakeStorageResolver) Resolve(_ domain.StorageType) (appservices.StorageDriver, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.driver, nil
}

type fakeStrategy struct {
	artifact *appservices.BackupArtifact
	err      error
}

func (s *fakeStrategy) Execute(_ context.Context, _ *domain.DatabaseSource, _ *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.artifact, nil
}

type fakeStorageDriver struct {
	storedPath string
	size       int64
	err        error
}

func (d *fakeStorageDriver) StoreFile(_ context.Context, _ *domain.Storage, _ string, localPath string) (*appservices.StoredObject, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &appservices.StoredObject{Path: d.storedPath + ":" + filepath.Base(localPath), Size: d.size}, nil
}
func (d *fakeStorageDriver) LoadFile(_ context.Context, _ *domain.Storage, path string) (*appservices.LocalObject, error) {
	return &appservices.LocalObject{Path: path, Cleanup: func() error { return nil }}, nil
}

func TestBackupExecutorExecuteBackupSuccess(t *testing.T) {
	db := openTestDB(t)
	sourceRepo := persistrepo.NewDatabaseSourceRepository(db)
	storageRepo := persistrepo.NewStorageRepository(db)
	configRepo := persistrepo.NewBackupConfigRepository(db)
	backupRepo := persistrepo.NewBackupRepository(db)

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
		Status:           domain.BackupStatusPending,
		BackupMethod:     domain.BackupMethodLogicalDump,
		EncryptionType:   domain.BackupEncryptionNone,
	})
	artifactPath := filepath.Join(t.TempDir(), "artifact.sql.gz")
	if err := os.WriteFile(artifactPath, []byte("backup-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	executor := NewBackupExecutor(
		backupRepo,
		sourceRepo,
		configRepo,
		storageRepo,
		nil,
		&fakeStrategyResolver{strategy: &fakeStrategy{
			artifact: &appservices.BackupArtifact{
				FileName:  "app.sql.gz",
				LocalPath: artifactPath,
				Metadata:  map[string]any{"tool": "mysqldump"},
			},
		}},
		&fakeStorageResolver{driver: &fakeStorageDriver{storedPath: "/stored", size: 11}},
	)

	if err := executor.ExecuteBackup(ctx, backup.ID); err != nil {
		t.Fatalf("ExecuteBackup() error = %v", err)
	}

	saved, err := backupRepo.GetByID(ctx, backup.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.BackupStatusCompleted {
		t.Fatalf("status = %s, want completed", saved.Status)
	}
	if saved.FileName != "app.sql.gz" {
		t.Fatalf("file name = %s", saved.FileName)
	}
	if saved.ChecksumSHA256 == "" {
		t.Fatal("expected checksum")
	}
	if saved.FilePath == "" {
		t.Fatal("expected file path")
	}
	if saved.Metadata["tool"] != "mysqldump" {
		t.Fatalf("metadata tool = %v", saved.Metadata["tool"])
	}
}

func TestBackupExecutorExecuteBackupFailure(t *testing.T) {
	db := openTestDB(t)
	sourceRepo := persistrepo.NewDatabaseSourceRepository(db)
	storageRepo := persistrepo.NewStorageRepository(db)
	configRepo := persistrepo.NewBackupConfigRepository(db)
	backupRepo := persistrepo.NewBackupRepository(db)

	ctx := context.Background()
	source, _ := sourceRepo.Create(ctx, domain.CreateDatabaseSourceInput{ID: "src_1", Name: "mysql", DBType: domain.DatabaseTypeMySQL, Host: "127.0.0.1", Port: 3306, Username: "user", PasswordEncrypted: "secret", DatabaseName: "app"})
	storage, _ := storageRepo.Create(ctx, domain.CreateStorageInput{ID: "stg_1", Name: "local", Type: domain.StorageTypeLocal, Config: map[string]any{"base_path": t.TempDir()}})
	config, _ := configRepo.Create(ctx, domain.CreateBackupConfigInput{ID: "cfg_1", DatabaseSourceID: source.ID, StorageID: storage.ID, IsEnabled: true, ScheduleType: domain.BackupScheduleManualOnly, RetentionType: domain.BackupRetentionNone, EncryptionType: domain.BackupEncryptionNone, CompressionType: domain.BackupCompressionGzip, BackupMethod: domain.BackupMethodLogicalDump})
	backup, _ := backupRepo.Create(ctx, domain.CreateBackupInput{ID: "bkp_1", DatabaseSourceID: source.ID, BackupConfigID: config.ID, StorageID: storage.ID, Status: domain.BackupStatusPending, BackupMethod: domain.BackupMethodLogicalDump, EncryptionType: domain.BackupEncryptionNone})

	executor := NewBackupExecutor(
		backupRepo,
		sourceRepo,
		configRepo,
		storageRepo,
		nil,
		&fakeStrategyResolver{err: errors.New("boom")},
		&fakeStorageResolver{},
	)

	if err := executor.ExecuteBackup(ctx, backup.ID); err == nil {
		t.Fatal("expected error")
	}

	saved, _ := backupRepo.GetByID(ctx, backup.ID)
	if saved.Status != domain.BackupStatusFailed {
		t.Fatalf("status = %s, want failed", saved.Status)
	}
	if saved.FailMessage == "" {
		t.Fatal("expected fail message")
	}
}

func openTestDB(t *testing.T) *gorm.DB {
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
