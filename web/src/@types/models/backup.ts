import { z } from "zod"

export const backupSourceSchema = z.object({
  id: z.string(),
  name: z.string(),
  db_type: z.string(),
  host: z.string(),
  port: z.number(),
  username: z.string(),
  database_name: z.string(),
  version: z.string().optional(),
  is_tls_enabled: z.boolean(),
  auth_database: z.string().optional(),
  has_connection_uri: z.boolean(),
  resource_id: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})

export const storageSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  config: z.record(z.string(), z.unknown()),
  has_credentials: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
})

export const backupConfigSchema = z.object({
  id: z.string(),
  database_source_id: z.string(),
  storage_id: z.string(),
  is_enabled: z.boolean(),
  schedule_type: z.string(),
  time_of_day: z.string().optional(),
  interval_hours: z.number().optional(),
  retention_type: z.string(),
  retention_days: z.number().optional(),
  retention_count: z.number().optional(),
  is_retry_if_failed: z.boolean(),
  max_retry_count: z.number(),
  encryption_type: z.string(),
  compression_type: z.string(),
  backup_method: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
})

export const backupSchema = z.object({
  id: z.string(),
  database_source_id: z.string(),
  backup_config_id: z.string().optional(),
  storage_id: z.string(),
  status: z.string(),
  backup_method: z.string(),
  file_name: z.string().optional(),
  file_path: z.string().optional(),
  file_size_bytes: z.number().optional(),
  checksum_sha256: z.string().optional(),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  duration_ms: z.number().optional(),
  fail_message: z.string().optional(),
  encryption_type: z.string(),
  metadata: z.record(z.string(), z.unknown()).default({}),
  created_at: z.string(),
})

export const restoreSchema = z.object({
  id: z.string(),
  backup_id: z.string(),
  database_source_id: z.string().optional(),
  status: z.string(),
  target_host: z.string().optional(),
  target_port: z.number().optional(),
  target_username: z.string().optional(),
  target_database_name: z.string().optional(),
  target_auth_database: z.string().optional(),
  has_target_connection_uri: z.boolean(),
  metadata: z.record(z.string(), z.unknown()).default({}),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  duration_ms: z.number().optional(),
  fail_message: z.string().optional(),
  created_at: z.string(),
})

export type BackupSourceModel = z.infer<typeof backupSourceSchema>
export type StorageModel = z.infer<typeof storageSchema>
export type BackupConfigModel = z.infer<typeof backupConfigSchema>
export type BackupModel = z.infer<typeof backupSchema>
export type RestoreModel = z.infer<typeof restoreSchema>

export type CreateBackupSourceModel = {
  name: string
  db_type: "mysql" | "mariadb" | "postgres" | "mongodb"
  version?: string
  is_tls_enabled?: boolean
  resource_id?: string
  connection: {
    host: string
    port: number
    username: string
    password: string
    database: string
    auth_database?: string
    connection_uri?: string
  }
}

export type CreateStorageModel = {
  name: string
  type: "local" | "s3" | "minio"
  config: Record<string, unknown>
  credentials?: Record<string, unknown>
}

export type CreateBackupConfigModel = {
  database_source_id: string
  storage_id: string
  is_enabled: boolean
  schedule_type: "manual_only" | "hourly" | "daily"
  time_of_day?: string
  interval_hours?: number
  retention_type: "none" | "days" | "count"
  retention_days?: number
  retention_count?: number
  is_retry_if_failed: boolean
  max_retry_count: number
  encryption_type: "none" | "aes256"
  compression_type: "none" | "gzip"
  backup_method: "logical_dump" | "postgres_pitr"
}
