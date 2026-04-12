import { z } from "zod"

// ── Server ──────────────────────────────────────────────────────────────────

export const serverStatusSchema = z.enum(["pending", "connected", "error"])

export const serverSchema = z.object({
  id: z.string(),
  name: z.string(),
  public_ip: z.string(),
  private_ip: z.string(),
  ssh_user: z.string(),
  ssh_port: z.number(),
  status: serverStatusSchema,
  error_msg: z.string().optional(),
  last_ping_at: z.string().nullable().optional(),
  created_at: z.string(),
})

export type ServerModel = z.infer<typeof serverSchema>

export const createServerSchema = z.object({
  name: z.string().min(1, "Name is required"),
  public_ip: z.string().min(1, "Public IP is required"),
  private_ip: z.string().optional(),
  ssh_user: z.string().optional(),
  ssh_port: z.number().optional(),
})

export type CreateServerModel = z.infer<typeof createServerSchema>

// ── Cluster ─────────────────────────────────────────────────────────────────

export const clusterStatusSchema = z.enum([
  "pending",
  "provisioning",
  "ready",
  "error",
])

export const clusterNodeRoleSchema = z.enum(["master", "worker"])

export const clusterNodeSchema = z.object({
  server_id: z.string(),
  role: clusterNodeRoleSchema,
})

export type ClusterNodeModel = z.infer<typeof clusterNodeSchema>

export const clusterSchema = z.object({
  id: z.string(),
  name: z.string(),
  status: clusterStatusSchema,
  error_msg: z.string().optional(),
  k8s_version: z.string(),
  pod_cidr: z.string(),
  nodes: z.array(clusterNodeSchema),
  created_at: z.string(),
})

export type ClusterModel = z.infer<typeof clusterSchema>

export const bootstrapClusterSchema = z.object({
  name: z.string().min(1, "Name is required"),
  k8s_version: z.string().optional(),
  pod_cidr: z.string().optional(),
  nodes: z
    .array(clusterNodeSchema)
    .min(1, "At least one node is required"),
})

export type BootstrapClusterModel = z.infer<typeof bootstrapClusterSchema>

// ── Kubernetes resources ─────────────────────────────────────────────────────

export type KubeNamespace = {
  name: string
  status: string
}

export type KubePod = {
  name: string
  namespace: string
  status: string
  node_name: string
  pod_ip: string
}

export type KubeServicePort = {
  name: string
  port: number
  target_port: string
  node_port?: number
  protocol: string
}

export type KubeService = {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  ports: KubeServicePort[]
}

export type KubePersistentVolume = {
  name: string
  capacity: string
  access_modes: string
  reclaim_policy: string
  status: string
  storage_class_name: string
}

export type KubePersistentVolumeClaim = {
  name: string
  namespace: string
  status: string
  volume_name: string
  capacity: string
  access_modes: string
  storage_class_name: string
}
