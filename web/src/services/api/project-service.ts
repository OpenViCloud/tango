import type {
  ProjectModel,
  CreateProjectModel,
  UpdateProjectModel,
  CreateEnvironmentModel,
  EnvironmentModel,
  ResourceModel,
  CreateResourceModel,
  UpdateResourceModel,
  ResourceEnvVarModel,
  ResourceLogsModel,
  ResourceConnectionInfoModel,
  ResourceRunModel,
  ResourceTemplateModel,
  ResourceStackTemplateModel,
  ResourceStackCreateResultModel,
} from "@/@types/models"
import type {
  CreateResourceFromGitModel,
  CreateResourceStackModel,
  ResourceDomainModel,
} from "@/@types/models/project"
import type { ApiResponse } from "@/@types/models/common"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const BASE = "/projects"

export type CheckRepoResponse = {
  available: boolean
  default_branch?: string
  branch_exists: boolean
  error?: string
}

export type ResourceRuntimeReconcileResponse = {
  checked: number
  updated: number
  running: number
  stopped: number
  errored: number
  missing_containers: number
}

export const projectService = {
  // ── Projects ────────────────────────────────────────────────────────────────

  listProjects: () =>
    api
      .get<ApiResponse<ProjectModel[]>>(BASE)
      .then((res) => unwrapApiResponse(res.data)),

  getProject: (id: string) =>
    api
      .get<ApiResponse<ProjectModel>>(`${BASE}/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  createProject: (payload: CreateProjectModel) =>
    api
      .post<ApiResponse<ProjectModel>>(BASE, payload)
      .then((res) => unwrapApiResponse(res.data)),

  updateProject: (id: string, payload: UpdateProjectModel) =>
    api
      .put<ApiResponse<ProjectModel>>(`${BASE}/${id}`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  deleteProject: (id: string) => api.delete(`${BASE}/${id}`),

  // ── Environments ────────────────────────────────────────────────────────────

  createEnvironment: (projectId: string, payload: CreateEnvironmentModel) =>
    api
      .post<ApiResponse<EnvironmentModel>>(`${BASE}/${projectId}/environments`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  deleteEnvironment: (envId: string) => api.delete(`/environments/${envId}`),

  forkEnvironment: (envId: string, payload: { name: string }) =>
    api
      .post<ApiResponse<EnvironmentModel>>(`/environments/${envId}/fork`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  // ── Resources ───────────────────────────────────────────────────────────────

  listResources: (envId: string) =>
    api
      .get<ApiResponse<ResourceModel[]>>(`/environments/${envId}/resources`)
      .then((res) => unwrapApiResponse(res.data)),

  listResourceTemplates: () =>
    api
      .get<ApiResponse<ResourceTemplateModel[]>>(`/resource-templates`)
      .then((res) => unwrapApiResponse(res.data)),

  listResourceStackTemplates: () =>
    api
      .get<ApiResponse<ResourceStackTemplateModel[]>>(`/resource-stack-templates`)
      .then((res) => unwrapApiResponse(res.data)),

  createResource: (envId: string, payload: CreateResourceModel) =>
    api
      .post<ApiResponse<ResourceModel>>(`/environments/${envId}/resources`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  createResourceStack: (envId: string, payload: CreateResourceStackModel) =>
    api
      .post<ApiResponse<ResourceStackCreateResultModel>>(`/environments/${envId}/resource-stacks`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  getResource: (resourceId: string) =>
    api
      .get<ApiResponse<ResourceModel>>(`/resources/${resourceId}`)
      .then((res) => unwrapApiResponse(res.data)),

  getResourceConnectionInfo: (resourceId: string) =>
    api
      .get<ApiResponse<ResourceConnectionInfoModel>>(`/resources/${resourceId}/connection-info`)
      .then((res) => unwrapApiResponse(res.data)),

  updateResource: (resourceId: string, payload: UpdateResourceModel) =>
    api
      .put<ApiResponse<ResourceModel>>(`/resources/${resourceId}`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  getResourceLogs: (resourceId: string, tail = 200) =>
    api
      .get<ApiResponse<ResourceLogsModel>>(`/resources/${resourceId}/logs`, {
        params: { tail },
      })
      .then((res) => unwrapApiResponse(res.data)),

  deleteResource: (resourceId: string) => api.delete(`/resources/${resourceId}`),

  startResource: (resourceId: string) =>
    api
      .post<ApiResponse<ResourceRunModel>>(`/resources/${resourceId}/start`)
      .then((res) => unwrapApiResponse(res.data)),

  stopResource: (resourceId: string) => api.post(`/resources/${resourceId}/stop`),

  scaleResource: (resourceId: string, replicas: number) =>
    api.post(`/resources/${resourceId}/scale`, { replicas }),

  reconcileResources: () =>
    api
      .post<ApiResponse<ResourceRuntimeReconcileResponse>>(`/resources/reconcile`)
      .then((res) => unwrapApiResponse(res.data)),

  reconcileResource: (resourceId: string) =>
    api
      .post<ApiResponse<ResourceRuntimeReconcileResponse>>(`/resources/${resourceId}/reconcile`)
      .then((res) => unwrapApiResponse(res.data)),

  // ── Resource env vars ────────────────────────────────────────────────────────

  getResourceEnvVars: (resourceId: string) =>
    api
      .get<ApiResponse<ResourceEnvVarModel[]>>(`/resources/${resourceId}/env-vars`)
      .then((res) => unwrapApiResponse(res.data)),

  setResourceEnvVars: (
    resourceId: string,
    vars: Array<{ key: string; value: string; is_secret: boolean }>
  ) =>
    api
      .put<ApiResponse<ResourceEnvVarModel[]>>(`/resources/${resourceId}/env-vars`, { vars })
      .then((res) => unwrapApiResponse(res.data)),

  createResourceFromGit: (envId: string, payload: CreateResourceFromGitModel) =>
    api
      .post<ApiResponse<ResourceModel>>(`/environments/${envId}/resources/from-git`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  checkRepo: (url: string, branch?: string, token?: string) =>
    api
      .get<{ data: CheckRepoResponse }>(`/builds/check-repo`, {
        params: { url, branch, token },
      })
      .then((res) => res.data.data),

  buildResource: (resourceId: string) =>
    api
      .post<ApiResponse<{ build_job_id: string }>>(`/resources/${resourceId}/build`)
      .then((res) => unwrapApiResponse(res.data)),

  // ── Resource domains ─────────────────────────────────────────────────────────

  listResourceDomains: (resourceId: string) =>
    api
      .get<ResourceDomainModel[]>(`/resources/${resourceId}/domains`)
      .then((res) => res.data),

  addResourceDomain: (resourceId: string, payload: { host: string; target_port: number; tls_enabled: boolean; auto_verify?: boolean }) =>
    api
      .post<ResourceDomainModel>(`/resources/${resourceId}/domains`, payload)
      .then((res) => res.data),

  updateResourceDomain: (
    resourceId: string,
    domainId: string,
    payload: { host: string; target_port: number; tls_enabled: boolean }
  ) =>
    api
      .patch<ResourceDomainModel>(`/resources/${resourceId}/domains/${domainId}`, payload)
      .then((res) => res.data),

  removeResourceDomain: (resourceId: string, domainId: string) =>
    api.delete(`/resources/${resourceId}/domains/${domainId}`),

  verifyResourceDomain: (resourceId: string, domainId: string) =>
    api
      .post<{ verified: boolean; resolved_ips: string[] }>(
        `/resources/${resourceId}/domains/${domainId}/verify`
      )
      .then((res) => res.data),
}
