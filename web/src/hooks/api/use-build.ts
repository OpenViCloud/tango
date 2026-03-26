import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { CreateBuildJobModel, GetBuildJobListRequestModel } from "@/@types/models"
import { buildService } from "@/services/api/build-service"

export const BUILD_QUERY_KEYS = {
  list: (params: GetBuildJobListRequestModel) => ["UseBuildJobList", params],
  getById: (id: string) => ["UseBuildJob", id],
}

export const useGetBuildJobList = (params: GetBuildJobListRequestModel) =>
  useQuery({
    queryKey: BUILD_QUERY_KEYS.list(params),
    queryFn: () => buildService.getList(params),
  })

export const useGetBuildJob = (id: string | null) =>
  useQuery({
    queryKey: BUILD_QUERY_KEYS.getById(id ?? ""),
    queryFn: () => buildService.getById(id ?? ""),
    enabled: Boolean(id),
    // Poll every 3s while job is still running
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (!status) return false
      const active = ["queued", "cloning", "detecting", "generating", "building"]
      return active.includes(status) ? 3000 : false
    },
  })

export const useCreateBuildJob = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateBuildJobModel) => buildService.create(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseBuildJobList"] })
    },
  })
}

export const useCancelBuildJob = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => buildService.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseBuildJobList"] })
    },
  })
}
