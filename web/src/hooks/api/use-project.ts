import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type {
  CreateProjectModel,
  UpdateProjectModel,
  CreateEnvironmentModel,
  CreateResourceModel,
  UpdateResourceModel,
} from "@/@types/models"
import type {
  CreateResourceFromGitModel,
  CreateResourceStackModel,
} from "@/@types/models/project"
import { projectService } from "@/services/api/project-service"

export const PROJECT_QUERY_KEYS = {
  projects: () => ["projects"],
  project: (id: string) => ["project", id],
  resourceTemplates: () => ["resourceTemplates"],
  resource: (resourceId: string) => ["resource", resourceId],
  resourceConnectionInfo: (resourceId: string) => ["resourceConnectionInfo", resourceId],
  resourceEnvVars: (resourceId: string) => ["resourceEnvVars", resourceId],
  resourceLogs: (resourceId: string, tail: number) => ["resourceLogs", resourceId, tail],
}

// ── Projects ──────────────────────────────────────────────────────────────────

export const useGetProjectList = () =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.projects(),
    queryFn: () => projectService.listProjects(),
  })

export const useGetProject = (id: string) =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.project(id),
    queryFn: () => projectService.getProject(id),
    enabled: !!id,
  })

export const useGetResourceTemplates = () =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.resourceTemplates(),
    queryFn: () => projectService.listResourceTemplates(),
  })

export const useGetResourceStackTemplates = () =>
  useQuery({
    queryKey: ["resourceStackTemplates"],
    queryFn: () => projectService.listResourceStackTemplates(),
  })

export const useCreateProject = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateProjectModel) => projectService.createProject(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.projects() })
    },
  })
}

export const useUpdateProject = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateProjectModel }) =>
      projectService.updateProject(id, payload),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(id) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.projects() })
    },
  })
}

export const useDeleteProject = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => projectService.deleteProject(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.projects() })
    },
  })
}

// ── Environments ──────────────────────────────────────────────────────────────

export const useCreateEnvironment = (projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateEnvironmentModel) =>
      projectService.createEnvironment(projectId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

export const useDeleteEnvironment = (projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (envId: string) => projectService.deleteEnvironment(envId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

export const useForkEnvironment = (projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ envId, name }: { envId: string; name: string }) =>
      projectService.forkEnvironment(envId, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

// ── Resources ─────────────────────────────────────────────────────────────────

export const useGetResourceList = (environmentId: string) =>
  useQuery({
    queryKey: ["resourceList", environmentId],
    queryFn: () => projectService.listResources(environmentId),
    enabled: !!environmentId,
  })

export const useGetResource = (resourceId: string) =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.resource(resourceId),
    queryFn: () => projectService.getResource(resourceId),
    enabled: !!resourceId,
  })

export const useGetResourceConnectionInfo = (resourceId: string) =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId),
    queryFn: () => projectService.getResourceConnectionInfo(resourceId),
    enabled: !!resourceId,
  })

export const useGetResourceLogs = (resourceId: string, tail = 200) =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.resourceLogs(resourceId, tail),
    queryFn: () => projectService.getResourceLogs(resourceId, tail),
    enabled: !!resourceId,
    refetchInterval: 5000,
  })

export const useCreateResource = (environmentId: string, projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateResourceModel) =>
      projectService.createResource(environmentId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

export const useCreateResourceStack = (environmentId: string, projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateResourceStackModel) =>
      projectService.createResourceStack(environmentId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

export const useUpdateResource = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: UpdateResourceModel) =>
      projectService.updateResource(resourceId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

export const useStartResource = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (resourceId: string) => projectService.startResource(resourceId),
    onSuccess: (_data, resourceId) => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

export const useStopResource = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (resourceId: string) => projectService.stopResource(resourceId),
    onSuccess: (_data, resourceId) => {
      queryClient.invalidateQueries({ queryKey: ["project"] })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

export const useScaleResource = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ resourceId, replicas }: { resourceId: string; replicas: number }) =>
      projectService.scaleResource(resourceId, replicas),
    onSuccess: (_data, { resourceId }) => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
    },
  })
}

export const useReconcileResources = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => projectService.reconcileResources(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.projects() })
      queryClient.invalidateQueries({ queryKey: ["project"] })
      queryClient.invalidateQueries({ queryKey: ["resourceList"] })
      queryClient.invalidateQueries({ queryKey: ["resource"] })
    },
  })
}

export const useReconcileResource = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => projectService.reconcileResource(resourceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.projects() })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

export const useDeleteResource = (projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (resourceId: string) => projectService.deleteResource(resourceId),
    onSuccess: (_data, resourceId) => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
      queryClient.removeQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.removeQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

export const useCreateResourceFromGit = (environmentId: string, projectId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateResourceFromGitModel) =>
      projectService.createResourceFromGit(environmentId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.project(projectId) })
    },
  })
}

export const useCheckRepo = () => {
  return useMutation({
    mutationFn: ({ url, branch, token }: { url: string; branch?: string; token?: string }) =>
      projectService.checkRepo(url, branch, token),
  })
}

export const useBuildResource = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => projectService.buildResource(resourceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
      queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resourceConnectionInfo(resourceId) })
    },
  })
}

// ── Resource env vars ─────────────────────────────────────────────────────────

export const useGetResourceEnvVars = (resourceId: string) =>
  useQuery({
    queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
    queryFn: () => projectService.getResourceEnvVars(resourceId),
    enabled: !!resourceId,
  })

export const useSetResourceEnvVars = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: Array<{ key: string; value: string; is_secret: boolean }>) =>
      projectService.setResourceEnvVars(resourceId, vars),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
      })
    },
  })
}

// ── Resource domains ──────────────────────────────────────────────────────────

export const DOMAIN_QUERY_KEY = (resourceId: string) => ["resourceDomains", resourceId]

export const useListResourceDomains = (resourceId: string) =>
  useQuery({
    queryKey: DOMAIN_QUERY_KEY(resourceId),
    queryFn: () => projectService.listResourceDomains(resourceId),
    enabled: !!resourceId,
  })

export const useAddResourceDomain = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { host: string; target_port: number; tls_enabled: boolean }) =>
      projectService.addResourceDomain(resourceId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DOMAIN_QUERY_KEY(resourceId) })
    },
  })
}

export const useRemoveResourceDomain = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (domainId: string) => projectService.removeResourceDomain(resourceId, domainId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DOMAIN_QUERY_KEY(resourceId) })
    },
  })
}

export const useUpdateResourceDomain = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      domainId,
      payload,
    }: {
      domainId: string
      payload: { host: string; target_port: number; tls_enabled: boolean }
    }) => projectService.updateResourceDomain(resourceId, domainId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DOMAIN_QUERY_KEY(resourceId) })
    },
  })
}

export const useVerifyResourceDomain = (resourceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (domainId: string) => projectService.verifyResourceDomain(resourceId, domainId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DOMAIN_QUERY_KEY(resourceId) })
    },
  })
}
