import type {
  ContainerModel,
  ContainerDetailsModel,
  ContainerStatsModel,
  CreateContainerModel,
  ImageModel,
  PullImageModel,
} from "@/@types/models"
import type { ApiResponse } from "@/@types/models/common"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const BASE = "/docker"

export const containerService = {
  listContainers: (all = false) =>
    api
      .get<ApiResponse<ContainerModel[]>>(`${BASE}/containers`, { params: { all } })
      .then((res) => unwrapApiResponse(res.data)),

  getContainer: (id: string) =>
    api
      .get<ApiResponse<ContainerDetailsModel>>(`${BASE}/containers/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  getContainerStats: (id: string) =>
    api
      .get<ApiResponse<ContainerStatsModel>>(`${BASE}/containers/${id}/stats`)
      .then((res) => unwrapApiResponse(res.data)),

  createContainer: (payload: CreateContainerModel) =>
    api
      .post<ApiResponse<ContainerModel>>(`${BASE}/containers`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  startContainer: (id: string) =>
    api.post(`${BASE}/containers/${id}/start`),

  stopContainer: (id: string) =>
    api.post(`${BASE}/containers/${id}/stop`),

  removeContainer: (id: string, force = false) =>
    api.delete(`${BASE}/containers/${id}`, { params: { force } }),

  listImages: () =>
    api
      .get<ApiResponse<ImageModel[]>>(`${BASE}/images`)
      .then((res) => unwrapApiResponse(res.data)),

  pullImage: (payload: PullImageModel) =>
    api.post(`${BASE}/images/pull`, payload),

  removeImage: (id: string, force = false) =>
    api.delete(`${BASE}/images/${encodeURIComponent(id)}`, { params: { force } }),
}
