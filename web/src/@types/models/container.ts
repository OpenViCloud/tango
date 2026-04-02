import { z } from "zod"

export const containerPortSchema = z.object({
  ip: z.string(),
  private_port: z.number(),
  public_port: z.number(),
  type: z.string(),
})

export const containerSchema = z.object({
  id: z.string(),
  short_id: z.string(),
  name: z.string(),
  image: z.string(),
  image_id: z.string(),
  state: z.string(),
  status: z.string(),
  command: z.string(),
  ports: z.array(containerPortSchema),
  labels: z.record(z.string(), z.string()).nullable(),
})

export type ContainerModel = z.infer<typeof containerSchema>
export type ContainerPortModel = z.infer<typeof containerPortSchema>

export const containerMountSchema = z.object({
  type: z.string(),
  name: z.string(),
  source: z.string(),
  destination: z.string(),
  driver: z.string(),
  mode: z.string(),
  rw: z.boolean(),
})

export const containerDetailsSchema = z.object({
  id: z.string(),
  short_id: z.string(),
  name: z.string(),
  image: z.string(),
  image_id: z.string(),
  command: z.array(z.string()),
  created_at: z.string(),
  state: z.string(),
  status: z.string(),
  started_at: z.string(),
  finished_at: z.string(),
  exit_code: z.number(),
  error: z.string(),
  restart_count: z.number(),
  ports: z.array(containerPortSchema),
  labels: z.record(z.string(), z.string()).nullable(),
  networks: z.record(z.string(), z.string()),
  mounts: z.array(containerMountSchema),
})

export const containerStatsSchema = z.object({
  read_at: z.string(),
  cpu_percent: z.number(),
  memory_usage_bytes: z.number(),
  memory_limit_bytes: z.number(),
  memory_percent: z.number(),
  network_rx_bytes: z.number(),
  network_tx_bytes: z.number(),
  block_read_bytes: z.number(),
  block_write_bytes: z.number(),
  pids_current: z.number(),
})

export type ContainerMountModel = z.infer<typeof containerMountSchema>
export type ContainerDetailsModel = z.infer<typeof containerDetailsSchema>
export type ContainerStatsModel = z.infer<typeof containerStatsSchema>

export const imageSchema = z.object({
  id: z.string(),
  short_id: z.string(),
  tags: z.array(z.string()),
  size: z.string(),
  size_bytes: z.number(),
  created: z.string(),
  digest: z.string(),
  in_use: z.number(),
})

export type ImageModel = z.infer<typeof imageSchema>

export const createContainerSchema = z.object({
  name: z.string().optional(),
  image: z.string().min(1, "validation.required"),
  cmd: z.array(z.string()).optional(),
  env: z.record(z.string(), z.string()).optional(),
  port_bindings: z.record(z.string(), z.string()).optional(),
  volumes: z.array(z.string()).optional(),
  auto_remove: z.boolean().optional(),
})

export type CreateContainerModel = z.infer<typeof createContainerSchema>

export const pullImageSchema = z.object({
  reference: z.string().min(1, "validation.required"),
})

export type PullImageModel = z.infer<typeof pullImageSchema>
