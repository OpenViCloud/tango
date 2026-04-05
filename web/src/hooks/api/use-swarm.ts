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

export const useSwarmNodes = (_enabled = true) =>
  useQuery({
    queryKey: SWARM_QUERY_KEYS.nodes(),
    // Fetch unconditionally — 503 (not a manager) is caught and returns [].
    // This avoids a race with useSwarmStatus where nodes are never loaded
    // because `enabled` flips true too late for the initial render.
    queryFn: () => swarmService.listNodes(),
    staleTime: 15_000,
    refetchOnMount: true,
  })
