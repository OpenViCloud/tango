package query

import (
	"context"
	"strings"

	"tango/internal/domain"
)

type GetDatabaseSourceHandler struct {
	repo domain.DatabaseSourceRepository
}
type ListDatabaseSourcesHandler struct {
	repo domain.DatabaseSourceRepository
}
type GetStorageHandler struct{ repo domain.StorageRepository }
type ListStoragesHandler struct{ repo domain.StorageRepository }
type GetBackupConfigHandler struct{ repo domain.BackupConfigRepository }
type GetBackupConfigByDatabaseSourceHandler struct{ repo domain.BackupConfigRepository }
type GetBackupHandler struct{ repo domain.BackupRepository }
type ListBackupsByDatabaseSourceHandler struct{ repo domain.BackupRepository }
type GetRestoreHandler struct{ repo domain.RestoreRepository }

func NewGetDatabaseSourceHandler(repo domain.DatabaseSourceRepository) *GetDatabaseSourceHandler {
	return &GetDatabaseSourceHandler{repo: repo}
}
func NewListDatabaseSourcesHandler(repo domain.DatabaseSourceRepository) *ListDatabaseSourcesHandler {
	return &ListDatabaseSourcesHandler{repo: repo}
}
func NewGetStorageHandler(repo domain.StorageRepository) *GetStorageHandler {
	return &GetStorageHandler{repo: repo}
}
func NewListStoragesHandler(repo domain.StorageRepository) *ListStoragesHandler {
	return &ListStoragesHandler{repo: repo}
}
func NewGetBackupConfigHandler(repo domain.BackupConfigRepository) *GetBackupConfigHandler {
	return &GetBackupConfigHandler{repo: repo}
}
func NewGetBackupConfigByDatabaseSourceHandler(repo domain.BackupConfigRepository) *GetBackupConfigByDatabaseSourceHandler {
	return &GetBackupConfigByDatabaseSourceHandler{repo: repo}
}
func NewGetBackupHandler(repo domain.BackupRepository) *GetBackupHandler {
	return &GetBackupHandler{repo: repo}
}
func NewListBackupsByDatabaseSourceHandler(repo domain.BackupRepository) *ListBackupsByDatabaseSourceHandler {
	return &ListBackupsByDatabaseSourceHandler{repo: repo}
}
func NewGetRestoreHandler(repo domain.RestoreRepository) *GetRestoreHandler {
	return &GetRestoreHandler{repo: repo}
}

func (h *GetDatabaseSourceHandler) Handle(ctx context.Context, id string) (*domain.DatabaseSource, error) {
	return h.repo.GetByID(ctx, id)
}
func (h *ListDatabaseSourcesHandler) Handle(ctx context.Context, resourceID string) ([]*domain.DatabaseSource, error) {
	if strings.TrimSpace(resourceID) != "" {
		return h.repo.ListByResourceID(ctx, strings.TrimSpace(resourceID))
	}
	return h.repo.List(ctx)
}
func (h *GetStorageHandler) Handle(ctx context.Context, id string) (*domain.Storage, error) {
	return h.repo.GetByID(ctx, id)
}
func (h *ListStoragesHandler) Handle(ctx context.Context) ([]*domain.Storage, error) {
	return h.repo.List(ctx)
}
func (h *GetBackupConfigHandler) Handle(ctx context.Context, id string) (*domain.BackupConfig, error) {
	return h.repo.GetByID(ctx, id)
}
func (h *GetBackupConfigByDatabaseSourceHandler) Handle(ctx context.Context, databaseSourceID string) (*domain.BackupConfig, error) {
	return h.repo.GetByDatabaseSourceID(ctx, databaseSourceID)
}
func (h *GetBackupHandler) Handle(ctx context.Context, id string) (*domain.Backup, error) {
	return h.repo.GetByID(ctx, id)
}
func (h *ListBackupsByDatabaseSourceHandler) Handle(ctx context.Context, databaseSourceID string) ([]*domain.Backup, error) {
	return h.repo.ListByDatabaseSourceID(ctx, databaseSourceID)
}
func (h *GetRestoreHandler) Handle(ctx context.Context, id string) (*domain.Restore, error) {
	return h.repo.GetByID(ctx, id)
}
