import { useEffect, useRef, useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import {
  NetworkIcon,
  ArrowLeftIcon,
  CloudIcon,
  DownloadIcon,
  ServerIcon,
  FileTextIcon,
  BoxIcon,
  GlobeIcon,
  DatabaseIcon,
  RefreshCwIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"
import { useForm, type Resolver, type SubmitHandler } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useQuery } from "@tanstack/react-query"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import {
  useGetCluster,
  useClusterLogs,
  useKubePods,
  useKubeServices,
  useKubePersistentVolumes,
  useKubePersistentVolumeClaims,
  useCreateKubePod,
  useCreateKubeService,
  useDeleteKubePod,
  useDeleteKubeService,
  useImportClusterTunnel,
} from "@/hooks/api/use-cluster"
import { useGetCloudflareConnections } from "@/hooks/api/use-cloudflare"
import { useGetServerList } from "@/hooks/api/use-server"
import { clusterService } from "@/services/api/cluster-service"
import { ClusterStatusBadge } from "./clusters-page"
import type {
  ClusterModel,
  KubePod,
  KubeService,
  KubePersistentVolume,
  KubePersistentVolumeClaim,
} from "@/@types/models/server"
import type { CloudflareConnectionModel } from "@/@types/models/cloudflare"

interface ClusterDetailPageProps {
  clusterId: string
}

export function ClusterDetailPage({ clusterId }: ClusterDetailPageProps) {
  const navigate = useNavigate()
  const { data: cluster, isLoading } = useGetCluster(clusterId)
  const { data: servers } = useGetServerList()
  const [importTunnelOpen, setImportTunnelOpen] = useState(false)
  const isProvisioning = cluster?.status === "provisioning" || cluster?.status === "pending"
  const { lines, done, connected } = useClusterLogs(isProvisioning ? clusterId : null)

  const serverMap = Object.fromEntries((servers ?? []).map((s) => [s.id, s]))

  const downloadKubeconfig = async () => {
    try {
      const blob = await clusterService.downloadKubeconfig(clusterId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `kubeconfig-${cluster?.name ?? clusterId}.yaml`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      toast.error("Kubeconfig not available yet.")
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }

  if (!cluster) {
    return <p className="text-sm text-muted-foreground">Cluster not found.</p>
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<NetworkIcon className="size-6" />}
        title={cluster.name}
        titleMeta={cluster.k8s_version}
        description={`Pod CIDR: ${cluster.pod_cidr}`}
        headerRight={
          <div className="flex gap-2">
            {cluster.status === "ready" && (
              <Button size="sm" variant="outline" onClick={() => setImportTunnelOpen(true)}>
                <CloudIcon className="size-4" />
                Import Existing Tunnel
              </Button>
            )}
            {cluster.status === "ready" && (
              <Button size="sm" variant="outline" onClick={() => void downloadKubeconfig()}>
                <DownloadIcon className="size-4" />
                kubeconfig
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => void navigate({ to: "/clusters" })}
            >
              <ArrowLeftIcon className="size-4" />
              Back
            </Button>
          </div>
        }
      />

      <SectionCard title="Status" icon={<NetworkIcon className="size-5" />}>
        <div className="flex items-center gap-3">
          <ClusterStatusBadge status={cluster.status} />
          {cluster.error_msg && (
            <p className="text-sm text-destructive">{cluster.error_msg}</p>
          )}
        </div>
      </SectionCard>

      <SectionCard title="Nodes" icon={<ServerIcon className="size-5" />}>
        <div className="overflow-x-auto rounded-xl border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
                <th className="px-4 py-3">Server</th>
                <th className="px-4 py-3">IP</th>
                <th className="px-4 py-3">Role</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {cluster.nodes.map((node) => {
                const server = serverMap[node.server_id]
                return (
                  <tr key={node.server_id} className="hover:bg-muted/20">
                    <td className="px-4 py-3 font-medium">{server?.name ?? node.server_id}</td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{server?.public_ip ?? "—"}</td>
                    <td className="px-4 py-3">
                      <Badge variant={node.role === "master" ? "default" : "secondary"} className="capitalize">
                        {node.role}
                      </Badge>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </SectionCard>

      {cluster.status === "ready" && (
        <KubeResourcesSection clusterId={clusterId} />
      )}

      <InventorySection cluster={cluster} />

      {(isProvisioning || lines.length > 0) && (
        <SectionCard
          title="Provisioning Logs"
          icon={<NetworkIcon className="size-5" />}
          description={connected ? "Live streaming..." : done ? "Provisioning complete." : ""}
          headerRight={connected ? <Badge variant="secondary" className="animate-pulse">Live</Badge> : null}
        >
          <LogViewer lines={lines} />
        </SectionCard>
      )}

      <ImportTunnelDialog
        clusterId={clusterId}
        open={importTunnelOpen}
        onClose={() => setImportTunnelOpen(false)}
      />
    </div>
  )
}

// ── K8s Resources Section ────────────────────────────────────────────────────

function KubeResourcesSection({ clusterId }: { clusterId: string }) {
  const [podDialogOpen, setPodDialogOpen] = useState(false)
  const [serviceDialogOpen, setServiceDialogOpen] = useState(false)

  const { data: pods, isLoading: podsLoading, refetch: refetchPods } = useKubePods(clusterId)
  const { data: services, isLoading: servicesLoading, refetch: refetchServices } = useKubeServices(clusterId)
  const { data: volumes, isLoading: volumesLoading, refetch: refetchVolumes } = useKubePersistentVolumes(clusterId)
  const { data: pvcs, isLoading: pvcsLoading, refetch: refetchPVCs } = useKubePersistentVolumeClaims(clusterId)

  const { mutate: deletePod } = useDeleteKubePod(clusterId)
  const { mutate: deleteService } = useDeleteKubeService(clusterId)

  return (
    <SectionCard title="Kubernetes Resources" icon={<BoxIcon className="size-5" />}>
      <Tabs defaultValue="pods">
        <TabsList>
          <TabsTrigger value="pods">
            <BoxIcon className="size-3.5" />
            Pods
            {pods && <span className="ml-1 rounded-full bg-muted px-1.5 py-0.5 text-xs font-mono">{pods.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="services">
            <GlobeIcon className="size-3.5" />
            Services
            {services && <span className="ml-1 rounded-full bg-muted px-1.5 py-0.5 text-xs font-mono">{services.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="volumes">
            <DatabaseIcon className="size-3.5" />
            Volumes
            {volumes && <span className="ml-1 rounded-full bg-muted px-1.5 py-0.5 text-xs font-mono">{volumes.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="pvcs">
            <DatabaseIcon className="size-3.5" />
            PVCs
            {pvcs && <span className="ml-1 rounded-full bg-muted px-1.5 py-0.5 text-xs font-mono">{pvcs.length}</span>}
          </TabsTrigger>
        </TabsList>

        {/* Pods */}
        <TabsContent value="pods">
          <div className="mt-3 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">All namespaces</p>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => setPodDialogOpen(true)}>
                <PlusIcon className="size-3.5" /> Add Pod
              </Button>
              <Button size="sm" variant="ghost" onClick={() => void refetchPods()}>
                <RefreshCwIcon className="size-3.5" /> Refresh
              </Button>
            </div>
          </div>
          <KubeTable loading={podsLoading} empty={!pods?.length} emptyText="No pods found"
            columns={["Name", "Namespace", "Status", "Node", "IP", ""]}>
            {pods?.map((pod) => (
              <KubePodRow
                key={`${pod.namespace}/${pod.name}`}
                pod={pod}
                onDelete={() => deletePod({ namespace: pod.namespace, name: pod.name })}
              />
            ))}
          </KubeTable>
        </TabsContent>

        {/* Services */}
        <TabsContent value="services">
          <div className="mt-3 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">All namespaces</p>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => setServiceDialogOpen(true)}>
                <PlusIcon className="size-3.5" /> Add Service
              </Button>
              <Button size="sm" variant="ghost" onClick={() => void refetchServices()}>
                <RefreshCwIcon className="size-3.5" /> Refresh
              </Button>
            </div>
          </div>
          <KubeTable loading={servicesLoading} empty={!services?.length} emptyText="No services found"
            columns={["Name", "Namespace", "Type", "Cluster IP", "Ports", ""]}>
            {services?.map((svc) => (
              <KubeServiceRow
                key={`${svc.namespace}/${svc.name}`}
                service={svc}
                onDelete={() => deleteService({ namespace: svc.namespace, name: svc.name })}
              />
            ))}
          </KubeTable>
        </TabsContent>

        {/* Volumes */}
        <TabsContent value="volumes">
          <div className="mt-3 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">Cluster-wide</p>
            <Button size="sm" variant="ghost" onClick={() => void refetchVolumes()}>
              <RefreshCwIcon className="size-3.5" /> Refresh
            </Button>
          </div>
          <KubeTable loading={volumesLoading} empty={!volumes?.length} emptyText="No persistent volumes found"
            columns={["Name", "Capacity", "Access", "Reclaim", "Status", "StorageClass"]}>
            {volumes?.map((pv) => <KubePVRow key={pv.name} pv={pv} />)}
          </KubeTable>
        </TabsContent>

        {/* PVCs */}
        <TabsContent value="pvcs">
          <div className="mt-3 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">All namespaces</p>
            <Button size="sm" variant="ghost" onClick={() => void refetchPVCs()}>
              <RefreshCwIcon className="size-3.5" /> Refresh
            </Button>
          </div>
          <KubeTable loading={pvcsLoading} empty={!pvcs?.length} emptyText="No persistent volume claims found"
            columns={["Name", "Namespace", "Status", "Volume", "Capacity", "StorageClass"]}>
            {pvcs?.map((pvc) => (
              <KubePVCRow key={`${pvc.namespace}/${pvc.name}`} pvc={pvc} />
            ))}
          </KubeTable>
        </TabsContent>
      </Tabs>

      <CreatePodDialog clusterId={clusterId} open={podDialogOpen} onClose={() => setPodDialogOpen(false)} />
      <CreateServiceDialog clusterId={clusterId} open={serviceDialogOpen} onClose={() => setServiceDialogOpen(false)} />
    </SectionCard>
  )
}

const importTunnelSchema = z.object({
  connection_id: z.string().optional(),
  tunnel_id: z.string().min(1, "Tunnel ID is required"),
  tunnel_token: z.string().min(1, "Tunnel token is required"),
  namespace: z.string().min(1, "Namespace is required"),
  overwrite: z.boolean().optional(),
})

type ImportTunnelForm = z.infer<typeof importTunnelSchema>

function ImportTunnelDialog({
  clusterId,
  open,
  onClose,
}: {
  clusterId: string
  open: boolean
  onClose: () => void
}) {
  const importTunnel = useImportClusterTunnel(clusterId)
  const { data: connections = [], isLoading } = useGetCloudflareConnections()
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ImportTunnelForm>({
    resolver: zodResolver(importTunnelSchema) as Resolver<ImportTunnelForm>,
    defaultValues: {
      connection_id: "",
      tunnel_id: "",
      tunnel_token: "",
      namespace: "default",
      overwrite: false,
    },
  })

  useEffect(() => {
    if (!open) {
      reset({
        connection_id: "",
        tunnel_id: "",
        tunnel_token: "",
        namespace: "default",
        overwrite: false,
      })
    }
  }, [open, reset])

  useEffect(() => {
    if (open && !watch("connection_id")) {
      setValue("connection_id", "")
    }
  }, [open, setValue, watch])

  const selectedConnectionId = watch("connection_id")

  const onSubmit: SubmitHandler<ImportTunnelForm> = (data) => {
    importTunnel.mutate(data, {
      onSuccess: () => {
        reset({
          connection_id: "",
          tunnel_id: "",
          tunnel_token: "",
          namespace: "default",
          overwrite: false,
        })
        onClose()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose() }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Import Existing Tunnel</DialogTitle>
        </DialogHeader>
        <form onSubmit={(event) => void handleSubmit(onSubmit)(event)} className="flex flex-col gap-4">
          <FieldGroup>
            <Field>
              <FieldLabel>Cloudflare Connection</FieldLabel>
              <Select
                value={selectedConnectionId || "__none__"}
                onValueChange={(value) =>
                  setValue("connection_id", value === "__none__" ? "" : value, {
                    shouldValidate: true,
                  })}
                disabled={isLoading || importTunnel.isPending}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Optional" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">No connection (runtime-only)</SelectItem>
                  {connections.map((connection: CloudflareConnectionModel) => (
                    <SelectItem key={connection.id} value={connection.id}>
                      {connection.display_name} ({connection.zone_id})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldDescription>
                Chọn connection nếu bạn muốn app quản lý DNS/expose sau này. Bỏ trống nếu chỉ cần deploy `cloudflared` với tunnel có sẵn.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel>Tunnel ID</FieldLabel>
              <Input
                className="font-mono"
                placeholder="cloudflare-tunnel-uuid"
                {...register("tunnel_id")}
                disabled={importTunnel.isPending}
              />
              {errors.tunnel_id && <p className="text-xs text-destructive">{errors.tunnel_id.message}</p>}
            </Field>

            <Field>
              <FieldLabel>Tunnel Token</FieldLabel>
              <Input
                type="password"
                placeholder="Paste existing tunnel token"
                {...register("tunnel_token")}
                disabled={importTunnel.isPending}
              />
              <FieldDescription>
                Token này sẽ được mã hóa trước khi lưu và dùng để deploy `cloudflared` vào cluster.
              </FieldDescription>
              {errors.tunnel_token && <p className="text-xs text-destructive">{errors.tunnel_token.message}</p>}
            </Field>

            <Field>
              <FieldLabel>Namespace</FieldLabel>
              <Input
                placeholder="default"
                {...register("namespace")}
                disabled={importTunnel.isPending}
              />
              {errors.namespace && <p className="text-xs text-destructive">{errors.namespace.message}</p>}
            </Field>

            <Field orientation="horizontal" className="rounded-xl border px-4 py-3">
              <FieldLabel className="gap-3 border-none p-0">
                <FieldDescription className="mt-0 text-xs">
                  Overwrite existing tunnel state on this cluster if one already exists.
                </FieldDescription>
              </FieldLabel>
              <Switch
                checked={watch("overwrite") ?? false}
                onCheckedChange={(checked) =>
                  setValue("overwrite", checked, { shouldValidate: true })
                }
                disabled={importTunnel.isPending}
              />
            </Field>
          </FieldGroup>

          <DialogFooter showCloseButton>
            <Button
              type="submit"
              disabled={importTunnel.isPending}
            >
              {importTunnel.isPending ? "Importing..." : "Import Tunnel"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ── Create Pod Dialog ─────────────────────────────────────────────────────────

const createPodSchema = z.object({
  name: z.string().min(1, "Name is required").regex(/^[a-z0-9-]+$/, "Lowercase, numbers, hyphens only"),
  image: z.string().min(1, "Image is required"),
  namespace: z.string().min(1, "Namespace is required"),
  labelKey: z.string().optional(),
  labelValue: z.string().optional(),
})
type CreatePodForm = z.infer<typeof createPodSchema>

function CreatePodDialog({ clusterId, open, onClose }: { clusterId: string; open: boolean; onClose: () => void }) {
  const { mutate: createPod, isPending } = useCreateKubePod(clusterId)
  const { register, handleSubmit, reset, formState: { errors } } = useForm<CreatePodForm>({
    resolver: zodResolver(createPodSchema),
    defaultValues: { namespace: "default", labelKey: "app" },
  })

  const onSubmit = (data: CreatePodForm) => {
    const labels: Record<string, string> = {}
    if (data.labelKey && data.labelValue) labels[data.labelKey] = data.labelValue

    createPod(
      { namespace: data.namespace, name: data.name, image: data.image, labels },
      { onSuccess: () => { reset(); onClose() } },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) { reset(); onClose() } }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader><DialogTitle>Create Pod</DialogTitle></DialogHeader>
        <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pod-name">Name</Label>
            <Input id="pod-name" placeholder="my-pod" {...register("name")} />
            {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pod-image">Image</Label>
            <Input id="pod-image" placeholder="nginx:latest" {...register("image")} />
            {errors.image && <p className="text-xs text-destructive">{errors.image.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pod-namespace">Namespace</Label>
            <Input id="pod-namespace" placeholder="default" {...register("namespace")} />
            {errors.namespace && <p className="text-xs text-destructive">{errors.namespace.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Label <span className="text-muted-foreground font-normal">(dùng cho Service selector)</span></Label>
            <div className="flex gap-2">
              <Input placeholder="app" {...register("labelKey")} />
              <span className="flex items-center text-muted-foreground">=</span>
              <Input placeholder="nginx" {...register("labelValue")} />
            </div>
          </div>

          <DialogFooter showCloseButton>
            <Button type="submit" disabled={isPending}>{isPending ? "Creating..." : "Create Pod"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ── Create Service Dialog ─────────────────────────────────────────────────────

const createServiceSchema = z.object({
  name: z.string().min(1, "Name is required").regex(/^[a-z0-9-]+$/, "Lowercase, numbers, hyphens only"),
  namespace: z.string().min(1, "Namespace is required"),
  type: z.enum(["ClusterIP", "NodePort", "LoadBalancer"]),
  selectorKey: z.string().min(1, "Selector key is required"),
  selectorValue: z.string().min(1, "Selector value is required"),
  port: z.coerce.number().int().min(1).max(65535),
  targetPort: z.string().min(1, "Target port is required"),
  protocol: z.enum(["TCP", "UDP"]),
})
type CreateServiceForm = z.infer<typeof createServiceSchema>

function CreateServiceDialog({ clusterId, open, onClose }: { clusterId: string; open: boolean; onClose: () => void }) {
  const { mutate: createService, isPending } = useCreateKubeService(clusterId)
  const { register, handleSubmit, reset, setValue, watch, formState: { errors } } = useForm<CreateServiceForm>({
    resolver: zodResolver(createServiceSchema) as Resolver<CreateServiceForm>,
    defaultValues: { namespace: "default", type: "ClusterIP", protocol: "TCP" },
  })

  const svcType = watch("type")
  const protocol = watch("protocol")

  const onSubmit: SubmitHandler<CreateServiceForm> = (data) => {
    createService(
      {
        namespace: data.namespace,
        name: data.name,
        type: data.type,
        selector: { [data.selectorKey]: data.selectorValue },
        ports: [{ name: "port-0", port: data.port, target_port: data.targetPort, protocol: data.protocol }],
      },
      { onSuccess: () => { reset(); onClose() } },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) { reset(); onClose() } }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader><DialogTitle>Create Service</DialogTitle></DialogHeader>
        <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="svc-name">Name</Label>
            <Input id="svc-name" placeholder="my-service" {...register("name")} />
            {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="svc-namespace">Namespace</Label>
            <Input id="svc-namespace" placeholder="default" {...register("namespace")} />
            {errors.namespace && <p className="text-xs text-destructive">{errors.namespace.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Type</Label>
            <Select value={svcType} onValueChange={(v) => setValue("type", v as CreateServiceForm["type"])}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="ClusterIP">ClusterIP</SelectItem>
                <SelectItem value="NodePort">NodePort</SelectItem>
                <SelectItem value="LoadBalancer">LoadBalancer</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Selector <span className="text-muted-foreground font-normal">(phải khớp label của pod)</span></Label>
            <div className="flex gap-2">
              <Input placeholder="app" {...register("selectorKey")} />
              <span className="flex items-center text-muted-foreground">=</span>
              <Input placeholder="nginx" {...register("selectorValue")} />
            </div>
            {(errors.selectorKey ?? errors.selectorValue) && (
              <p className="text-xs text-destructive">{errors.selectorKey?.message ?? errors.selectorValue?.message}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="svc-port">Port</Label>
              <Input id="svc-port" type="number" placeholder="80" {...register("port")} />
              {errors.port && <p className="text-xs text-destructive">{errors.port.message}</p>}
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="svc-target-port">Target Port</Label>
              <Input id="svc-target-port" placeholder="80" {...register("targetPort")} />
              {errors.targetPort && <p className="text-xs text-destructive">{errors.targetPort.message}</p>}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Protocol</Label>
            <Select value={protocol} onValueChange={(v) => setValue("protocol", v as "TCP" | "UDP")}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="TCP">TCP</SelectItem>
                <SelectItem value="UDP">UDP</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <DialogFooter showCloseButton>
            <Button type="submit" disabled={isPending}>{isPending ? "Creating..." : "Create Service"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ── Shared table shell ────────────────────────────────────────────────────────

function KubeTable({
  loading, empty, emptyText, columns, children,
}: {
  loading: boolean; empty: boolean; emptyText: string; columns: string[]; children: React.ReactNode
}) {
  if (loading) {
    return (
      <div className="mt-2 flex flex-col gap-2">
        {[...Array(3)].map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
      </div>
    )
  }
  if (empty) {
    return (
      <div className="mt-2 flex h-24 items-center justify-center rounded-xl border border-dashed">
        <p className="text-xs text-muted-foreground">{emptyText}</p>
      </div>
    )
  }
  return (
    <div className="mt-2 overflow-x-auto rounded-xl border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
            {columns.map((col) => <th key={col} className="px-4 py-3">{col}</th>)}
          </tr>
        </thead>
        <tbody className="divide-y">{children}</tbody>
      </table>
    </div>
  )
}

// ── Delete confirm ────────────────────────────────────────────────────────────

function DeleteConfirm({
  open, name, kind, onConfirm, onCancel, isPending,
}: {
  open: boolean; name: string; kind: string; onConfirm: () => void; onCancel: () => void; isPending: boolean
}) {
  return (
    <AlertDialog open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {kind}</AlertDialogTitle>
          <AlertDialogDescription>
            Xóa <span className="font-mono font-semibold">{name}</span>? Hành động này không thể hoàn tác.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>Hủy</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={isPending}>
            {isPending ? "Deleting..." : "Xóa"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

// ── Row components ────────────────────────────────────────────────────────────

function podStatusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  if (status === "Running") return "default"
  if (status === "Pending") return "secondary"
  if (status === "Failed") return "destructive"
  return "outline"
}

function KubePodRow({ pod, onDelete }: { pod: KubePod; onDelete: () => void }) {
  const [confirm, setConfirm] = useState(false)

  return (
    <>
      <tr className="hover:bg-muted/20">
        <td className="px-4 py-3 font-medium font-mono text-xs">{pod.name}</td>
        <td className="px-4 py-3 text-muted-foreground">{pod.namespace}</td>
        <td className="px-4 py-3">
          <Badge variant={podStatusVariant(pod.status)} className="capitalize">{pod.status || "Unknown"}</Badge>
        </td>
        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pod.node_name || "—"}</td>
        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pod.pod_ip || "—"}</td>
        <td className="px-4 py-3 text-right">
          <Button size="icon-sm" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setConfirm(true)}>
            <Trash2Icon className="size-3.5" />
          </Button>
        </td>
      </tr>
      <DeleteConfirm
        open={confirm}
        name={pod.name}
        kind="Pod"
        isPending={false}
        onConfirm={() => { onDelete(); setConfirm(false) }}
        onCancel={() => setConfirm(false)}
      />
    </>
  )
}

function KubeServiceRow({ service, onDelete }: { service: KubeService; onDelete: () => void }) {
  const [confirm, setConfirm] = useState(false)
  // isPending not tracked per-row; toast feedback is sufficient

  const portsSummary = (service.ports ?? [])
    .map((p) => `${p.port}${p.node_port ? `:${p.node_port}` : ""}/${p.protocol}`)
    .join(", ")

  return (
    <>
      <tr className="hover:bg-muted/20">
        <td className="px-4 py-3 font-medium font-mono text-xs">{service.name}</td>
        <td className="px-4 py-3 text-muted-foreground">{service.namespace}</td>
        <td className="px-4 py-3"><Badge variant="outline">{service.type}</Badge></td>
        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{service.cluster_ip || "—"}</td>
        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{portsSummary || "—"}</td>
        <td className="px-4 py-3 text-right">
          <Button size="icon-sm" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setConfirm(true)}>
            <Trash2Icon className="size-3.5" />
          </Button>
        </td>
      </tr>
      <DeleteConfirm
        open={confirm}
        name={service.name}
        kind="Service"
        isPending={false}
        onConfirm={() => { onDelete(); setConfirm(false) }}
        onCancel={() => setConfirm(false)}
      />
    </>
  )
}

function pvStatusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  if (status === "Bound") return "default"
  if (status === "Available") return "secondary"
  if (status === "Released") return "outline"
  if (status === "Failed") return "destructive"
  return "outline"
}

function KubePVRow({ pv }: { pv: KubePersistentVolume }) {
  return (
    <tr className="hover:bg-muted/20">
      <td className="px-4 py-3 font-medium font-mono text-xs">{pv.name}</td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pv.capacity || "—"}</td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pv.access_modes || "—"}</td>
      <td className="px-4 py-3 text-muted-foreground">{pv.reclaim_policy || "—"}</td>
      <td className="px-4 py-3"><Badge variant={pvStatusVariant(pv.status)}>{pv.status || "Unknown"}</Badge></td>
      <td className="px-4 py-3 text-muted-foreground">{pv.storage_class_name || "—"}</td>
    </tr>
  )
}

function KubePVCRow({ pvc }: { pvc: KubePersistentVolumeClaim }) {
  return (
    <tr className="hover:bg-muted/20">
      <td className="px-4 py-3 font-medium font-mono text-xs">{pvc.name}</td>
      <td className="px-4 py-3 text-muted-foreground">{pvc.namespace}</td>
      <td className="px-4 py-3"><Badge variant={pvStatusVariant(pvc.status)}>{pvc.status || "Unknown"}</Badge></td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pvc.volume_name || "—"}</td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{pvc.capacity || "—"}</td>
      <td className="px-4 py-3 text-muted-foreground">{pvc.storage_class_name || "—"}</td>
    </tr>
  )
}

// ── Inventory ─────────────────────────────────────────────────────────────────

function InventorySection({ cluster }: { cluster: ClusterModel }) {
  const { data, isLoading } = useQuery({
    queryKey: ["clusters", cluster.id, "inventory"],
    queryFn: () => clusterService.inventoryPreview(cluster.id, cluster.nodes),
    staleTime: Infinity,
  })

  return (
    <SectionCard title="Ansible Inventory" icon={<FileTextIcon className="size-5" />} description="inventory.ini used for provisioning">
      {isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : (
        <pre className="overflow-x-auto rounded-lg border bg-zinc-950 p-4 font-mono text-xs text-zinc-100 whitespace-pre">
          {data?.inventory ?? "—"}
        </pre>
      )}
    </SectionCard>
  )
}

// ── Log viewer ────────────────────────────────────────────────────────────────

function LogViewer({ lines }: { lines: string[] }) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [lines])

  if (!lines.length) {
    return (
      <div className="flex h-32 items-center justify-center rounded-lg border bg-muted/20">
        <p className="text-xs text-muted-foreground animate-pulse">Waiting for logs...</p>
      </div>
    )
  }

  return (
    <div className="max-h-125 overflow-y-auto rounded-lg border bg-zinc-950 p-4 font-mono text-xs text-zinc-100">
      {lines.map((line, i) => (
        <div key={i} className={
          !line ? "" :
          line.startsWith("[ERROR]") ? "text-red-400" :
          line.startsWith(">>>") ? "text-yellow-300 font-semibold mt-2" :
          line.startsWith("===") ? "text-green-400 font-semibold" : ""
        }>
          {line}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
