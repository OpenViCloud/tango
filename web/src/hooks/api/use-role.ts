import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import type {
  CreateRoleModel,
  GetRoleRequestModel,
  UpdateRoleModel,
} from "@/@types/models"
import { roleService } from "@/services/api/role-service"

export const ROLE_QUERY_KEYS = {
  UseGetRoleList: (params?: GetRoleRequestModel) => [
    "UseGetRoleList",
    ...Object.values(params || {}),
  ],
}

export const useGetRoleList = (params: GetRoleRequestModel) =>
  useQuery({
    queryKey: ROLE_QUERY_KEYS.UseGetRoleList(params),
    queryFn: () => roleService.getList(params),
  })

export const useCreateRole = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateRoleModel) => roleService.createRole(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetRoleList"] })
    },
  })
}

export const useUpdateRole = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      roleId,
      payload,
    }: {
      roleId: string
      payload: UpdateRoleModel
    }) => roleService.updateRole(roleId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetRoleList"] })
    },
  })
}

export const useDeleteRoles = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (roleIds: string[]) => {
      const results = await Promise.allSettled(
        roleIds.map((roleId) => roleService.deleteRole(roleId))
      )

      const failedCount = results.filter(
        (result) => result.status === "rejected"
      ).length

      if (failedCount > 0) {
        throw new Error(
          failedCount === roleIds.length
            ? "Delete roles failed."
            : `Deleted ${roleIds.length - failedCount}/${roleIds.length} roles.`
        )
      }

      return results
    },
    meta: {
      skipErrorToast: true,
    },
    onSuccess: (_result, roleIds) => {
      void queryClient.invalidateQueries({ queryKey: ["UseGetRoleList"] })
      toast.success(
        roleIds.length === 1 ? "Role deleted." : `${roleIds.length} roles deleted.`
      )
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Delete roles failed.")
    },
  })
}
