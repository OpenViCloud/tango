import type { SwarmStatusModel, SwarmNodeModel } from "@/@types/models"
import { api } from "@/lib/api"

export const swarmService = {
  getStatus: (): Promise<SwarmStatusModel> =>
    api.get<SwarmStatusModel>("/swarm/status").then((res) => res.data),

  listNodes: (): Promise<SwarmNodeModel[]> =>
    api
      .get<SwarmNodeModel[]>("/swarm/nodes")
      .then((res) => res.data)
      .catch((err) => {
        // 503 = not a swarm manager — return empty list instead of throwing
        if (err?.response?.status === 503) return []
        throw err
      }),
}
