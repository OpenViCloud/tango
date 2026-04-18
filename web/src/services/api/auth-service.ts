import type { ApiResponse } from "@/@types/models/common"
import type { RegisterRequestModel, SetupStatusResponse } from "@/@types/models/auth"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

type ChangePasswordPayload = {
  current_password: string
  new_password: string
}

type MessageResponse = {
  message: string
}

export const authService = {
  changePassword: (payload: ChangePasswordPayload) =>
    api
      .post<ApiResponse<MessageResponse>>("/auth/change-password", payload)
      .then((res) => unwrapApiResponse(res.data)),

  setupStatus: () =>
    api
      .get<ApiResponse<SetupStatusResponse>>("/auth/setup-status")
      .then((res) => unwrapApiResponse(res.data)),

  register: (payload: RegisterRequestModel) =>
    api
      .post("/auth/register", payload, { withCredentials: true })
      .then(() => undefined),
}

export type { ChangePasswordPayload }
