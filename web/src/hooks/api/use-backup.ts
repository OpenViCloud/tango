import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { CreateBackupConfigModel, CreateBackupSourceModel, CreateStorageModel } from "@/@types/models"
import { backupService } from "@/services/api/backup-service"

export const BACKUP_QUERY_KEYS = {
  sources: (resourceId?: string) => ["backupSources", resourceId || "all"],
  source: (id: string) => ["backupSources", id],
  storages: () => ["backupStorages"],
  config: (sourceId: string) => ["backupConfig", sourceId],
  backups: (sourceId: string) => ["backups", sourceId],
  backup: (id: string) => ["backup", id],
  restore: (id: string) => ["restore", id],
}

export const useGetBackupSourceList = (resourceId?: string) =>
  useQuery({
    queryKey: BACKUP_QUERY_KEYS.sources(resourceId),
    queryFn: () => backupService.listBackupSources(resourceId),
  })

export const useCreateBackupSource = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateBackupSourceModel) => backupService.createBackupSource(payload),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources() })
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources(data.resource_id) })
    },
  })
}

export const useUpdateBackupSource = (resourceId?: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: CreateBackupSourceModel }) =>
      backupService.updateBackupSource(id, payload),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources() })
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources(resourceId) })
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.source(data.id) })
    },
  })
}

export const useDeleteBackupSource = (resourceId?: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => backupService.deleteBackupSource(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources() })
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.sources(resourceId) })
    },
  })
}

export const useGetStorageList = () =>
  useQuery({
    queryKey: BACKUP_QUERY_KEYS.storages(),
    queryFn: () => backupService.listStorages(),
  })

export const useCreateStorage = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateStorageModel) => backupService.createStorage(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.storages() })
    },
  })
}

export const useUpdateStorage = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: CreateStorageModel }) =>
      backupService.updateStorage(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.storages() })
    },
  })
}

export const useDeleteStorage = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => backupService.deleteStorage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.storages() })
    },
  })
}

export const useGetBackupConfig = (sourceId: string) =>
  useQuery({
    queryKey: BACKUP_QUERY_KEYS.config(sourceId),
    queryFn: () => backupService.getBackupConfigBySource(sourceId),
    enabled: !!sourceId,
    retry: false,
  })

export const useCreateBackupConfig = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateBackupConfigModel) => backupService.createBackupConfig(payload),
    onSuccess: (_data, payload) => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.config(payload.database_source_id) })
    },
  })
}

export const useUpdateBackupConfig = (sourceId: string, configId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: Omit<CreateBackupConfigModel, "database_source_id">) =>
      backupService.updateBackupConfig(configId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.config(sourceId) })
    },
  })
}

export const useGetBackupList = (sourceId: string) =>
  useQuery({
    queryKey: BACKUP_QUERY_KEYS.backups(sourceId),
    queryFn: () => backupService.listBackups(sourceId),
    enabled: !!sourceId,
    refetchInterval: 5000,
  })

export const useTriggerBackup = (sourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload?: { storage_id?: string; metadata?: Record<string, unknown> }) =>
      backupService.triggerBackup(sourceId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.backups(sourceId) })
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.config(sourceId) })
    },
  })
}

export const useTriggerRestore = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ backupId, payload }: { backupId: string; payload: { database_source_id?: string } }) =>
      backupService.triggerRestore(backupId, payload),
    onSuccess: (restore) => {
      queryClient.invalidateQueries({ queryKey: BACKUP_QUERY_KEYS.restore(restore.id) })
    },
  })
}

export const useGetRestore = (restoreId: string) =>
  useQuery({
    queryKey: BACKUP_QUERY_KEYS.restore(restoreId),
    queryFn: () => backupService.getRestore(restoreId),
    enabled: !!restoreId,
    refetchInterval: 4000,
  })
