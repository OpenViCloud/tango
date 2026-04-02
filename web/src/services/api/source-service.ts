import type {
  ApiResponse,
  BeginGitHubAppManifestResponseModel,
  ConnectPATModel,
  CreateGitHubAppManifestModel,
  GitBranchModel,
  GitRepositoryModel,
  SourceConnectionModel,
} from "@/@types/models"
import { unwrapApiResponse } from "@/lib/api-response"
import { api } from "@/lib/api"

export const sourceService = {
  listSources: () =>
    api
      .get<ApiResponse<SourceConnectionModel[]>>("/source-connections")
      .then((res) => unwrapApiResponse(res.data)),

  beginGitHubAppManifest: (payload: CreateGitHubAppManifestModel) =>
    api
      .post<ApiResponse<BeginGitHubAppManifestResponseModel>>(
        "/source-connections/github/apps",
        payload
      )
      .then((res) => unwrapApiResponse(res.data)),

  connectPAT: (payload: ConnectPATModel) =>
    api
      .post<ApiResponse<SourceConnectionModel>>("/source-connections/pat", payload)
      .then((res) => unwrapApiResponse(res.data)),

  listRepositories: (connectionId: string) =>
    api
      .get<ApiResponse<GitRepositoryModel[]>>(`/source-connections/${connectionId}/repos`)
      .then((res) => unwrapApiResponse(res.data)),

  listBranches: (connectionId: string, owner: string, repo: string) =>
    api
      .get<ApiResponse<GitBranchModel[]>>(
        `/source-connections/${connectionId}/repos/${owner}/${repo}/branches`
      )
      .then((res) => unwrapApiResponse(res.data)),
}
