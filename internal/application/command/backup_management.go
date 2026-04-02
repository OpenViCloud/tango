package command

import (
	"context"
	"log/slog"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
	"tango/internal/infrastructure/tools"
)

type CreateDatabaseSourceCommand struct {
	ID            string
	Name          string
	DBType        string
	Host          string
	Port          int
	Username      string
	Password      string
	DatabaseName  string
	Version       string
	IsTLSEnabled  bool
	AuthDatabase  string
	ConnectionURI string
	ResourceID    string
}

type UpdateDatabaseSourceCommand = CreateDatabaseSourceCommand

type CreateDatabaseSourceHandler struct {
	repo   domain.DatabaseSourceRepository
	cipher appservices.SecretCipher
}

type UpdateDatabaseSourceHandler struct {
	repo   domain.DatabaseSourceRepository
	cipher appservices.SecretCipher
}

type DeleteDatabaseSourceCommand struct {
	ID string
}

type DeleteDatabaseSourceHandler struct {
	repo domain.DatabaseSourceRepository
}

func NewCreateDatabaseSourceHandler(repo domain.DatabaseSourceRepository, cipher appservices.SecretCipher) *CreateDatabaseSourceHandler {
	return &CreateDatabaseSourceHandler{repo: repo, cipher: cipher}
}

func NewUpdateDatabaseSourceHandler(repo domain.DatabaseSourceRepository, cipher appservices.SecretCipher) *UpdateDatabaseSourceHandler {
	return &UpdateDatabaseSourceHandler{repo: repo, cipher: cipher}
}

func NewDeleteDatabaseSourceHandler(repo domain.DatabaseSourceRepository) *DeleteDatabaseSourceHandler {
	return &DeleteDatabaseSourceHandler{repo: repo}
}

func (h *CreateDatabaseSourceHandler) Handle(ctx context.Context, cmd CreateDatabaseSourceCommand) (*domain.DatabaseSource, error) {
	input, err := buildDatabaseSourceInput(ctx, h.cipher, cmd)
	if err != nil {
		return nil, err
	}
	input.ID = cmd.ID
	return h.repo.Create(ctx, input)
}

func (h *UpdateDatabaseSourceHandler) Handle(ctx context.Context, cmd UpdateDatabaseSourceCommand) (*domain.DatabaseSource, error) {
	input, err := buildDatabaseSourceUpdateInput(ctx, h.cipher, cmd)
	if err != nil {
		return nil, err
	}
	return h.repo.Update(ctx, cmd.ID, input)
}

func (h *DeleteDatabaseSourceHandler) Handle(ctx context.Context, cmd DeleteDatabaseSourceCommand) error {
	return h.repo.Delete(ctx, strings.TrimSpace(cmd.ID))
}

func buildDatabaseSourceInput(ctx context.Context, cipher appservices.SecretCipher, cmd CreateDatabaseSourceCommand) (domain.CreateDatabaseSourceInput, error) {
	dbType, err := domain.ValidateDatabaseType(cmd.DBType)
	if err != nil {
		return domain.CreateDatabaseSourceInput{}, domain.ErrInvalidInput
	}
	passwordEncrypted, connectionURIEncrypted, err := encryptConnectionSecrets(ctx, cipher, cmd.Password, cmd.ConnectionURI)
	if err != nil {
		return domain.CreateDatabaseSourceInput{}, err
	}
	version := strings.TrimSpace(cmd.Version)
	if dbType == domain.DatabaseTypeMySQL {
		detectedVersion, err := tools.DetectMySQLVersion(ctx, tools.MySQLConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if err != nil {
			slog.Default().Warn("mysql version detection failed on create source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", err,
			)
		} else {
			version = detectedVersion
		}
	}
	if dbType == domain.DatabaseTypeMariaDB {
		detectedVersion, err := tools.DetectMariaDBVersion(ctx, tools.MariaDBConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if err != nil {
			slog.Default().Warn("mariadb version detection failed on create source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", err,
			)
		} else {
			version = detectedVersion
		}
	}
	if dbType == domain.DatabaseTypePostgres {
		detectedVersion, err := tools.DetectPostgresVersion(ctx, tools.PostgresConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if err != nil {
			slog.Default().Warn("postgres version detection failed on create source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", err,
			)
		} else {
			version = detectedVersion
		}
	}
	return domain.CreateDatabaseSourceInput{
		Name:                   strings.TrimSpace(cmd.Name),
		DBType:                 dbType,
		Host:                   strings.TrimSpace(cmd.Host),
		Port:                   cmd.Port,
		Username:               strings.TrimSpace(cmd.Username),
		PasswordEncrypted:      passwordEncrypted,
		DatabaseName:           strings.TrimSpace(cmd.DatabaseName),
		Version:                version,
		IsTLSEnabled:           cmd.IsTLSEnabled,
		AuthDatabase:           strings.TrimSpace(cmd.AuthDatabase),
		ConnectionURIEncrypted: connectionURIEncrypted,
		ResourceID:             strings.TrimSpace(cmd.ResourceID),
	}, nil
}

func buildDatabaseSourceUpdateInput(ctx context.Context, cipher appservices.SecretCipher, cmd UpdateDatabaseSourceCommand) (domain.UpdateDatabaseSourceInput, error) {
	passwordEncrypted, connectionURIEncrypted, err := encryptConnectionSecrets(ctx, cipher, cmd.Password, cmd.ConnectionURI)
	if err != nil {
		return domain.UpdateDatabaseSourceInput{}, err
	}
	version := strings.TrimSpace(cmd.Version)
	if dbType, err := domain.ValidateDatabaseType(cmd.DBType); err == nil && dbType == domain.DatabaseTypeMySQL {
		detectedVersion, detectErr := tools.DetectMySQLVersion(ctx, tools.MySQLConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if detectErr != nil {
			slog.Default().Warn("mysql version detection failed on update source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", detectErr,
			)
		} else {
			version = detectedVersion
		}
	}
	if dbType, err := domain.ValidateDatabaseType(cmd.DBType); err == nil && dbType == domain.DatabaseTypeMariaDB {
		detectedVersion, detectErr := tools.DetectMariaDBVersion(ctx, tools.MariaDBConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if detectErr != nil {
			slog.Default().Warn("mariadb version detection failed on update source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", detectErr,
			)
		} else {
			version = detectedVersion
		}
	}
	if dbType, err := domain.ValidateDatabaseType(cmd.DBType); err == nil && dbType == domain.DatabaseTypePostgres {
		detectedVersion, detectErr := tools.DetectPostgresVersion(ctx, tools.PostgresConnectionConfig{
			Host:     strings.TrimSpace(cmd.Host),
			Port:     cmd.Port,
			Username: strings.TrimSpace(cmd.Username),
			Password: cmd.Password,
			Database: strings.TrimSpace(cmd.DatabaseName),
		})
		if detectErr != nil {
			slog.Default().Warn("postgres version detection failed on update source; keeping provided version",
				"host", strings.TrimSpace(cmd.Host),
				"port", cmd.Port,
				"database", strings.TrimSpace(cmd.DatabaseName),
				"err", detectErr,
			)
		} else {
			version = detectedVersion
		}
	}
	return domain.UpdateDatabaseSourceInput{
		Name:                   strings.TrimSpace(cmd.Name),
		Host:                   strings.TrimSpace(cmd.Host),
		Port:                   cmd.Port,
		Username:               strings.TrimSpace(cmd.Username),
		PasswordEncrypted:      passwordEncrypted,
		DatabaseName:           strings.TrimSpace(cmd.DatabaseName),
		Version:                version,
		IsTLSEnabled:           cmd.IsTLSEnabled,
		AuthDatabase:           strings.TrimSpace(cmd.AuthDatabase),
		ConnectionURIEncrypted: connectionURIEncrypted,
		ResourceID:             strings.TrimSpace(cmd.ResourceID),
	}, nil
}

type CreateStorageCommand struct {
	ID          string
	Name        string
	Type        string
	Config      map[string]any
	Credentials map[string]any
}

type UpdateStorageCommand = CreateStorageCommand

type CreateStorageHandler struct {
	repo   domain.StorageRepository
	cipher appservices.SecretCipher
}

type UpdateStorageHandler struct {
	repo   domain.StorageRepository
	cipher appservices.SecretCipher
}

type DeleteStorageCommand struct {
	ID string
}

type DeleteStorageHandler struct {
	repo domain.StorageRepository
}

func NewCreateStorageHandler(repo domain.StorageRepository, cipher appservices.SecretCipher) *CreateStorageHandler {
	return &CreateStorageHandler{repo: repo, cipher: cipher}
}

func NewUpdateStorageHandler(repo domain.StorageRepository, cipher appservices.SecretCipher) *UpdateStorageHandler {
	return &UpdateStorageHandler{repo: repo, cipher: cipher}
}

func NewDeleteStorageHandler(repo domain.StorageRepository) *DeleteStorageHandler {
	return &DeleteStorageHandler{repo: repo}
}

func (h *CreateStorageHandler) Handle(ctx context.Context, cmd CreateStorageCommand) (*domain.Storage, error) {
	input, err := buildStorageInput(ctx, h.cipher, cmd)
	if err != nil {
		return nil, err
	}
	input.ID = cmd.ID
	return h.repo.Create(ctx, input)
}

func (h *UpdateStorageHandler) Handle(ctx context.Context, cmd UpdateStorageCommand) (*domain.Storage, error) {
	input, err := buildStorageUpdateInput(ctx, h.cipher, cmd)
	if err != nil {
		return nil, err
	}
	return h.repo.Update(ctx, cmd.ID, input)
}

func (h *DeleteStorageHandler) Handle(ctx context.Context, cmd DeleteStorageCommand) error {
	return h.repo.Delete(ctx, strings.TrimSpace(cmd.ID))
}

func buildStorageInput(ctx context.Context, cipher appservices.SecretCipher, cmd CreateStorageCommand) (domain.CreateStorageInput, error) {
	storageType, err := domain.ValidateStorageType(cmd.Type)
	if err != nil {
		return domain.CreateStorageInput{}, domain.ErrInvalidInput
	}
	credentialsEncrypted, err := encryptJSONMap(ctx, cipher, cmd.Credentials)
	if err != nil {
		return domain.CreateStorageInput{}, err
	}
	return domain.CreateStorageInput{
		Name:                 strings.TrimSpace(cmd.Name),
		Type:                 storageType,
		Config:               cmd.Config,
		CredentialsEncrypted: credentialsEncrypted,
	}, nil
}

func buildStorageUpdateInput(ctx context.Context, cipher appservices.SecretCipher, cmd UpdateStorageCommand) (domain.UpdateStorageInput, error) {
	storageType, err := domain.ValidateStorageType(cmd.Type)
	if err != nil {
		return domain.UpdateStorageInput{}, domain.ErrInvalidInput
	}
	credentialsEncrypted, err := encryptJSONMap(ctx, cipher, cmd.Credentials)
	if err != nil {
		return domain.UpdateStorageInput{}, err
	}
	return domain.UpdateStorageInput{
		Name:                 strings.TrimSpace(cmd.Name),
		Type:                 storageType,
		Config:               cmd.Config,
		CredentialsEncrypted: credentialsEncrypted,
	}, nil
}

type CreateBackupConfigCommand struct {
	ID               string
	DatabaseSourceID string
	StorageID        string
	IsEnabled        bool
	ScheduleType     string
	TimeOfDay        string
	IntervalHours    int
	RetentionType    string
	RetentionDays    int
	RetentionCount   int
	IsRetryIfFailed  bool
	MaxRetryCount    int
	EncryptionType   string
	CompressionType  string
	BackupMethod     string
}

type UpdateBackupConfigCommand struct {
	ID              string
	StorageID       string
	IsEnabled       bool
	ScheduleType    string
	TimeOfDay       string
	IntervalHours   int
	RetentionType   string
	RetentionDays   int
	RetentionCount  int
	IsRetryIfFailed bool
	MaxRetryCount   int
	EncryptionType  string
	CompressionType string
	BackupMethod    string
}

type CreateBackupConfigHandler struct {
	repo        domain.BackupConfigRepository
	sourceRepo  domain.DatabaseSourceRepository
	storageRepo domain.StorageRepository
}

type UpdateBackupConfigHandler struct {
	repo        domain.BackupConfigRepository
	storageRepo domain.StorageRepository
}

func NewCreateBackupConfigHandler(repo domain.BackupConfigRepository, sourceRepo domain.DatabaseSourceRepository, storageRepo domain.StorageRepository) *CreateBackupConfigHandler {
	return &CreateBackupConfigHandler{repo: repo, sourceRepo: sourceRepo, storageRepo: storageRepo}
}

func NewUpdateBackupConfigHandler(repo domain.BackupConfigRepository, storageRepo domain.StorageRepository) *UpdateBackupConfigHandler {
	return &UpdateBackupConfigHandler{repo: repo, storageRepo: storageRepo}
}

func (h *CreateBackupConfigHandler) Handle(ctx context.Context, cmd CreateBackupConfigCommand) (*domain.BackupConfig, error) {
	if _, err := h.sourceRepo.GetByID(ctx, cmd.DatabaseSourceID); err != nil {
		return nil, err
	}
	if _, err := h.storageRepo.GetByID(ctx, cmd.StorageID); err != nil {
		return nil, err
	}
	input, err := buildBackupConfigCreateInput(cmd)
	if err != nil {
		return nil, err
	}
	input.ID = cmd.ID
	return h.repo.Create(ctx, input)
}

func (h *UpdateBackupConfigHandler) Handle(ctx context.Context, cmd UpdateBackupConfigCommand) (*domain.BackupConfig, error) {
	if _, err := h.storageRepo.GetByID(ctx, cmd.StorageID); err != nil {
		return nil, err
	}
	input, err := buildBackupConfigUpdateInput(cmd)
	if err != nil {
		return nil, err
	}
	return h.repo.Update(ctx, cmd.ID, input)
}

func buildBackupConfigCreateInput(cmd CreateBackupConfigCommand) (domain.CreateBackupConfigInput, error) {
	scheduleType, err := domain.ValidateBackupScheduleType(cmd.ScheduleType)
	if err != nil {
		return domain.CreateBackupConfigInput{}, domain.ErrInvalidInput
	}
	retentionType, err := domain.ValidateBackupRetentionType(cmd.RetentionType)
	if err != nil {
		return domain.CreateBackupConfigInput{}, domain.ErrInvalidInput
	}
	encryptionType, err := domain.ValidateBackupEncryptionType(cmd.EncryptionType)
	if err != nil {
		return domain.CreateBackupConfigInput{}, domain.ErrInvalidInput
	}
	compressionType, err := domain.ValidateBackupCompressionType(cmd.CompressionType)
	if err != nil {
		return domain.CreateBackupConfigInput{}, domain.ErrInvalidInput
	}
	backupMethod, err := domain.ValidateBackupMethod(cmd.BackupMethod)
	if err != nil {
		return domain.CreateBackupConfigInput{}, domain.ErrInvalidInput
	}
	return domain.CreateBackupConfigInput{
		DatabaseSourceID: strings.TrimSpace(cmd.DatabaseSourceID),
		StorageID:        strings.TrimSpace(cmd.StorageID),
		IsEnabled:        cmd.IsEnabled,
		ScheduleType:     scheduleType,
		TimeOfDay:        strings.TrimSpace(cmd.TimeOfDay),
		IntervalHours:    cmd.IntervalHours,
		RetentionType:    retentionType,
		RetentionDays:    cmd.RetentionDays,
		RetentionCount:   cmd.RetentionCount,
		IsRetryIfFailed:  cmd.IsRetryIfFailed,
		MaxRetryCount:    cmd.MaxRetryCount,
		EncryptionType:   encryptionType,
		CompressionType:  compressionType,
		BackupMethod:     backupMethod,
	}, nil
}

func buildBackupConfigUpdateInput(cmd UpdateBackupConfigCommand) (domain.UpdateBackupConfigInput, error) {
	scheduleType, err := domain.ValidateBackupScheduleType(cmd.ScheduleType)
	if err != nil {
		return domain.UpdateBackupConfigInput{}, domain.ErrInvalidInput
	}
	retentionType, err := domain.ValidateBackupRetentionType(cmd.RetentionType)
	if err != nil {
		return domain.UpdateBackupConfigInput{}, domain.ErrInvalidInput
	}
	encryptionType, err := domain.ValidateBackupEncryptionType(cmd.EncryptionType)
	if err != nil {
		return domain.UpdateBackupConfigInput{}, domain.ErrInvalidInput
	}
	compressionType, err := domain.ValidateBackupCompressionType(cmd.CompressionType)
	if err != nil {
		return domain.UpdateBackupConfigInput{}, domain.ErrInvalidInput
	}
	backupMethod, err := domain.ValidateBackupMethod(cmd.BackupMethod)
	if err != nil {
		return domain.UpdateBackupConfigInput{}, domain.ErrInvalidInput
	}
	return domain.UpdateBackupConfigInput{
		StorageID:       strings.TrimSpace(cmd.StorageID),
		IsEnabled:       cmd.IsEnabled,
		ScheduleType:    scheduleType,
		TimeOfDay:       strings.TrimSpace(cmd.TimeOfDay),
		IntervalHours:   cmd.IntervalHours,
		RetentionType:   retentionType,
		RetentionDays:   cmd.RetentionDays,
		RetentionCount:  cmd.RetentionCount,
		IsRetryIfFailed: cmd.IsRetryIfFailed,
		MaxRetryCount:   cmd.MaxRetryCount,
		EncryptionType:  encryptionType,
		CompressionType: compressionType,
		BackupMethod:    backupMethod,
	}, nil
}

type TriggerBackupCommand struct {
	ID               string
	DatabaseSourceID string
	BackupConfigID   string
	StorageID        string
	Metadata         map[string]any
}

type TriggerBackupHandler struct {
	sourceRepo  domain.DatabaseSourceRepository
	configRepo  domain.BackupConfigRepository
	storageRepo domain.StorageRepository
	backupRepo  domain.BackupRepository
	executor    appservices.BackupExecutor
}

func NewTriggerBackupHandler(sourceRepo domain.DatabaseSourceRepository, configRepo domain.BackupConfigRepository, storageRepo domain.StorageRepository, backupRepo domain.BackupRepository, executor appservices.BackupExecutor) *TriggerBackupHandler {
	return &TriggerBackupHandler{sourceRepo: sourceRepo, configRepo: configRepo, storageRepo: storageRepo, backupRepo: backupRepo, executor: executor}
}

func (h *TriggerBackupHandler) Handle(ctx context.Context, cmd TriggerBackupCommand) (*domain.Backup, error) {
	source, err := h.sourceRepo.GetByID(ctx, cmd.DatabaseSourceID)
	if err != nil {
		return nil, err
	}
	config, err := h.configRepo.GetByDatabaseSourceID(ctx, source.ID)
	if err != nil {
		return nil, err
	}
	storageID := strings.TrimSpace(cmd.StorageID)
	if storageID == "" {
		storageID = config.StorageID
	}
	if _, err := h.storageRepo.GetByID(ctx, storageID); err != nil {
		return nil, err
	}
	backup, err := h.backupRepo.Create(ctx, domain.CreateBackupInput{
		ID:               cmd.ID,
		DatabaseSourceID: source.ID,
		BackupConfigID:   config.ID,
		StorageID:        storageID,
		Status:           domain.BackupStatusPending,
		BackupMethod:     config.BackupMethod,
		EncryptionType:   config.EncryptionType,
		Metadata:         cmd.Metadata,
	})
	if err != nil {
		return nil, err
	}
	if h.executor != nil {
		go func(backupID string) {
			_ = h.executor.ExecuteBackup(context.Background(), backupID)
		}(backup.ID)
	}
	return backup, nil
}

type TriggerRestoreCommand struct {
	ID                  string
	BackupID            string
	DatabaseSourceID    string
	TargetHost          string
	TargetPort          int
	TargetUsername      string
	TargetPassword      string
	TargetDatabase      string
	TargetAuthDatabase  string
	TargetConnectionURI string
	Metadata            map[string]any
}

type TriggerRestoreHandler struct {
	backupRepo  domain.BackupRepository
	restoreRepo domain.RestoreRepository
	cipher      appservices.SecretCipher
	executor    appservices.RestoreExecutor
}

func NewTriggerRestoreHandler(backupRepo domain.BackupRepository, restoreRepo domain.RestoreRepository, cipher appservices.SecretCipher, executor appservices.RestoreExecutor) *TriggerRestoreHandler {
	return &TriggerRestoreHandler{backupRepo: backupRepo, restoreRepo: restoreRepo, cipher: cipher, executor: executor}
}

func (h *TriggerRestoreHandler) Handle(ctx context.Context, cmd TriggerRestoreCommand) (*domain.Restore, error) {
	if _, err := h.backupRepo.GetByID(ctx, cmd.BackupID); err != nil {
		return nil, err
	}
	passwordEncrypted, targetURIEncrypted, err := encryptConnectionSecrets(ctx, h.cipher, cmd.TargetPassword, cmd.TargetConnectionURI)
	if err != nil {
		return nil, err
	}
	restore, err := h.restoreRepo.Create(ctx, domain.CreateRestoreInput{
		ID:                      cmd.ID,
		BackupID:                cmd.BackupID,
		DatabaseSourceID:        strings.TrimSpace(cmd.DatabaseSourceID),
		Status:                  domain.RestoreStatusPending,
		TargetHost:              strings.TrimSpace(cmd.TargetHost),
		TargetPort:              cmd.TargetPort,
		TargetUsername:          strings.TrimSpace(cmd.TargetUsername),
		TargetPasswordEncrypted: passwordEncrypted,
		TargetDatabaseName:      strings.TrimSpace(cmd.TargetDatabase),
		TargetAuthDatabase:      strings.TrimSpace(cmd.TargetAuthDatabase),
		TargetURIEncrypted:      targetURIEncrypted,
		Metadata:                cmd.Metadata,
	})
	if err != nil {
		return nil, err
	}
	if h.executor != nil {
		go func(restoreID string) {
			_ = h.executor.ExecuteRestore(context.Background(), restoreID)
		}(restore.ID)
	}
	return restore, nil
}
