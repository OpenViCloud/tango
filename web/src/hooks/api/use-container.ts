import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { CreateContainerModel, PullImageModel } from "@/@types/models"
import { containerService } from "@/services/api/container-service"

export const CONTAINER_QUERY_KEYS = {
  containers: (all: boolean) => ["UseContainerList", all],
  container: (id: string) => ["UseContainer", id],
  containerStats: (id: string) => ["UseContainerStats", id],
  images: () => ["UseImageList"],
}

export const useGetContainerList = (all = false) =>
  useQuery({
    queryKey: CONTAINER_QUERY_KEYS.containers(all),
    queryFn: () => containerService.listContainers(all),
    refetchInterval: 5000,
  })

export const useGetContainer = (id: string) =>
  useQuery({
    queryKey: CONTAINER_QUERY_KEYS.container(id),
    queryFn: () => containerService.getContainer(id),
    enabled: !!id,
  })

export const useGetContainerStats = (id: string, enabled = true) =>
  useQuery({
    queryKey: CONTAINER_QUERY_KEYS.containerStats(id),
    queryFn: () => containerService.getContainerStats(id),
    enabled: !!id && enabled,
    refetchInterval: enabled ? 5000 : false,
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
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
      queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.container(id) })
      queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.containerStats(id) })
    },
  })
}

export const useStopContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => containerService.stopContainer(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
      queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.container(id) })
      queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.containerStats(id) })
    },
  })
}

export const useRemoveContainer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      containerService.removeContainer(id, force),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["UseContainerList"] })
      queryClient.removeQueries({ queryKey: CONTAINER_QUERY_KEYS.container(id) })
      queryClient.removeQueries({ queryKey: CONTAINER_QUERY_KEYS.containerStats(id) })
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
