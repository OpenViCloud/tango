import type { ApiResponse } from "@/@types/models/common"
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
}

export type { ChangePasswordPayload }
