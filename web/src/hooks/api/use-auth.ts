import { useMutation } from "@tanstack/react-query"

import {
  authService,
  type ChangePasswordPayload,
} from "@/services/api/auth-service"

export const useChangePassword = () =>
  useMutation({
    mutationFn: (payload: ChangePasswordPayload) =>
      authService.changePassword(payload),
  })
