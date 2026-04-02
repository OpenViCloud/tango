import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { ConnectPATModel, CreateGitHubAppManifestModel } from "@/@types/models"
import { sourceService } from "@/services/api/source-service"

export const SOURCE_QUERY_KEYS = {
  list: () => ["sources"],
  repos: (connectionId: string) => ["sources", connectionId, "repos"],
  branches: (connectionId: string, owner: string, repo: string) => [
    "sources",
    connectionId,
    "branches",
    owner,
    repo,
  ],
}

export const useGetSourceList = () =>
  useQuery({
    queryKey: SOURCE_QUERY_KEYS.list(),
    queryFn: () => sourceService.listSources(),
  })

export const useBeginGitHubAppManifest = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateGitHubAppManifestModel) =>
      sourceService.beginGitHubAppManifest(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SOURCE_QUERY_KEYS.list() })
    },
  })
}

export const useConnectPAT = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: ConnectPATModel) => sourceService.connectPAT(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SOURCE_QUERY_KEYS.list() })
    },
  })
}

export const useGetSourceRepos = (connectionId: string) =>
  useQuery({
    queryKey: SOURCE_QUERY_KEYS.repos(connectionId),
    queryFn: () => sourceService.listRepositories(connectionId),
    enabled: !!connectionId,
  })

export const useGetSourceBranches = (
  connectionId: string,
  owner: string,
  repo: string
) =>
  useQuery({
    queryKey: SOURCE_QUERY_KEYS.branches(connectionId, owner, repo),
    queryFn: () => sourceService.listBranches(connectionId, owner, repo),
    enabled: !!connectionId && !!owner && !!repo,
  })
