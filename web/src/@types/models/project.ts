import { z } from "zod"

export const resourcePortSchema = z.object({
  id: z.string(),
  host_port: z.number(),
  internal_port: z.number(),
  proto: z.string(),
  label: z.string(),
})

export const resourceSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  status: z.string(),
  image: z.string(),
  tag: z.string(),
  container_id: z.string(),
  config: z.record(z.unknown()),
  environment_id: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
  ports: z.array(resourcePortSchema),
  source_type: z.string().optional(),
  git_url: z.string().optional(),
  build_job_id: z.string().optional(),
})

export const resourceRunSchema = z.object({
  id: z.string(),
  resource_id: z.string(),
  status: z.string(),
  logs: z.string(),
  error_msg: z.string().optional(),
  started_at: z.string().optional(),
  finished_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})

export const resourceLogsSchema = z.object({
  resource_id: z.string(),
  container_id: z.string(),
  status: z.string(),
  lines: z.array(z.string()),
})

export const environmentSchema = z.object({
  id: z.string(),
  name: z.string(),
  project_id: z.string(),
  created_at: z.string(),
  resources: z.array(resourceSchema),
})

export const projectSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  created_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
  environments: z.array(environmentSchema),
})

export const resourceEnvVarSchema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(),
  is_secret: z.boolean(),
})

export type ProjectModel = z.infer<typeof projectSchema>
export type EnvironmentModel = z.infer<typeof environmentSchema>
export type ResourceModel = z.infer<typeof resourceSchema>
export type ResourcePortModel = z.infer<typeof resourcePortSchema>
export type ResourceEnvVarModel = z.infer<typeof resourceEnvVarSchema>
export type ResourceRunModel = z.infer<typeof resourceRunSchema>
export type ResourceLogsModel = z.infer<typeof resourceLogsSchema>

// ── Create schemas ─────────────────────────────────────────────────────────────

export const createProjectSchema = z.object({
  name: z.string().min(1, "validation.required"),
  description: z.string().optional(),
})

export const createEnvironmentSchema = z.object({
  name: z.string().min(1, "validation.required"),
})

export const createResourceSchema = z.object({
  name: z.string().min(1, "validation.required"),
  type: z.string().default("db"),
  image: z.string().min(1, "validation.required"),
  tag: z.string().default("latest"),
  config: z.record(z.unknown()).optional(),
  ports: z
    .array(
      z.object({
        host_port: z.number(),
        internal_port: z.number(),
        proto: z.string().default("tcp"),
        label: z.string().optional(),
      })
    )
    .optional(),
  env_vars: z
    .array(
      z.object({
        key: z.string(),
        value: z.string(),
        is_secret: z.boolean().default(false),
      })
    )
    .optional(),
})

export const updateProjectSchema = z.object({
  name: z.string().min(1, "validation.required"),
  description: z.string().optional(),
})

export const updateResourceSchema = z.object({
  name: z.string().min(1, "validation.required"),
  ports: z
    .array(
      z.object({
        host_port: z.number(),
        internal_port: z.number(),
        proto: z.string().default("tcp"),
        label: z.string().optional(),
      })
    )
    .optional(),
})

export const createResourceFromGitSchema = z.object({
  name: z.string().min(1, "validation.required"),
  git_url: z.string().min(1, "validation.required"),
  git_branch: z.string().optional(),
  build_mode: z.enum(["auto", "dockerfile"]).default("auto"),
  git_token: z.string().optional(),
  image_tag: z.string().min(1, "validation.required"),
  ports: z.array(z.object({
    host_port: z.number(),
    internal_port: z.number(),
    proto: z.string().default("tcp"),
    label: z.string().optional(),
  })).optional(),
  env_vars: z.array(z.object({
    key: z.string(),
    value: z.string(),
    is_secret: z.boolean().default(false),
  })).optional(),
})
export type CreateResourceFromGitModel = z.infer<typeof createResourceFromGitSchema>

export type CreateProjectModel = z.infer<typeof createProjectSchema>
export type CreateEnvironmentModel = z.infer<typeof createEnvironmentSchema>
export type CreateResourceModel = z.infer<typeof createResourceSchema>
export type UpdateProjectModel = z.infer<typeof updateProjectSchema>
export type UpdateResourceModel = z.infer<typeof updateResourceSchema>
