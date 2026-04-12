import type {
  ApiResponse,
  CloudflareConnectionModel,
  CreateCloudflareConnectionModel,
  UpdateCloudflareConnectionModel,
} from "@/@types/models"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

export const cloudflareService = {
  list: () =>
    api
      .get<ApiResponse<CloudflareConnectionModel[]>>("/cloudflare/connections")
      .then((res) => unwrapApiResponse(res.data)),

  getById: (id: string) =>
    api
      .get<ApiResponse<CloudflareConnectionModel>>(`/cloudflare/connections/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  create: (payload: CreateCloudflareConnectionModel) =>
    api
      .post<ApiResponse<CloudflareConnectionModel>>("/cloudflare/connections", payload)
      .then((res) => unwrapApiResponse(res.data)),

  update: (id: string, payload: UpdateCloudflareConnectionModel) =>
    api
      .put<ApiResponse<CloudflareConnectionModel>>(`/cloudflare/connections/${id}`, payload)
      .then((res) => unwrapApiResponse(res.data)),
}
