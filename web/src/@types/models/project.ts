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
  config: z.record(z.string(), z.unknown()),
  environment_id: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
  ports: z.array(resourcePortSchema),
  node_id: z.string().nullable().optional(),
  replicas: z.number().default(1),
  memory_limit: z.number().default(0),
  cpu_limit: z.number().default(0),
  source_type: z.string().optional(),
  git_url: z.string().optional(),
  build_job_id: z.string().optional(),
  image_tag: z.string().optional(),
  connection_id: z.string().optional(),
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

export const resourceTemplateSchema = z.object({
  id: z.string(),
  name: z.string(),
  icon_url: z.string(),
  image: z.string(),
  description: z.string(),
  color: z.string(),
  abbr: z.string(),
  tags: z.array(z.string()),
  ports: z.array(
    z.object({
      host: z.string(),
      container: z.string(),
    })
  ),
  env: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    })
  ),
  type: z.string(),
  volumes: z.array(z.string()).optional(),
  cmd: z.array(z.string()).optional(),
})

export const resourceStackTemplateComponentSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  type: z.string(),
  required: z.boolean(),
  default_enabled: z.boolean(),
  ports: z.array(
    z.object({
      host: z.string(),
      container: z.string(),
    })
  ),
  env: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    })
  ),
  volumes: z.array(z.string()).optional(),
  cmd: z.array(z.string()).optional(),
  volume_files: z
    .array(z.object({ path: z.string(), content: z.string() }))
    .optional(),
})

export const resourceStackTemplateSchema = z.object({
  id: z.string(),
  name: z.string(),
  icon_url: z.string(),
  image: z.string(),
  description: z.string(),
  color: z.string(),
  abbr: z.string(),
  tags: z.array(z.string()),
  shared_env: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    })
  ),
  components: z.array(resourceStackTemplateComponentSchema),
})

export const resourceStackCreateResultSchema = z.object({
  template_id: z.string(),
  resources: z.array(resourceSchema),
})

export type ProjectModel = z.infer<typeof projectSchema>
export type EnvironmentModel = z.infer<typeof environmentSchema>
export type ResourceModel = z.infer<typeof resourceSchema>
export type ResourcePortModel = z.infer<typeof resourcePortSchema>
export type ResourceEnvVarModel = z.infer<typeof resourceEnvVarSchema>
export type ResourceTemplateModel = z.infer<typeof resourceTemplateSchema>
export type ResourceStackTemplateModel = z.infer<typeof resourceStackTemplateSchema>
export type ResourceStackCreateResultModel = z.infer<typeof resourceStackCreateResultSchema>
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
  config: z.record(z.string(), z.unknown()).optional(),
  node_id: z.string().nullable().optional(),
  replicas: z.number().min(1).default(1),
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

export const createResourceStackSchema = z.object({
  template_id: z.string().min(1, "validation.required"),
  name_prefix: z.string().min(1, "validation.required"),
  image: z.string().optional(),
  tag: z.string().optional(),
  node_id: z.string().nullable().optional(),
  shared_env_vars: z
    .array(
      z.object({
        key: z.string(),
        value: z.string(),
        is_secret: z.boolean().default(false),
      })
    )
    .optional(),
  custom_components: z.array(
    z.object({
      id: z.string(),
      type: z.string().optional(), // "service" | "job"
      cmd: z.array(z.string()),
      ports: z
        .array(
          z.object({
            host_port: z.number(),
            internal_port: z.number(),
            proto: z.string().default("tcp"),
          })
        )
        .optional(),
      volumes: z.array(z.string()).optional(),
      volume_files: z
        .array(z.object({ path: z.string(), content: z.string() }))
        .optional(),
      env: z
        .array(z.object({ key: z.string(), value: z.string(), is_secret: z.boolean() }))
        .optional(),
    })
  ),
})

export const updateProjectSchema = z.object({
  name: z.string().min(1, "validation.required"),
  description: z.string().optional(),
})

export const updateResourceSchema = z.object({
  name: z.string().min(1, "validation.required"),
  replicas: z.number().min(1).default(1),
  memory_limit: z.number().min(0).default(0),
  cpu_limit: z.number().min(0).default(0),
  config: z.record(z.string(), z.unknown()).optional(),
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
  connection_id: z.string().optional(),
  git_url: z.string().min(1, "validation.required"),
  git_branch: z.string().optional(),
  build_mode: z.enum(["auto", "dockerfile"]),
  git_token: z.string().optional(),
  image_tag: z.string().min(1, "validation.required"),
  ports: z.array(z.object({
    host_port: z.number(),
    internal_port: z.number(),
    proto: z.string(),
    label: z.string().optional(),
  })).optional(),
  env_vars: z.array(z.object({
    key: z.string(),
    value: z.string(),
    is_secret: z.boolean(),
  })).optional(),
})
export type CreateResourceFromGitModel = z.infer<typeof createResourceFromGitSchema>

export type CreateProjectModel = z.infer<typeof createProjectSchema>
export type CreateEnvironmentModel = z.infer<typeof createEnvironmentSchema>
export type CreateResourceModel = z.infer<typeof createResourceSchema>
export type CreateResourceStackModel = z.infer<typeof createResourceStackSchema>
export type UpdateProjectModel = z.infer<typeof updateProjectSchema>
export type UpdateResourceModel = z.infer<typeof updateResourceSchema>

export type ResourceDomainModel = {
  id: string
  resource_id: string
  host: string
  target_port: number
  type: "auto" | "custom"
  verified: boolean
  verified_at?: string
  created_at: string
  tls_enabled: boolean
}

export const resourceConnectionPortSchema = z.object({
  id: z.string(),
  host_port: z.number(),
  internal_port: z.number(),
  label: z.string(),
  internal_endpoint: z.string(),
  external_endpoint: z.string().optional(),
})

export const resourceConnectionInfoSchema = z.object({
  resource_id: z.string(),
  internal_host: z.string(),
  external_host: z.string().optional(),
  ports: z.array(resourceConnectionPortSchema),
})

export type ResourceConnectionPortModel = z.infer<typeof resourceConnectionPortSchema>
export type ResourceConnectionInfoModel = z.infer<typeof resourceConnectionInfoSchema>
