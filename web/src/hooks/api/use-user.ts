import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { userService } from "@/services/api/user-service"
import type {
  CreateUserModel,
  GetUserRequestModel,
  UpdateUserModel,
} from "@/@types/models"

export const USER_QUERY_KEYS = {
  UseGetUserById: (userId: string) => ["UseGetUserById", userId],
  UseGetUserList: (params?: GetUserRequestModel) => [
    "UseGetUserList",
    ...Object.values(params || {}),
  ],
}

export const useGetUserById = (userId: string) =>
  useQuery({
    queryKey: USER_QUERY_KEYS.UseGetUserById(userId),
    queryFn: () => userService.getById(userId),
    enabled: Boolean(userId),
  })

export const useGetUserList = (params: GetUserRequestModel) =>
  useQuery({
    queryKey: USER_QUERY_KEYS.UseGetUserList(params),
    queryFn: () => userService.getList(params),
  })

export const useCreateUser = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateUserModel) => userService.createUser(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetUserList"] })
    },
  })
}

export const useUpdateUser = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      userId,
      payload,
    }: {
      userId: string
      payload: UpdateUserModel
    }) => userService.updateUser(userId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetUserList"] })
      void queryClient.invalidateQueries({ queryKey: ["UseGetUserById"] })
    },
  })
}

export const useDeleteUsers = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (userIds: string[]) => {
      const results = await Promise.allSettled(
        userIds.map((userId) => userService.deleteUser(userId))
      )

      const failedCount = results.filter(
        (result) => result.status === "rejected"
      ).length

      if (failedCount > 0) {
        throw new Error(
          failedCount === userIds.length
            ? "Delete users failed."
            : `Deleted ${userIds.length - failedCount}/${userIds.length} users.`
        )
      }

      return results
    },
    meta: {
      skipErrorToast: true,
    },
    onSuccess: (_result, userIds) => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetUserList"] })
      toast.success(
        userIds.length === 1 ? "User deleted." : `${userIds.length} users deleted.`
      )
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Delete users failed.")
    },
  })
}
