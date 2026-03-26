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
