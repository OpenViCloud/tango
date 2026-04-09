import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import type { CreateServerModel } from "@/@types/models/server"
import { getErrorMessage } from "@/lib/get-error-message"
import { serverService } from "@/services/api/server-service"

export const SERVER_QUERY_KEYS = {
  list: () => ["servers"],
  sshPublicKey: () => ["servers", "ssh-public-key"],
}

export const useGetSshPublicKey = () =>
  useQuery({
    queryKey: SERVER_QUERY_KEYS.sshPublicKey(),
    queryFn: () => serverService.getSshPublicKey(),
    staleTime: Infinity,
  })

export const useGetServerList = () =>
  useQuery({
    queryKey: SERVER_QUERY_KEYS.list(),
    queryFn: () => serverService.list(),
  })

export const useCreateServer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateServerModel) => serverService.create(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SERVER_QUERY_KEYS.list() })
      toast.success("Server added.")
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

export const useDeleteServer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => serverService.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SERVER_QUERY_KEYS.list() })
      toast.success("Server removed.")
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

export const usePingServer = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => serverService.ping(id),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: SERVER_QUERY_KEYS.list() })
      if (result.status === "connected") {
        toast.success("SSH connection successful.")
      } else {
        toast.error(result.error ?? "SSH connection failed.")
      }
    },
    onError: (error) => {
      void queryClient.invalidateQueries({ queryKey: SERVER_QUERY_KEYS.list() })
      toast.error(getErrorMessage(error))
    },
  })
}
