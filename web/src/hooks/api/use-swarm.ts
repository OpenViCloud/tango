import { useQuery } from "@tanstack/react-query"
import { swarmService } from "@/services/api/swarm-service"

export const SWARM_QUERY_KEYS = {
  status: () => ["swarm", "status"],
  nodes: () => ["swarm", "nodes"],
}

export const useSwarmStatus = () =>
  useQuery({
    queryKey: SWARM_QUERY_KEYS.status(),
    queryFn: () => swarmService.getStatus(),
    staleTime: 30_000,
  })

export const useSwarmNodes = (enabled = true) =>
  useQuery({
    queryKey: SWARM_QUERY_KEYS.nodes(),
    queryFn: () => swarmService.listNodes(),
    enabled,
    staleTime: 15_000,
  })
