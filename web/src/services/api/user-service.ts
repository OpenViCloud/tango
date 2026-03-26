import type {
  CreateUserModel,
  GetUserRequestModel,
  UpdateUserModel,
  UserModel,
} from "@/@types/models"
import type { ApiResponse, PagedResponse } from "@/@types/models/common"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const baseUrl = "/users"

export const userService = {
  getById: (userId: string) =>
    api
      .get<ApiResponse<UserModel>>(`/user/${userId}`)
      .then((res) => unwrapApiResponse(res.data)),
  getList: (params: GetUserRequestModel) =>
    api
      .get<ApiResponse<PagedResponse<UserModel>>>(`${baseUrl}`, { params })
      .then((res) => unwrapApiResponse(res.data)),
  createUser: (payload: CreateUserModel) =>
    api
      .post<ApiResponse<UserModel>>(baseUrl, {
        id: crypto.randomUUID(),
        ...payload,
      })
      .then((res) => unwrapApiResponse(res.data)),
  updateUser: (userId: string, payload: UpdateUserModel) =>
    api
      .put<ApiResponse<UserModel>>(`${baseUrl}/${userId}`, payload)
      .then((res) => unwrapApiResponse(res.data)),
  deleteUser: (userId: string) => api.delete(`${baseUrl}/${userId}`),
}
