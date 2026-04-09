import type { ApiResponse } from "@/@types/models/common"
import type { CreateServerModel, ServerModel } from "@/@types/models/server"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

export const serverService = {
  getSshPublicKey: (): Promise<{ public_key: string }> =>
    api
      .get<ApiResponse<{ public_key: string }>>("/servers/ssh-public-key")
      .then((res) => unwrapApiResponse(res.data)),

  list: (): Promise<ServerModel[]> =>
    api
      .get<ApiResponse<ServerModel[]>>("/servers")
      .then((res) => unwrapApiResponse(res.data)),

  create: (payload: CreateServerModel): Promise<ServerModel> =>
    api
      .post<ApiResponse<ServerModel>>("/servers", payload)
      .then((res) => unwrapApiResponse(res.data)),

  delete: (id: string): Promise<void> =>
    api.delete(`/servers/${id}`).then(() => undefined),

  ping: (id: string): Promise<{ status: string; error?: string }> =>
    api
      .post<ApiResponse<{ status: string; error?: string }>>(
        `/servers/${id}/ping`,
      )
      .then((res) => unwrapApiResponse(res.data))
      .catch((err) => {
        // 502 = SSH failed — return the response body so UI can show error
        if (err?.response?.data?.status) return err.response.data
        throw err
      }),
}
