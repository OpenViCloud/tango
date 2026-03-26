import type {
  CreateRoleModel,
  GetRoleRequestModel,
  RoleModel,
  UpdateRoleModel,
} from "@/@types/models"
import type { ApiResponse, PagedResponse } from "@/@types/models/common"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const baseUrl = "/roles"

export const roleService = {
  getList: (params: GetRoleRequestModel) =>
    api
      .get<ApiResponse<PagedResponse<RoleModel>>>(baseUrl, { params })
      .then((res) => unwrapApiResponse(res.data)),
  createRole: (payload: CreateRoleModel) =>
    api
      .post<ApiResponse<RoleModel>>(baseUrl, payload)
      .then((res) => unwrapApiResponse(res.data)),
  updateRole: (roleId: string, payload: UpdateRoleModel) =>
    api
      .put<ApiResponse<RoleModel>>(`${baseUrl}/${roleId}`, payload)
      .then((res) => unwrapApiResponse(res.data)),
  deleteRole: (roleId: string) => api.delete(`${baseUrl}/${roleId}`),
}
