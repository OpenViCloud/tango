import type {
  ApiResponse,
  BackupConfigModel,
  BackupModel,
  BackupSourceModel,
  CreateBackupConfigModel,
  CreateBackupSourceModel,
  CreateStorageModel,
  RestoreModel,
  StorageModel,
} from "@/@types/models"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

export const backupService = {
  listBackupSources: (resourceId?: string) =>
    api
      .get<ApiResponse<{ items: BackupSourceModel[] }>>("/backup-sources", {
        params: resourceId ? { resource_id: resourceId } : undefined,
      })
      .then((res) => unwrapApiResponse(res.data).items),

  getBackupSource: (id: string) =>
    api
      .get<ApiResponse<BackupSourceModel>>(`/backup-sources/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  createBackupSource: (payload: CreateBackupSourceModel) =>
    api
      .post<BackupSourceModel>("/backup-sources", payload)
      .then((res) => res.data),

  updateBackupSource: (id: string, payload: CreateBackupSourceModel) =>
    api
      .put<BackupSourceModel>(`/backup-sources/${id}`, payload)
      .then((res) => res.data),

  deleteBackupSource: (id: string) =>
    api
      .delete(`/backup-sources/${id}`)
      .then(() => undefined),

  listStorages: () =>
    api
      .get<ApiResponse<{ items: StorageModel[] }>>("/storages")
      .then((res) => unwrapApiResponse(res.data).items),

  createStorage: (payload: CreateStorageModel) =>
    api
      .post<StorageModel>("/storages", payload)
      .then((res) => res.data),

  updateStorage: (id: string, payload: CreateStorageModel) =>
    api
      .put<StorageModel>(`/storages/${id}`, payload)
      .then((res) => res.data),

  deleteStorage: (id: string) =>
    api
      .delete(`/storages/${id}`)
      .then(() => undefined),

  createBackupConfig: (payload: CreateBackupConfigModel) =>
    api
      .post<BackupConfigModel>("/backup-configs", payload)
      .then((res) => res.data),

  updateBackupConfig: (id: string, payload: Omit<CreateBackupConfigModel, "database_source_id">) =>
    api
      .put<BackupConfigModel>(`/backup-configs/${id}`, payload)
      .then((res) => res.data),

  getBackupConfigBySource: (sourceId: string) =>
    api
      .get<ApiResponse<BackupConfigModel>>(`/backup-sources/${sourceId}/backup-config`)
      .then((res) => unwrapApiResponse(res.data)),

  listBackups: (sourceId: string) =>
    api
      .get<ApiResponse<{ items: BackupModel[] }>>(`/backup-sources/${sourceId}/backups`)
      .then((res) => unwrapApiResponse(res.data).items),

  getBackup: (id: string) =>
    api
      .get<ApiResponse<BackupModel>>(`/backups/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  triggerBackup: (sourceId: string, payload?: { storage_id?: string; metadata?: Record<string, unknown> }) =>
    api
      .post<BackupModel>(`/backup-sources/${sourceId}/backups`, payload ?? {})
      .then((res) => res.data),

  triggerRestore: (backupId: string, payload: { database_source_id?: string }) =>
    api
      .post<RestoreModel>(`/backups/${backupId}/restore`, payload)
      .then((res) => res.data),

  getRestore: (id: string) =>
    api
      .get<ApiResponse<RestoreModel>>(`/restores/${id}`)
      .then((res) => unwrapApiResponse(res.data)),
}
