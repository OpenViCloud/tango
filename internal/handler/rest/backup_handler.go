package rest

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"
)

type BackupHandler struct {
	createDatabaseSource        *command.CreateDatabaseSourceHandler
	updateDatabaseSource        *command.UpdateDatabaseSourceHandler
	deleteDatabaseSource        *command.DeleteDatabaseSourceHandler
	listDatabaseSources         *query.ListDatabaseSourcesHandler
	getDatabaseSource           *query.GetDatabaseSourceHandler
	createStorage               *command.CreateStorageHandler
	updateStorage               *command.UpdateStorageHandler
	deleteStorage               *command.DeleteStorageHandler
	listStorages                *query.ListStoragesHandler
	getStorage                  *query.GetStorageHandler
	createBackupConfig          *command.CreateBackupConfigHandler
	updateBackupConfig          *command.UpdateBackupConfigHandler
	getBackupConfig             *query.GetBackupConfigHandler
	getBackupConfigBySource     *query.GetBackupConfigByDatabaseSourceHandler
	triggerBackup               *command.TriggerBackupHandler
	listBackupsByDatabaseSource *query.ListBackupsByDatabaseSourceHandler
	getBackup                   *query.GetBackupHandler
	triggerRestore              *command.TriggerRestoreHandler
	getRestore                  *query.GetRestoreHandler
}

func NewBackupHandler(
	createDatabaseSource *command.CreateDatabaseSourceHandler,
	updateDatabaseSource *command.UpdateDatabaseSourceHandler,
	deleteDatabaseSource *command.DeleteDatabaseSourceHandler,
	listDatabaseSources *query.ListDatabaseSourcesHandler,
	getDatabaseSource *query.GetDatabaseSourceHandler,
	createStorage *command.CreateStorageHandler,
	updateStorage *command.UpdateStorageHandler,
	deleteStorage *command.DeleteStorageHandler,
	listStorages *query.ListStoragesHandler,
	getStorage *query.GetStorageHandler,
	createBackupConfig *command.CreateBackupConfigHandler,
	updateBackupConfig *command.UpdateBackupConfigHandler,
	getBackupConfig *query.GetBackupConfigHandler,
	getBackupConfigBySource *query.GetBackupConfigByDatabaseSourceHandler,
	triggerBackup *command.TriggerBackupHandler,
	listBackupsByDatabaseSource *query.ListBackupsByDatabaseSourceHandler,
	getBackup *query.GetBackupHandler,
	triggerRestore *command.TriggerRestoreHandler,
	getRestore *query.GetRestoreHandler,
) *BackupHandler {
	return &BackupHandler{
		createDatabaseSource:        createDatabaseSource,
		updateDatabaseSource:        updateDatabaseSource,
		deleteDatabaseSource:        deleteDatabaseSource,
		listDatabaseSources:         listDatabaseSources,
		getDatabaseSource:           getDatabaseSource,
		createStorage:               createStorage,
		updateStorage:               updateStorage,
		deleteStorage:               deleteStorage,
		listStorages:                listStorages,
		getStorage:                  getStorage,
		createBackupConfig:          createBackupConfig,
		updateBackupConfig:          updateBackupConfig,
		getBackupConfig:             getBackupConfig,
		getBackupConfigBySource:     getBackupConfigBySource,
		triggerBackup:               triggerBackup,
		listBackupsByDatabaseSource: listBackupsByDatabaseSource,
		getBackup:                   getBackup,
		triggerRestore:              triggerRestore,
		getRestore:                  getRestore,
	}
}

func (h *BackupHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/backup-sources", h.CreateDatabaseSource)
	rg.GET("/backup-sources", h.ListDatabaseSources)
	rg.GET("/backup-sources/:id", h.GetDatabaseSource)
	rg.PUT("/backup-sources/:id", h.UpdateDatabaseSource)
	rg.DELETE("/backup-sources/:id", h.DeleteDatabaseSource)

	rg.POST("/storages", h.CreateStorage)
	rg.GET("/storages", h.ListStorages)
	rg.GET("/storages/:id", h.GetStorage)
	rg.PUT("/storages/:id", h.UpdateStorage)
	rg.DELETE("/storages/:id", h.DeleteStorage)

	rg.POST("/backup-configs", h.CreateBackupConfig)
	rg.GET("/backup-configs/:id", h.GetBackupConfig)
	rg.GET("/backup-sources/:id/backup-config", h.GetBackupConfigBySource)
	rg.PUT("/backup-configs/:id", h.UpdateBackupConfig)

	rg.POST("/backup-sources/:id/backups", h.TriggerBackup)
	rg.GET("/backup-sources/:id/backups", h.ListBackupsByDatabaseSource)
	rg.GET("/backups/:id", h.GetBackup)
	rg.POST("/backups/:id/restore", h.TriggerRestore)
	rg.GET("/restores/:id", h.GetRestore)
}

type databaseSourceConnectionRequest struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Database      string `json:"database"`
	AuthDatabase  string `json:"auth_database"`
	ConnectionURI string `json:"connection_uri"`
}

type createDatabaseSourceRequest struct {
	Name         string                          `json:"name"`
	DBType       string                          `json:"db_type"`
	Version      string                          `json:"version"`
	IsTLSEnabled bool                            `json:"is_tls_enabled"`
	ResourceID   string                          `json:"resource_id"`
	Connection   databaseSourceConnectionRequest `json:"connection"`
}

type createStorageRequest struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Credentials map[string]any `json:"credentials"`
}

type createBackupConfigRequest struct {
	DatabaseSourceID string `json:"database_source_id"`
	StorageID        string `json:"storage_id"`
	IsEnabled        bool   `json:"is_enabled"`
	ScheduleType     string `json:"schedule_type"`
	TimeOfDay        string `json:"time_of_day"`
	IntervalHours    int    `json:"interval_hours"`
	RetentionType    string `json:"retention_type"`
	RetentionDays    int    `json:"retention_days"`
	RetentionCount   int    `json:"retention_count"`
	IsRetryIfFailed  bool   `json:"is_retry_if_failed"`
	MaxRetryCount    int    `json:"max_retry_count"`
	EncryptionType   string `json:"encryption_type"`
	CompressionType  string `json:"compression_type"`
	BackupMethod     string `json:"backup_method"`
}

type triggerBackupRequest struct {
	StorageID string         `json:"storage_id"`
	Metadata  map[string]any `json:"metadata"`
}

type triggerRestoreRequest struct {
	DatabaseSourceID    string         `json:"database_source_id"`
	TargetHost          string         `json:"target_host"`
	TargetPort          int            `json:"target_port"`
	TargetUsername      string         `json:"target_username"`
	TargetPassword      string         `json:"target_password"`
	TargetDatabase      string         `json:"target_database_name"`
	TargetAuthDatabase  string         `json:"target_auth_database"`
	TargetConnectionURI string         `json:"target_connection_uri"`
	Metadata            map[string]any `json:"metadata"`
}

type databaseSourceResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DBType           string `json:"db_type"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Username         string `json:"username"`
	DatabaseName     string `json:"database_name"`
	Version          string `json:"version,omitempty"`
	IsTLSEnabled     bool   `json:"is_tls_enabled"`
	AuthDatabase     string `json:"auth_database,omitempty"`
	HasConnectionURI bool   `json:"has_connection_uri"`
	ResourceID       string `json:"resource_id,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type storageResponse struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Config         map[string]any `json:"config"`
	HasCredentials bool           `json:"has_credentials"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

type backupConfigResponse struct {
	ID               string `json:"id"`
	DatabaseSourceID string `json:"database_source_id"`
	StorageID        string `json:"storage_id"`
	IsEnabled        bool   `json:"is_enabled"`
	ScheduleType     string `json:"schedule_type"`
	TimeOfDay        string `json:"time_of_day,omitempty"`
	IntervalHours    int    `json:"interval_hours,omitempty"`
	RetentionType    string `json:"retention_type"`
	RetentionDays    int    `json:"retention_days,omitempty"`
	RetentionCount   int    `json:"retention_count,omitempty"`
	IsRetryIfFailed  bool   `json:"is_retry_if_failed"`
	MaxRetryCount    int    `json:"max_retry_count"`
	EncryptionType   string `json:"encryption_type"`
	CompressionType  string `json:"compression_type"`
	BackupMethod     string `json:"backup_method"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type backupResponse struct {
	ID               string         `json:"id"`
	DatabaseSourceID string         `json:"database_source_id"`
	BackupConfigID   string         `json:"backup_config_id,omitempty"`
	StorageID        string         `json:"storage_id"`
	Status           string         `json:"status"`
	BackupMethod     string         `json:"backup_method"`
	FileName         string         `json:"file_name,omitempty"`
	FilePath         string         `json:"file_path,omitempty"`
	FileSizeBytes    int64          `json:"file_size_bytes,omitempty"`
	ChecksumSHA256   string         `json:"checksum_sha256,omitempty"`
	StartedAt        string         `json:"started_at,omitempty"`
	CompletedAt      string         `json:"completed_at,omitempty"`
	DurationMs       int64          `json:"duration_ms,omitempty"`
	FailMessage      string         `json:"fail_message,omitempty"`
	EncryptionType   string         `json:"encryption_type"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        string         `json:"created_at"`
}

type restoreResponse struct {
	ID                 string         `json:"id"`
	BackupID           string         `json:"backup_id"`
	DatabaseSourceID   string         `json:"database_source_id,omitempty"`
	Status             string         `json:"status"`
	TargetHost         string         `json:"target_host,omitempty"`
	TargetPort         int            `json:"target_port,omitempty"`
	TargetUsername     string         `json:"target_username,omitempty"`
	TargetDatabaseName string         `json:"target_database_name,omitempty"`
	TargetAuthDatabase string         `json:"target_auth_database,omitempty"`
	HasTargetURI       bool           `json:"has_target_connection_uri"`
	Metadata           map[string]any `json:"metadata"`
	StartedAt          string         `json:"started_at,omitempty"`
	CompletedAt        string         `json:"completed_at,omitempty"`
	DurationMs         int64          `json:"duration_ms,omitempty"`
	FailMessage        string         `json:"fail_message,omitempty"`
	CreatedAt          string         `json:"created_at"`
}

func (h *BackupHandler) CreateDatabaseSource(c *gin.Context) {
	var req createDatabaseSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	source, err := h.createDatabaseSource.Handle(c.Request.Context(), command.CreateDatabaseSourceCommand{
		ID:            uuid.New().String(),
		Name:          req.Name,
		DBType:        req.DBType,
		Host:          req.Connection.Host,
		Port:          req.Connection.Port,
		Username:      req.Connection.Username,
		Password:      req.Connection.Password,
		DatabaseName:  req.Connection.Database,
		Version:       req.Version,
		IsTLSEnabled:  req.IsTLSEnabled,
		AuthDatabase:  req.Connection.AuthDatabase,
		ConnectionURI: req.Connection.ConnectionURI,
		ResourceID:    req.ResourceID,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDatabaseSourceResponse(source))
}

func (h *BackupHandler) ListDatabaseSources(c *gin.Context) {
	items, err := h.listDatabaseSources.Handle(c.Request.Context(), c.Query("resource_id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	out := make([]databaseSourceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toDatabaseSourceResponse(item))
	}
	response.OK(c, gin.H{"items": out})
}

func (h *BackupHandler) GetDatabaseSource(c *gin.Context) {
	item, err := h.getDatabaseSource.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toDatabaseSourceResponse(item))
}

func (h *BackupHandler) UpdateDatabaseSource(c *gin.Context) {
	var req createDatabaseSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.updateDatabaseSource.Handle(c.Request.Context(), command.UpdateDatabaseSourceCommand{
		ID:            c.Param("id"),
		Name:          req.Name,
		DBType:        req.DBType,
		Host:          req.Connection.Host,
		Port:          req.Connection.Port,
		Username:      req.Connection.Username,
		Password:      req.Connection.Password,
		DatabaseName:  req.Connection.Database,
		Version:       req.Version,
		IsTLSEnabled:  req.IsTLSEnabled,
		AuthDatabase:  req.Connection.AuthDatabase,
		ConnectionURI: req.Connection.ConnectionURI,
		ResourceID:    req.ResourceID,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toDatabaseSourceResponse(item))
}

func (h *BackupHandler) DeleteDatabaseSource(c *gin.Context) {
	if err := h.deleteDatabaseSource.Handle(c.Request.Context(), command.DeleteDatabaseSourceCommand{
		ID: c.Param("id"),
	}); err != nil {
		writeBackupError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BackupHandler) CreateStorage(c *gin.Context) {
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.createStorage.Handle(c.Request.Context(), command.CreateStorageCommand{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Type:        req.Type,
		Config:      req.Config,
		Credentials: req.Credentials,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toStorageResponse(item))
}

func (h *BackupHandler) ListStorages(c *gin.Context) {
	items, err := h.listStorages.Handle(c.Request.Context())
	if err != nil {
		writeBackupError(c, err)
		return
	}
	out := make([]storageResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toStorageResponse(item))
	}
	response.OK(c, gin.H{"items": out})
}

func (h *BackupHandler) GetStorage(c *gin.Context) {
	item, err := h.getStorage.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toStorageResponse(item))
}

func (h *BackupHandler) DeleteStorage(c *gin.Context) {
	if err := h.deleteStorage.Handle(c.Request.Context(), command.DeleteStorageCommand{
		ID: c.Param("id"),
	}); err != nil {
		writeBackupError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BackupHandler) UpdateStorage(c *gin.Context) {
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.updateStorage.Handle(c.Request.Context(), command.UpdateStorageCommand{
		ID:          c.Param("id"),
		Name:        req.Name,
		Type:        req.Type,
		Config:      req.Config,
		Credentials: req.Credentials,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toStorageResponse(item))
}

func (h *BackupHandler) CreateBackupConfig(c *gin.Context) {
	var req createBackupConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.createBackupConfig.Handle(c.Request.Context(), command.CreateBackupConfigCommand{
		ID:               uuid.New().String(),
		DatabaseSourceID: req.DatabaseSourceID,
		StorageID:        req.StorageID,
		IsEnabled:        req.IsEnabled,
		ScheduleType:     req.ScheduleType,
		TimeOfDay:        req.TimeOfDay,
		IntervalHours:    req.IntervalHours,
		RetentionType:    req.RetentionType,
		RetentionDays:    req.RetentionDays,
		RetentionCount:   req.RetentionCount,
		IsRetryIfFailed:  req.IsRetryIfFailed,
		MaxRetryCount:    req.MaxRetryCount,
		EncryptionType:   req.EncryptionType,
		CompressionType:  req.CompressionType,
		BackupMethod:     req.BackupMethod,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toBackupConfigResponse(item))
}

func (h *BackupHandler) GetBackupConfig(c *gin.Context) {
	item, err := h.getBackupConfig.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toBackupConfigResponse(item))
}

func (h *BackupHandler) GetBackupConfigBySource(c *gin.Context) {
	item, err := h.getBackupConfigBySource.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toBackupConfigResponse(item))
}

func (h *BackupHandler) UpdateBackupConfig(c *gin.Context) {
	var req createBackupConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.updateBackupConfig.Handle(c.Request.Context(), command.UpdateBackupConfigCommand{
		ID:              c.Param("id"),
		StorageID:       req.StorageID,
		IsEnabled:       req.IsEnabled,
		ScheduleType:    req.ScheduleType,
		TimeOfDay:       req.TimeOfDay,
		IntervalHours:   req.IntervalHours,
		RetentionType:   req.RetentionType,
		RetentionDays:   req.RetentionDays,
		RetentionCount:  req.RetentionCount,
		IsRetryIfFailed: req.IsRetryIfFailed,
		MaxRetryCount:   req.MaxRetryCount,
		EncryptionType:  req.EncryptionType,
		CompressionType: req.CompressionType,
		BackupMethod:    req.BackupMethod,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toBackupConfigResponse(item))
}

func (h *BackupHandler) TriggerBackup(c *gin.Context) {
	var req triggerBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.triggerBackup.Handle(c.Request.Context(), command.TriggerBackupCommand{
		ID:               uuid.New().String(),
		DatabaseSourceID: c.Param("id"),
		StorageID:        req.StorageID,
		Metadata:         req.Metadata,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toBackupResponse(item))
}

func (h *BackupHandler) ListBackupsByDatabaseSource(c *gin.Context) {
	items, err := h.listBackupsByDatabaseSource.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	out := make([]backupResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toBackupResponse(item))
	}
	response.OK(c, gin.H{"items": out})
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	item, err := h.getBackup.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toBackupResponse(item))
}

func (h *BackupHandler) TriggerRestore(c *gin.Context) {
	var req triggerRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	item, err := h.triggerRestore.Handle(c.Request.Context(), command.TriggerRestoreCommand{
		ID:                  uuid.New().String(),
		BackupID:            c.Param("id"),
		DatabaseSourceID:    req.DatabaseSourceID,
		TargetHost:          req.TargetHost,
		TargetPort:          req.TargetPort,
		TargetUsername:      req.TargetUsername,
		TargetPassword:      req.TargetPassword,
		TargetDatabase:      req.TargetDatabase,
		TargetAuthDatabase:  req.TargetAuthDatabase,
		TargetConnectionURI: req.TargetConnectionURI,
		Metadata:            req.Metadata,
	})
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toRestoreResponse(item))
}

func (h *BackupHandler) GetRestore(c *gin.Context) {
	item, err := h.getRestore.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	response.OK(c, toRestoreResponse(item))
}

func writeBackupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrDatabaseSourceNotFound),
		errors.Is(err, domain.ErrStorageNotFound),
		errors.Is(err, domain.ErrBackupConfigNotFound),
		errors.Is(err, domain.ErrBackupNotFound),
		errors.Is(err, domain.ErrRestoreNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrStorageInUse):
		_ = c.Error(response.BadRequest(err.Error()))
	case errors.Is(err, domain.ErrInvalidInput):
		_ = c.Error(response.BadRequest(err.Error()))
	default:
		_ = c.Error(response.InternalCause(err, "backup request failed"))
	}
}

func toDatabaseSourceResponse(item *domain.DatabaseSource) databaseSourceResponse {
	return databaseSourceResponse{
		ID:               item.ID,
		Name:             item.Name,
		DBType:           string(item.DBType),
		Host:             item.Host,
		Port:             item.Port,
		Username:         item.Username,
		DatabaseName:     item.DatabaseName,
		Version:          item.Version,
		IsTLSEnabled:     item.IsTLSEnabled,
		AuthDatabase:     item.AuthDatabase,
		HasConnectionURI: strings.TrimSpace(item.ConnectionURIEncrypted) != "",
		ResourceID:       item.ResourceID,
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
	}
}

func toStorageResponse(item *domain.Storage) storageResponse {
	return storageResponse{
		ID:             item.ID,
		Name:           item.Name,
		Type:           string(item.Type),
		Config:         item.Config,
		HasCredentials: strings.TrimSpace(item.CredentialsEncrypted) != "",
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func toBackupConfigResponse(item *domain.BackupConfig) backupConfigResponse {
	return backupConfigResponse{
		ID:               item.ID,
		DatabaseSourceID: item.DatabaseSourceID,
		StorageID:        item.StorageID,
		IsEnabled:        item.IsEnabled,
		ScheduleType:     string(item.ScheduleType),
		TimeOfDay:        item.TimeOfDay,
		IntervalHours:    item.IntervalHours,
		RetentionType:    string(item.RetentionType),
		RetentionDays:    item.RetentionDays,
		RetentionCount:   item.RetentionCount,
		IsRetryIfFailed:  item.IsRetryIfFailed,
		MaxRetryCount:    item.MaxRetryCount,
		EncryptionType:   string(item.EncryptionType),
		CompressionType:  string(item.CompressionType),
		BackupMethod:     string(item.BackupMethod),
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
	}
}

func toBackupResponse(item *domain.Backup) backupResponse {
	return backupResponse{
		ID:               item.ID,
		DatabaseSourceID: item.DatabaseSourceID,
		BackupConfigID:   item.BackupConfigID,
		StorageID:        item.StorageID,
		Status:           string(item.Status),
		BackupMethod:     string(item.BackupMethod),
		FileName:         item.FileName,
		FilePath:         item.FilePath,
		FileSizeBytes:    item.FileSizeBytes,
		ChecksumSHA256:   item.ChecksumSHA256,
		StartedAt:        formatOptionalTime(item.StartedAt),
		CompletedAt:      formatOptionalTime(item.CompletedAt),
		DurationMs:       item.DurationMs,
		FailMessage:      item.FailMessage,
		EncryptionType:   string(item.EncryptionType),
		Metadata:         item.Metadata,
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
	}
}

func toRestoreResponse(item *domain.Restore) restoreResponse {
	return restoreResponse{
		ID:                 item.ID,
		BackupID:           item.BackupID,
		DatabaseSourceID:   item.DatabaseSourceID,
		Status:             string(item.Status),
		TargetHost:         item.TargetHost,
		TargetPort:         item.TargetPort,
		TargetUsername:     item.TargetUsername,
		TargetDatabaseName: item.TargetDatabaseName,
		TargetAuthDatabase: item.TargetAuthDatabase,
		HasTargetURI:       strings.TrimSpace(item.TargetURIEncrypted) != "",
		Metadata:           item.Metadata,
		StartedAt:          formatOptionalTime(item.StartedAt),
		CompletedAt:        formatOptionalTime(item.CompletedAt),
		DurationMs:         item.DurationMs,
		FailMessage:        item.FailMessage,
		CreatedAt:          item.CreatedAt.Format(time.RFC3339),
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
