import type {
  ChannelModel,
  ChannelQRCodeModel,
  CreateChannelModel,
  DiscordRuntimeModel,
  DiscordRuntimeRequestModel,
  GetChannelRequestModel,
  TestChannelConnectionModel,
  UpdateChannelModel,
} from "@/@types/models"
import type { ApiResponse, PagedResponse } from "@/@types/models/common"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const baseUrl = "/channels"

export const channelService = {
  getById: (channelId: string) =>
    api
      .get<ApiResponse<ChannelModel>>(`${baseUrl}/${channelId}`)
      .then((res) => unwrapApiResponse(res.data)),
  getList: (params: GetChannelRequestModel) =>
    api
      .get<ApiResponse<PagedResponse<ChannelModel>>>(baseUrl, { params })
      .then((res) => unwrapApiResponse(res.data)),
  createChannel: (payload: CreateChannelModel) =>
    api
      .post<ApiResponse<ChannelModel>>(baseUrl, payload)
      .then((res) => unwrapApiResponse(res.data)),
  updateChannel: (channelId: string, payload: UpdateChannelModel) =>
    api
      .put<ApiResponse<ChannelModel>>(`${baseUrl}/${channelId}`, payload)
      .then((res) => unwrapApiResponse(res.data)),
  deleteChannel: (channelId: string) => api.delete(`${baseUrl}/${channelId}`),
  getQRCode: (channelId: string) =>
    api
      .get<ApiResponse<ChannelQRCodeModel>>(`${baseUrl}/${channelId}/qr-code`)
      .then((res) => unwrapApiResponse(res.data)),
  testConnection: (payload: TestChannelConnectionModel) =>
    api
      .post<ApiResponse<{ message?: string }>>(`${baseUrl}/test-connection`, payload)
      .then((res) => unwrapApiResponse(res.data)),
  startChannel: (channelId: string) =>
    api
      .post<ApiResponse<ChannelModel>>(`${baseUrl}/${channelId}/start`)
      .then((res) => unwrapApiResponse(res.data)),
  stopChannel: (channelId: string) =>
    api
      .post<ApiResponse<ChannelModel>>(`${baseUrl}/${channelId}/stop`)
      .then((res) => unwrapApiResponse(res.data)),
  restartChannel: (channelId: string) =>
    api
      .post<ApiResponse<ChannelModel>>(`${baseUrl}/${channelId}/restart`)
      .then((res) => unwrapApiResponse(res.data)),
  getDiscordRuntimeStatus: () =>
    api
      .get<ApiResponse<DiscordRuntimeModel>>("/runtime/discord/status")
      .then((res) => unwrapApiResponse(res.data)),
  startDiscordRuntime: (payload: DiscordRuntimeRequestModel) =>
    api
      .post<ApiResponse<DiscordRuntimeModel>>("/runtime/discord/start", payload)
      .then((res) => unwrapApiResponse(res.data)),
  restartDiscordRuntime: () =>
    api
      .post<ApiResponse<DiscordRuntimeModel>>("/runtime/discord/restart")
      .then((res) => unwrapApiResponse(res.data)),
  stopDiscordRuntime: () =>
    api
      .post<ApiResponse<DiscordRuntimeModel>>("/runtime/discord/stop")
      .then((res) => unwrapApiResponse(res.data)),
}
