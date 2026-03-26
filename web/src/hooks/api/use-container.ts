import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { CreateContainerModel, PullImageModel } from "@/@types/models"
import { containerService } from "@/services/api/container-service"

export const CONTAINER_QUERY_KEYS = {
  containers: (all: boolean) => ["UseContainerList", all],
  images: () => ["UseImageList"],
}

export const useGetContainerList = (all = false) =>
  useQuery({
    queryKey: CONTAINER_QUERY_KEYS.containers(all),
    queryFn: () => containerService.listContainers(all),
    refetchInterval: 5000,
  })

export const useGetImageList = () =>
  useQuery({
    queryKey: CONTAINER_QUERY_KEYS.images(),
    queryFn: () => containerService.listImages(),
  })

export const useCreateContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateContainerModel) =>
      containerService.createContainer(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
    },
  })
}

export const useStartContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => containerService.startContainer(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
    },
  })
}

export const useStopContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => containerService.stopContainer(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
    },
  })
}

export const useRemoveContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      containerService.removeContainer(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
    },
  })
}

export const usePullImage = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: PullImageModel) => containerService.pullImage(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseImageList"] })
    },
  })
}

export const useRemoveImage = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      containerService.removeImage(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["UseImageList"] })
    },
  })
}
