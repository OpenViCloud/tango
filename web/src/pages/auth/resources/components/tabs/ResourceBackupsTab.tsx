import {
  HardDriveDownload,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  ServerCrash,
  Trash2,
} from "lucide-react"
import { useMemo, useState } from "react"
import { toast } from "sonner"

import type {
  BackupSourceModel,
  CreateBackupSourceModel,
  CreateStorageModel,
  ResourceModel,
  StorageModel,
} from "@/@types/models"
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import {
  useCreateBackupConfig,
  useCreateBackupSource,
  useCreateStorage,
  useDeleteBackupSource,
  useDeleteStorage,
  useGetBackupConfig,
  useGetBackupList,
  useGetBackupSourceList,
  useGetRestore,
  useGetStorageList,
  useTriggerBackup,
  useTriggerRestore,
  useUpdateBackupConfig,
  useUpdateBackupSource,
  useUpdateStorage,
} from "@/hooks/api/use-backup"
import {
  useGetResourceConnectionInfo,
  useGetResourceEnvVars,
} from "@/hooks/api/use-project"
import { cn } from "@/lib/utils"

type ResourceBackupsTabProps = {
  resource: ResourceModel
}

type BackupConfigFormState = {
  is_enabled: boolean
  schedule_type: "manual_only" | "hourly" | "daily"
  time_of_day: string
  interval_hours: number
  retention_type: "none" | "days" | "count"
  retention_days: number
  retention_count: number
  is_retry_if_failed: boolean
  max_retry_count: number
  encryption_type: "none" | "aes256"
  compression_type: "none" | "gzip"
  backup_method: "logical_dump" | "postgres_pitr"
}

type StorageFormState = {
  name: string
  path: string
}

type HostMode = "internal" | "external" | "manual"

const ADD_NEW_STORAGE_VALUE = "__add_new_storage__"

export function ResourceBackupsTab({ resource }: ResourceBackupsTabProps) {
  const { data: connectionInfo } = useGetResourceConnectionInfo(resource.id)
  const { data: envVars } = useGetResourceEnvVars(resource.id)
  const { data: sources = [], isLoading: isLoadingSources } =
    useGetBackupSourceList(resource.id)
  const { data: storages = [], isLoading: isLoadingStorages } =
    useGetStorageList()

  const [selectedSourceId, setSelectedSourceId] = useState("")
  const [selectedStorageId, setSelectedStorageId] = useState("")
  const [lastRestoreId, setLastRestoreId] = useState("")
  const [sourceSheetOpen, setSourceSheetOpen] = useState(false)
  const [storageSheetOpen, setStorageSheetOpen] = useState(false)
  const [configSheetOpen, setConfigSheetOpen] = useState(false)
  const [activeTab, setActiveTab] = useState("config")
  const [editingSourceId, setEditingSourceId] = useState<string | null>(null)
  const [editingStorageId, setEditingStorageId] = useState<string | null>(null)
  const [hostMode, setHostMode] = useState<HostMode>("manual")

  const prefill = useMemo(
    () => buildPrefillFromResource(resource, connectionInfo, envVars ?? []),
    [resource, connectionInfo, envVars]
  )

  const emptySourceForm = useMemo<CreateBackupSourceModel>(
    () => ({
      name: `${resource.name} backup`,
      db_type: prefill.dbType,
      version: "",
      is_tls_enabled: false,
      resource_id: resource.id,
      connection: {
        host: prefill.host,
        port: prefill.port,
        username: prefill.username,
        password: prefill.password,
        database: prefill.database,
        auth_database: prefill.authDatabase,
        connection_uri: "",
      },
    }),
    [prefill, resource.id, resource.name]
  )

  const hostOptions = useMemo(() => {
    const options: Array<{ value: HostMode; label: string; host: string }> = []
    if (prefill.internalHost) {
      options.push({
        value: "internal",
        label: `Internal container (${prefill.internalHost})`,
        host: prefill.internalHost,
      })
    }
    if (prefill.externalHost) {
      options.push({
        value: "external",
        label: `Published host (${prefill.externalHost})`,
        host: prefill.externalHost,
      })
    }
    return options
  }, [prefill.externalHost, prefill.internalHost])

  const emptyStorageForm = useMemo<StorageFormState>(
    () => ({
      name: `${resource.name} local storage`,
      path: `/tmp/tango-backups/${resource.name}`,
    }),
    [resource.name]
  )

  const [sourceForm, setSourceForm] =
    useState<CreateBackupSourceModel>(emptySourceForm)
  const [storageForm, setStorageForm] =
    useState<StorageFormState>(emptyStorageForm)

  const effectiveSelectedSourceId = useMemo(() => {
    if (
      selectedSourceId &&
      sources.some((item) => item.id === selectedSourceId)
    ) {
      return selectedSourceId
    }
    const matchingName = sources.find((item) =>
      item.name.toLowerCase().includes(resource.name.toLowerCase())
    )
    return matchingName?.id ?? sources[0]?.id ?? ""
  }, [resource.name, selectedSourceId, sources])

  const selectedSource = useMemo(
    () => sources.find((item) => item.id === effectiveSelectedSourceId) ?? null,
    [sources, effectiveSelectedSourceId]
  )
  const { data: backupConfig } = useGetBackupConfig(effectiveSelectedSourceId)
  const defaultStorageId = useMemo(
    () =>
      (storages.find((item) => item.type === "local") ?? storages[0])?.id ?? "",
    [storages]
  )
  const effectiveStorageId =
    backupConfig?.storage_id || selectedStorageId || defaultStorageId
  const configSummary = useMemo(
    () => ({
      is_enabled: backupConfig?.is_enabled ?? true,
      schedule_type: normalizeScheduleType(
        backupConfig?.schedule_type || "manual_only"
      ),
      time_of_day: backupConfig?.time_of_day || "02:00",
      interval_hours: backupConfig?.interval_hours || 24,
      retention_type: normalizeRetentionType(
        backupConfig?.retention_type || "count"
      ),
      retention_days: backupConfig?.retention_days || 7,
      retention_count: backupConfig?.retention_count || 7,
      is_retry_if_failed: backupConfig?.is_retry_if_failed ?? true,
      max_retry_count: backupConfig?.max_retry_count ?? 2,
      encryption_type: normalizeEncryptionType(
        backupConfig?.encryption_type || "none"
      ),
      compression_type: normalizeCompressionType(
        backupConfig?.compression_type || "gzip"
      ),
      backup_method: normalizeBackupMethod(
        backupConfig?.backup_method || "logical_dump"
      ),
    }),
    [backupConfig]
  )

  const selectedStorage = useMemo(
    () => storages.find((item) => item.id === effectiveStorageId) ?? null,
    [effectiveStorageId, storages]
  )

  const { data: backups = [], isLoading: isLoadingBackups } = useGetBackupList(
    effectiveSelectedSourceId
  )
  const { data: restoreStatus } = useGetRestore(lastRestoreId)

  const latestFailedBackup = useMemo(
    () =>
      backups.find(
        (backup) => backup.status === "failed" && backup.fail_message
      ),
    [backups]
  )

  const createSourceMutation = useCreateBackupSource()
  const updateSourceMutation = useUpdateBackupSource(resource.id)
  const deleteSourceMutation = useDeleteBackupSource(resource.id)
  const createStorageMutation = useCreateStorage()
  const updateStorageMutation = useUpdateStorage()
  const deleteStorageMutation = useDeleteStorage()
  const createConfigMutation = useCreateBackupConfig()
  const updateConfigMutation = useUpdateBackupConfig(
    effectiveSelectedSourceId,
    backupConfig?.id ?? ""
  )
  const triggerBackupMutation = useTriggerBackup(effectiveSelectedSourceId)
  const triggerRestoreMutation = useTriggerRestore()

  const openCreateSourceSheet = () => {
    setEditingSourceId(null)
    setSourceForm(emptySourceForm)
    setSourceSheetOpen(true)
  }

  const openEditSourceSheet = () => {
    if (!selectedSource) return
    setEditingSourceId(selectedSource.id)
    setSourceForm({
      name: selectedSource.name,
      db_type: selectedSource.db_type as
        | "mysql"
        | "mariadb"
        | "postgres"
        | "mongodb",
      version: selectedSource.version || "",
      is_tls_enabled: selectedSource.is_tls_enabled,
      resource_id: selectedSource.resource_id || resource.id,
      connection: {
        host: selectedSource.host,
        port: selectedSource.port,
        username: selectedSource.username,
        password: "",
        database: selectedSource.database_name,
        auth_database: selectedSource.auth_database || "",
        connection_uri: "",
      },
    })
    setHostMode(resolveHostMode(selectedSource, prefill))
    setSourceSheetOpen(true)
  }

  const openCreateStorageSheet = () => {
    setEditingStorageId(null)
    setStorageForm(emptyStorageForm)
    setStorageSheetOpen(true)
  }

  const openEditStorageSheet = (storage: StorageModel) => {
    setEditingStorageId(storage.id)
    setStorageForm({
      name: storage.name,
      path: String(storage.config.base_path || ""),
    })
    setStorageSheetOpen(true)
  }

  const handleSaveSource = async () => {
    try {
      if (editingSourceId) {
        const updated = await updateSourceMutation.mutateAsync({
          id: editingSourceId,
          payload: sourceForm,
        })
        setSelectedSourceId(updated.id)
        toast.success("Backup source updated")
      } else {
        const created = await createSourceMutation.mutateAsync(sourceForm)
        setSelectedSourceId(created.id)
        toast.success("Backup source created")
      }
      setSourceSheetOpen(false)
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleDeleteSource = async (sourceId: string) => {
    try {
      await deleteSourceMutation.mutateAsync(sourceId)
      if (selectedSourceId === sourceId) {
        setSelectedSourceId("")
      }
      toast.success("Backup source deleted")
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleSaveStorage = async () => {
    const payload: CreateStorageModel = {
      name: storageForm.name.trim(),
      type: "local",
      config: { base_path: storageForm.path.trim() },
      credentials: {},
    }

    try {
      if (editingStorageId) {
        const updated = await updateStorageMutation.mutateAsync({
          id: editingStorageId,
          payload,
        })
        if (selectedStorageId === editingStorageId) {
          setSelectedStorageId(updated.id)
        }
        toast.success("Storage updated")
      } else {
        const created = await createStorageMutation.mutateAsync(payload)
        setSelectedStorageId(created.id)
        toast.success("Local storage created")
      }
      setStorageSheetOpen(false)
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleDeleteStorage = async (storageId: string) => {
    try {
      await deleteStorageMutation.mutateAsync(storageId)
      if (selectedStorageId === storageId) {
        setSelectedStorageId("")
      }
      toast.success("Storage deleted")
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleSaveConfigDraft = async (
    storageId: string,
    configForm: BackupConfigFormState
  ) => {
    if (!effectiveSelectedSourceId || !storageId) {
      toast.error("Select a backup source and storage first")
      return
    }
    const payload = {
      database_source_id: effectiveSelectedSourceId,
      storage_id: storageId,
      ...configForm,
    }
    try {
      if (backupConfig?.id) {
        await updateConfigMutation.mutateAsync({
          storage_id: storageId,
          ...configForm,
        })
        toast.success("Backup config updated")
      } else {
        await createConfigMutation.mutateAsync(payload)
        toast.success("Backup config created")
      }
      setSelectedStorageId(storageId)
      setConfigSheetOpen(false)
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleTriggerBackup = async () => {
    if (!effectiveSelectedSourceId) {
      toast.error("Select a backup source first")
      return
    }
    try {
      await triggerBackupMutation.mutateAsync({
        storage_id: effectiveStorageId || undefined,
        metadata: {
          triggered_from: "resource_tab",
          resource_name: resource.name,
        },
      })
      toast.success("Backup started")
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  const handleRestore = async (backupId: string) => {
    if (!effectiveSelectedSourceId) {
      toast.error("Select a target backup source first")
      return
    }
    try {
      const restore = await triggerRestoreMutation.mutateAsync({
        backupId,
        payload: { database_source_id: effectiveSelectedSourceId },
      })
      setLastRestoreId(restore.id)
      toast.success("Restore started")
    } catch (error) {
      toast.error(getErrorMessage(error))
    }
  }

  return (
    <>
      <div className="flex min-w-0 flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8">
        <div className="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)]">
          <Card className="xl:sticky xl:top-6 xl:h-fit">
            <CardHeader className="gap-3">
              <CardTitle className="text-base">Backup Sources</CardTitle>
              <CardDescription>
                Choose the database connection you want to back up for this
                resource.
              </CardDescription>
              <Button type="button" onClick={openCreateSourceSheet}>
                <Plus data-icon="inline-start" />
                Add source
              </Button>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              {isLoadingSources ? (
                <>
                  <Skeleton className="h-24 w-full rounded-xl" />
                  <Skeleton className="h-24 w-full rounded-xl" />
                </>
              ) : sources.length === 0 ? (
                <div className="rounded-xl border border-dashed px-4 py-6 text-sm text-muted-foreground">
                  No backup sources yet. Create one to continue.
                </div>
              ) : (
                sources.map((source) => {
                  const sourceBackups =
                    source.id === effectiveSelectedSourceId ? backups : []
                  const latestBackup = sourceBackups[0]
                  const hasError = latestBackup?.status === "failed"

                  return (
                    <button
                      key={source.id}
                      type="button"
                      onClick={() => setSelectedSourceId(source.id)}
                      className={cn(
                        "flex flex-col gap-3 rounded-xl border px-4 py-4 text-left transition-colors",
                        effectiveSelectedSourceId === source.id
                          ? "border-primary bg-primary/5"
                          : "border-border/70 bg-card hover:bg-muted/40"
                      )}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="truncate font-medium text-foreground">
                            {source.name}
                          </div>
                          <div className="mt-1 text-xs text-muted-foreground">
                            {source.host}:{source.port}
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge variant={hasError ? "warning" : "secondary"}>
                            {source.db_type}
                          </Badge>
                          <DeleteConfirmDialog
                            title="Delete backup source"
                            description={`This will delete ${source.name}, its backup config, and backup/restore history for this source.`}
                            confirmLabel="Delete"
                            cancelLabel="Cancel"
                            onConfirm={() => handleDeleteSource(source.id)}
                            trigger={
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="size-8 shrink-0"
                                disabled={deleteSourceMutation.isPending}
                                onClick={(event) => event.stopPropagation()}
                              >
                                <Trash2 className="size-4" />
                              </Button>
                            }
                          />
                        </div>
                      </div>

                      <div className="flex flex-col gap-1 text-xs text-muted-foreground">
                        <span>
                          Storage:{" "}
                          {effectiveSelectedSourceId === source.id
                            ? selectedStorage?.name || "Not set"
                            : "Open source to inspect"}
                        </span>
                        <span>
                          Last backup:{" "}
                          {effectiveSelectedSourceId === source.id &&
                          latestBackup
                            ? formatRelativeDate(latestBackup.created_at)
                            : "Open source to inspect"}
                        </span>
                        {hasError ? (
                          <span className="font-medium text-destructive">
                            Has backup error
                          </span>
                        ) : null}
                      </div>
                    </button>
                  )
                })
              )}
            </CardContent>
          </Card>

          <div className="min-w-0">
            {!selectedSource ? (
              <Card>
                <CardContent className="flex min-h-64 items-center justify-center px-6 py-12 text-sm text-muted-foreground">
                  Select a backup source or add a new one to configure database
                  backups.
                </CardContent>
              </Card>
            ) : (
              <Tabs
                value={activeTab}
                onValueChange={setActiveTab}
                className="flex flex-col gap-4"
              >
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 className="text-2xl font-semibold tracking-tight text-foreground">
                      {selectedSource.name}
                    </h2>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {selectedSource.host}:{selectedSource.port} ·{" "}
                      {selectedSource.database_name}
                    </p>
                  </div>
                  <TabsList className="w-full sm:w-auto">
                    <TabsTrigger value="config" className="flex-1 sm:flex-none">
                      Config
                    </TabsTrigger>
                    <TabsTrigger
                      value="storage"
                      className="flex-1 sm:flex-none"
                    >
                      Storage
                    </TabsTrigger>
                    <TabsTrigger
                      value="backups"
                      className="flex-1 sm:flex-none"
                    >
                      Backups
                    </TabsTrigger>
                  </TabsList>
                </div>

                <TabsContent
                  value="config"
                  className="mt-0 flex flex-col gap-6"
                >
                  {latestFailedBackup?.fail_message ? (
                    <Card className="border-destructive/50 bg-destructive/5">
                      <CardHeader className="gap-2">
                        <CardTitle className="flex items-center gap-2 text-destructive">
                          <ServerCrash className="size-4" />
                          Last backup error
                        </CardTitle>
                        <CardDescription className="text-destructive/80">
                          The latest backup for this source failed. Update the
                          connection or policy below, then run a new backup.
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <Textarea
                          readOnly
                          value={latestFailedBackup.fail_message}
                          className="min-h-24 resize-none border-destructive/30 bg-background"
                        />
                      </CardContent>
                    </Card>
                  ) : null}

                  <div className="grid gap-6 xl:grid-cols-2">
                    <Card>
                      <CardHeader className="flex-row items-start justify-between gap-3 space-y-0">
                        <div className="space-y-1.5">
                          <CardTitle className="text-base">
                            Database settings
                          </CardTitle>
                          <CardDescription>
                            Connection details stored in the selected backup
                            source.
                          </CardDescription>
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          onClick={openEditSourceSheet}
                        >
                          <Pencil className="size-4" />
                        </Button>
                      </CardHeader>
                      <CardContent className="grid gap-3 text-sm">
                        <InfoRow
                          label="Database type"
                          value={selectedSource.db_type.toUpperCase()}
                        />
                        <InfoRow
                          label="Version"
                          value={
                            selectedSource.version || "Auto-detect on runner"
                          }
                        />
                        <InfoRow label="Host" value={selectedSource.host} />
                        <InfoRow
                          label="Port"
                          value={String(selectedSource.port)}
                        />
                        <InfoRow
                          label="Username"
                          value={selectedSource.username}
                        />
                        <InfoRow
                          label="Database"
                          value={selectedSource.database_name}
                        />
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader className="flex-row items-start justify-between gap-3 space-y-0">
                        <div className="space-y-1.5">
                          <CardTitle className="text-base">
                            Backup config
                          </CardTitle>
                          <CardDescription>
                            Choose where artifacts are stored and how backups
                            should behave.
                          </CardDescription>
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          onClick={() => setConfigSheetOpen(true)}
                        >
                          <Pencil className="size-4" />
                        </Button>
                      </CardHeader>
                      <CardContent className="grid gap-3 text-sm">
                        <InfoRow
                          label="Storage"
                          value={selectedStorage?.name || "Not configured"}
                        />
                        <InfoRow
                          label="Compression"
                          value={configSummary.compression_type}
                        />
                        <InfoRow
                          label="Retention"
                          value={
                            configSummary.retention_type === "count"
                              ? `Count (${configSummary.retention_count})`
                              : configSummary.retention_type === "days"
                                ? `Days (${configSummary.retention_days})`
                                : "None"
                          }
                        />
                        <InfoRow
                          label="Retry failed backups"
                          value={
                            configSummary.is_retry_if_failed ? "Yes" : "No"
                          }
                        />
                        <InfoRow
                          label="Max retry count"
                          value={String(configSummary.max_retry_count)}
                        />
                      </CardContent>
                    </Card>
                  </div>
                </TabsContent>

                <TabsContent value="storage" className="mt-0">
                  <Card>
                    <CardHeader className="flex-row items-start justify-between gap-3 space-y-0">
                      <div className="space-y-1.5">
                        <CardTitle className="text-base">Storage</CardTitle>
                        <CardDescription>
                          Manage destinations where backup artifacts are stored
                          for this project.
                        </CardDescription>
                      </div>
                      <Button type="button" onClick={openCreateStorageSheet}>
                        <Plus data-icon="inline-start" />
                        Add storage
                      </Button>
                    </CardHeader>
                    <CardContent>
                      {isLoadingStorages ? (
                        <div className="flex flex-col gap-3">
                          <Skeleton className="h-14 w-full rounded-xl" />
                          <Skeleton className="h-14 w-full rounded-xl" />
                        </div>
                      ) : storages.length === 0 ? (
                        <div className="rounded-lg border border-dashed px-4 py-6 text-sm text-muted-foreground">
                          No storage destinations yet. Add one to save backup
                          artifacts.
                        </div>
                      ) : (
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Name</TableHead>
                              <TableHead>Type</TableHead>
                              <TableHead>Base path</TableHead>
                              <TableHead>Updated</TableHead>
                              <TableHead className="w-40 text-right">
                                Actions
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {storages.map((storage) => {
                              const isSelected =
                                effectiveStorageId === storage.id
                              return (
                                <TableRow
                                  key={storage.id}
                                  data-state={
                                    isSelected ? "selected" : undefined
                                  }
                                >
                                  <TableCell>
                                    <div className="flex items-center gap-2">
                                      <span className="font-medium text-foreground">
                                        {storage.name}
                                      </span>
                                      {isSelected ? (
                                        <Badge variant="secondary">
                                          Selected
                                        </Badge>
                                      ) : null}
                                    </div>
                                  </TableCell>
                                  <TableCell>{storage.type}</TableCell>
                                  <TableCell className="max-w-[320px] truncate">
                                    {String(storage.config.base_path || "n/a")}
                                  </TableCell>
                                  <TableCell>
                                    {formatDate(storage.updated_at)}
                                  </TableCell>
                                  <TableCell>
                                    <div className="flex justify-end gap-2">
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={() =>
                                          openEditStorageSheet(storage)
                                        }
                                      >
                                        <Pencil data-icon="inline-start" />
                                        Edit
                                      </Button>
                                      <DeleteConfirmDialog
                                        title="Delete storage"
                                        description={`This will delete ${storage.name} if it is not currently used by any backup config or backup history.`}
                                        confirmLabel="Delete"
                                        cancelLabel="Cancel"
                                        onConfirm={() =>
                                          handleDeleteStorage(storage.id)
                                        }
                                        trigger={
                                          <Button
                                            type="button"
                                            variant="outline"
                                            size="sm"
                                            disabled={
                                              deleteStorageMutation.isPending
                                            }
                                          >
                                            <Trash2 data-icon="inline-start" />
                                            Delete
                                          </Button>
                                        }
                                      />
                                    </div>
                                  </TableCell>
                                </TableRow>
                              )
                            })}
                          </TableBody>
                        </Table>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>

                <TabsContent value="backups" className="mt-0">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center justify-between gap-3">
                        <span>Backup runs</span>
                        <Button
                          type="button"
                          onClick={handleTriggerBackup}
                          disabled={
                            !effectiveSelectedSourceId ||
                            triggerBackupMutation.isPending
                          }
                        >
                          <RefreshCw data-icon="inline-start" />
                          {triggerBackupMutation.isPending
                            ? "Starting..."
                            : "Run backup"}
                        </Button>
                      </CardTitle>
                      <CardDescription>
                        Run a fresh snapshot for the selected source and restore
                        any completed backup back into the same target.
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-4">
                      {restoreStatus ? (
                        <div className="rounded-lg border border-border/70 bg-muted/30 px-4 py-3 text-sm">
                          Latest restore status:{" "}
                          <span className="font-medium text-foreground">
                            {restoreStatus.status}
                          </span>
                          {restoreStatus.fail_message ? (
                            <span className="text-destructive">
                              {" "}
                              · {restoreStatus.fail_message}
                            </span>
                          ) : null}
                        </div>
                      ) : null}

                      {isLoadingBackups ? (
                        <div className="flex flex-col gap-3">
                          <Skeleton className="h-20 w-full rounded-xl" />
                          <Skeleton className="h-20 w-full rounded-xl" />
                        </div>
                      ) : backups.length === 0 ? (
                        <div className="rounded-lg border border-dashed px-4 py-6 text-sm text-muted-foreground">
                          No backups yet for this source.
                        </div>
                      ) : (
                        <div className="flex flex-col gap-3">
                          {backups.map((backup) => (
                            <div
                              key={backup.id}
                              className="flex flex-col gap-3 rounded-xl border border-border/70 px-4 py-4 lg:flex-row lg:items-center lg:justify-between"
                            >
                              <div className="min-w-0 flex-1">
                                <div className="flex flex-wrap items-center gap-2">
                                  <span className="font-medium text-foreground">
                                    {backup.file_name || backup.id}
                                  </span>
                                  <Badge variant="outline">
                                    {backup.status}
                                  </Badge>
                                  <Badge variant="secondary">
                                    {backup.backup_method}
                                  </Badge>
                                </div>
                                <div className="mt-1 flex flex-wrap gap-3 text-xs text-muted-foreground">
                                  <span>
                                    Created: {formatDate(backup.created_at)}
                                  </span>
                                  {typeof backup.file_size_bytes ===
                                  "number" ? (
                                    <span>
                                      Size:{" "}
                                      {formatBytes(backup.file_size_bytes)}
                                    </span>
                                  ) : null}
                                  {backup.fail_message ? (
                                    <span className="text-destructive">
                                      Error: {backup.fail_message}
                                    </span>
                                  ) : null}
                                </div>
                                {backup.file_path ? (
                                  <Textarea
                                    readOnly
                                    value={backup.file_path}
                                    className="mt-3 min-h-14 resize-none text-xs"
                                  />
                                ) : null}
                              </div>

                              <DeleteConfirmDialog
                                title="Restore backup"
                                description={`This will restore ${backup.file_name || backup.id} into ${selectedSource.name}.`}
                                confirmLabel="Restore"
                                cancelLabel="Cancel"
                                onConfirm={() => handleRestore(backup.id)}
                                trigger={
                                  <Button
                                    type="button"
                                    variant="outline"
                                    disabled={
                                      backup.status !== "completed" ||
                                      triggerRestoreMutation.isPending
                                    }
                                  >
                                    <RotateCcw data-icon="inline-start" />
                                    Restore
                                  </Button>
                                }
                              />
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            )}
          </div>
        </div>
      </div>

      <Sheet
        open={sourceSheetOpen}
        onOpenChange={(open) => {
          setSourceSheetOpen(open)
          if (!open) setEditingSourceId(null)
        }}
      >
        <SheetContent className="flex w-full flex-col sm:max-w-lg">
          <SheetHeader className="border-b pb-4">
            <SheetTitle>
              {editingSourceId ? "Edit backup source" : "Add backup source"}
            </SheetTitle>
            <SheetDescription>
              {editingSourceId
                ? "Update the selected database connection."
                : "Create a reusable database connection for backup and restore."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto px-4 py-6">
            <FieldGroup>
              <Field>
                <FieldLabel>Name</FieldLabel>
                <Input
                  value={sourceForm.name}
                  onChange={(event) =>
                    setSourceForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
              </Field>

              <Field>
                <FieldLabel>Database type</FieldLabel>
                <Select
                  value={sourceForm.db_type}
                  onValueChange={(
                    value: "mysql" | "mariadb" | "postgres" | "mongodb"
                  ) =>
                    setSourceForm((current) => ({ ...current, db_type: value }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Choose a database type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="mysql">MySQL</SelectItem>
                      <SelectItem value="mariadb">MariaDB</SelectItem>
                      <SelectItem value="postgres">PostgreSQL</SelectItem>
                      <SelectItem value="mongodb">MongoDB</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FieldDescription>
                  MySQL and MariaDB backup/restore are wired now. PostgreSQL and
                  MongoDB can be prepared in the UI but backend execution is not
                  wired yet.
                </FieldDescription>
              </Field>

              <Field>
                <div className="flex items-center justify-between gap-3">
                  <FieldLabel>Host</FieldLabel>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">
                      Manual
                    </span>
                    <Switch
                      checked={hostMode === "manual"}
                      onCheckedChange={(checked) => {
                        if (checked) {
                          setHostMode("manual")
                          return
                        }
                        const fallbackMode = hostOptions[0]?.value ?? "manual"
                        setHostMode(fallbackMode)
                        const selectedOption = hostOptions.find(
                          (option) => option.value === fallbackMode
                        )
                        if (selectedOption) {
                          setSourceForm((current) => ({
                            ...current,
                            connection: {
                              ...current.connection,
                              host: selectedOption.host,
                            },
                          }))
                        }
                      }}
                    />
                  </div>
                </div>
                {hostMode === "manual" ? (
                  <Input
                    value={sourceForm.connection.host}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        connection: {
                          ...current.connection,
                          host: event.target.value,
                        },
                      }))
                    }
                  />
                ) : (
                  <Select
                    value={hostMode}
                    onValueChange={(value: HostMode) => {
                      setHostMode(value)
                      const selectedOption = hostOptions.find(
                        (option) => option.value === value
                      )
                      if (selectedOption) {
                        setSourceForm((current) => ({
                          ...current,
                          connection: {
                            ...current.connection,
                            host: selectedOption.host,
                          },
                        }))
                      }
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Choose host preset" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {hostOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
                <FieldDescription>
                  Use the internal container host when backup-runner is on the
                  same Docker network. Switch to manual if you need a custom
                  address.
                </FieldDescription>
              </Field>
              <Field>
                <FieldLabel>Port</FieldLabel>
                <Input
                  type="number"
                  value={sourceForm.connection.port}
                  onChange={(event) =>
                    setSourceForm((current) => ({
                      ...current,
                      connection: {
                        ...current.connection,
                        port: Number(event.target.value) || 0,
                      },
                    }))
                  }
                />
              </Field>

              <div className="grid gap-4 md:grid-cols-2">
                <Field>
                  <FieldLabel>Username</FieldLabel>
                  <Input
                    value={sourceForm.connection.username}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        connection: {
                          ...current.connection,
                          username: event.target.value,
                        },
                      }))
                    }
                  />
                </Field>
                <Field>
                  <FieldLabel>Password</FieldLabel>
                  <Input
                    type="password"
                    value={sourceForm.connection.password}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        connection: {
                          ...current.connection,
                          password: event.target.value,
                        },
                      }))
                    }
                  />
                </Field>
              </div>

              <Field>
                <FieldLabel>Database</FieldLabel>
                <Input
                  value={sourceForm.connection.database}
                  onChange={(event) =>
                    setSourceForm((current) => ({
                      ...current,
                      connection: {
                        ...current.connection,
                        database: event.target.value,
                      },
                    }))
                  }
                />
              </Field>
            </FieldGroup>
          </div>
          <SheetFooter className="border-t pt-4 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="outline"
              onClick={() => setSourceSheetOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleSaveSource}
              disabled={
                createSourceMutation.isPending || updateSourceMutation.isPending
              }
            >
              <Save data-icon="inline-start" />
              {createSourceMutation.isPending || updateSourceMutation.isPending
                ? "Saving..."
                : editingSourceId
                  ? "Save changes"
                  : "Create source"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Sheet
        open={storageSheetOpen}
        onOpenChange={(open) => {
          setStorageSheetOpen(open)
          if (!open) setEditingStorageId(null)
        }}
      >
        <SheetContent className="flex w-full flex-col sm:max-w-lg">
          <SheetHeader className="border-b pb-4">
            <SheetTitle>
              {editingStorageId ? "Edit storage" : "Add local storage"}
            </SheetTitle>
            <SheetDescription>
              {editingStorageId
                ? "Update the local destination for backup artifacts."
                : "Create a local destination for backup artifacts."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto py-6">
            <FieldGroup>
              <Field>
                <FieldLabel>Name</FieldLabel>
                <Input
                  value={storageForm.name}
                  onChange={(event) =>
                    setStorageForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel>Local base path</FieldLabel>
                <Input
                  value={storageForm.path}
                  onChange={(event) =>
                    setStorageForm((current) => ({
                      ...current,
                      path: event.target.value,
                    }))
                  }
                />
              </Field>
            </FieldGroup>
          </div>
          <SheetFooter className="border-t pt-4 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="outline"
              onClick={() => setStorageSheetOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleSaveStorage}
              disabled={
                createStorageMutation.isPending ||
                updateStorageMutation.isPending
              }
            >
              <HardDriveDownload data-icon="inline-start" />
              {createStorageMutation.isPending ||
              updateStorageMutation.isPending
                ? "Saving..."
                : editingStorageId
                  ? "Save changes"
                  : "Create storage"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <BackupConfigSheet
        open={configSheetOpen}
        onOpenChange={setConfigSheetOpen}
        initialConfig={configSummary}
        initialStorageId={effectiveStorageId}
        isLoadingStorages={isLoadingStorages}
        storages={storages}
        onAddStorage={openCreateStorageSheet}
        onSave={handleSaveConfigDraft}
        isSaving={
          createConfigMutation.isPending || updateConfigMutation.isPending
        }
        canSave={Boolean(effectiveSelectedSourceId)}
      />
    </>
  )
}

function BackupConfigSheet({
  open,
  onOpenChange,
  initialConfig,
  initialStorageId,
  isLoadingStorages,
  storages,
  onAddStorage,
  onSave,
  isSaving,
  canSave,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialConfig: BackupConfigFormState
  initialStorageId: string
  isLoadingStorages: boolean
  storages: StorageModel[]
  onAddStorage: () => void
  onSave: (storageId: string, configForm: BackupConfigFormState) => void
  isSaving: boolean
  canSave: boolean
}) {
  const [storageId, setStorageId] = useState(initialStorageId)
  const [configForm, setConfigForm] = useState(initialConfig)

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        key={`${initialStorageId}:${JSON.stringify(initialConfig)}`}
        className="flex w-full flex-col sm:max-w-lg"
      >
        <SheetHeader className="border-b pb-4">
          <SheetTitle>Edit backup config</SheetTitle>
          <SheetDescription>
            Choose storage and policy for the selected backup source.
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto py-6">
          <FieldGroup>
            <Field>
              <FieldLabel>Storage</FieldLabel>
              <Select
                value={storageId}
                onValueChange={(value) => {
                  if (value === ADD_NEW_STORAGE_VALUE) {
                    onAddStorage()
                    return
                  }
                  setStorageId(value)
                }}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      isLoadingStorages
                        ? "Loading storages..."
                        : "Choose a storage"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {storages.map((item) => (
                      <SelectItem key={item.id} value={item.id}>
                        {item.name} · {item.type}
                      </SelectItem>
                    ))}
                    <SelectItem value={ADD_NEW_STORAGE_VALUE}>
                      + Add new storage
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel>Compression</FieldLabel>
                <Select
                  value={configForm.compression_type}
                  onValueChange={(value: "none" | "gzip") =>
                    setConfigForm((current) => ({
                      ...current,
                      compression_type: value,
                    }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="gzip">gzip</SelectItem>
                      <SelectItem value="none">none</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>Retention</FieldLabel>
                <Select
                  value={configForm.retention_type}
                  onValueChange={(value: "none" | "days" | "count") =>
                    setConfigForm((current) => ({
                      ...current,
                      retention_type: value,
                    }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="count">Count</SelectItem>
                      <SelectItem value="days">Days</SelectItem>
                      <SelectItem value="none">None</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel>Retention count</FieldLabel>
                <Input
                  type="number"
                  value={configForm.retention_count}
                  onChange={(event) =>
                    setConfigForm((current) => ({
                      ...current,
                      retention_count: Number(event.target.value) || 0,
                    }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel>Max retry count</FieldLabel>
                <Input
                  type="number"
                  value={configForm.max_retry_count}
                  onChange={(event) =>
                    setConfigForm((current) => ({
                      ...current,
                      max_retry_count: Number(event.target.value) || 0,
                    }))
                  }
                />
              </Field>
            </div>
            <Field
              orientation="horizontal"
              className="items-center justify-between rounded-lg border p-3"
            >
              <FieldContent>
                <FieldLabel>Retry failed backups</FieldLabel>
                <FieldDescription>
                  Keep this ready for scheduler support later.
                </FieldDescription>
              </FieldContent>
              <Switch
                checked={configForm.is_retry_if_failed}
                onCheckedChange={(checked) =>
                  setConfigForm((current) => ({
                    ...current,
                    is_retry_if_failed: checked,
                  }))
                }
              />
            </Field>
          </FieldGroup>
        </div>
        <SheetFooter className="border-t pt-4 sm:flex-row sm:justify-end">
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => onSave(storageId, configForm)}
              disabled={!canSave || !storageId || isSaving}
            >
              <Save data-icon="inline-start" />
              {isSaving ? "Saving..." : "Save backup config"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[140px_minmax(0,1fr)] sm:gap-4">
      <div className="text-muted-foreground">{label}</div>
      <div className="min-w-0 truncate font-medium text-foreground">
        {value}
      </div>
    </div>
  )
}

function buildPrefillFromResource(
  resource: ResourceModel,
  connectionInfo: any,
  envVars: Array<{ key: string; value?: string }>
) {
  const image = resource.image.toLowerCase()
  const envMap = new Map(envVars.map((item) => [item.key, item.value ?? ""]))
  const internalHost = connectionInfo?.internal_host || resource.name || ""
  const externalHost = connectionInfo?.external_host || ""
  const host = internalHost || externalHost || "127.0.0.1"
  const port =
    connectionInfo?.ports?.[0]?.host_port ||
    resource.ports?.[0]?.host_port ||
    defaultPortForImage(image)

  if (image.includes("postgres")) {
    return {
      dbType: "postgres" as const,
      host,
      internalHost,
      externalHost,
      port,
      username: envMap.get("POSTGRES_USER") || "postgres",
      password: envMap.get("POSTGRES_PASSWORD") || "",
      database: envMap.get("POSTGRES_DB") || "postgres",
      authDatabase: "",
    }
  }

  if (image.includes("mongo")) {
    return {
      dbType: "mongodb" as const,
      host,
      internalHost,
      externalHost,
      port,
      username: envMap.get("MONGO_INITDB_ROOT_USERNAME") || "root",
      password: envMap.get("MONGO_INITDB_ROOT_PASSWORD") || "",
      database: "admin",
      authDatabase: "admin",
    }
  }

  if (image.includes("mariadb")) {
    return {
      dbType: "mariadb" as const,
      host,
      internalHost,
      externalHost,
      port,
      username:
        envMap.get("MARIADB_USER") || envMap.get("MYSQL_USER") || "root",
      password:
        envMap.get("MARIADB_PASSWORD") ||
        envMap.get("MARIADB_ROOT_PASSWORD") ||
        envMap.get("MYSQL_PASSWORD") ||
        envMap.get("MYSQL_ROOT_PASSWORD") ||
        "",
      database:
        envMap.get("MARIADB_DATABASE") ||
        envMap.get("MYSQL_DATABASE") ||
        "mysql",
      authDatabase: "",
    }
  }

  return {
    dbType: "mysql" as const,
    host,
    internalHost,
    externalHost,
    port,
    username:
      envMap.get("MYSQL_USER") || envMap.get("MYSQL_ROOT_USER") || "root",
    password:
      envMap.get("MYSQL_PASSWORD") || envMap.get("MYSQL_ROOT_PASSWORD") || "",
    database: envMap.get("MYSQL_DATABASE") || "mysql",
    authDatabase: "",
  }
}

function resolveHostMode(
  source: Pick<BackupSourceModel, "host">,
  prefill: { internalHost?: string; externalHost?: string }
): HostMode {
  if (prefill.internalHost && source.host === prefill.internalHost)
    return "internal"
  if (prefill.externalHost && source.host === prefill.externalHost)
    return "external"
  return "manual"
}

function defaultPortForImage(image: string) {
  if (image.includes("postgres")) return 5432
  if (image.includes("mongo")) return 27017
  return 3306
}

function formatDate(value?: string) {
  if (!value) return "n/a"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatRelativeDate(value?: string) {
  if (!value) return "Never"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const diffMs = Date.now() - date.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  if (diffHours < 1) return "Less than an hour ago"
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  if (value < 1024 * 1024 * 1024)
    return `${(value / (1024 * 1024)).toFixed(1)} MB`
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function normalizeScheduleType(
  value: string
): BackupConfigFormState["schedule_type"] {
  if (value === "daily" || value === "hourly") return value
  return "manual_only"
}

function normalizeRetentionType(
  value: string
): BackupConfigFormState["retention_type"] {
  if (value === "days" || value === "count") return value
  return "none"
}

function normalizeEncryptionType(
  value: string
): BackupConfigFormState["encryption_type"] {
  if (value === "aes256") return value
  return "none"
}

function normalizeCompressionType(
  value: string
): BackupConfigFormState["compression_type"] {
  if (value === "none") return value
  return "gzip"
}

function normalizeBackupMethod(
  value: string
): BackupConfigFormState["backup_method"] {
  if (value === "postgres_pitr") return value
  return "logical_dump"
}

function getErrorMessage(error: unknown) {
  if (error && typeof error === "object" && "response" in error) {
    const response = (
      error as { response?: { data?: { error?: { message?: string } } } }
    ).response
    return response?.data?.error?.message || "Request failed"
  }
  if (error instanceof Error) return error.message
  return "Request failed"
}
