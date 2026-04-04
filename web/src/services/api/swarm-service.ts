import type { SwarmStatusModel, SwarmNodeModel } from "@/@types/models"
import { api } from "@/lib/api"

export const swarmService = {
  getStatus: (): Promise<SwarmStatusModel> =>
    api.get<SwarmStatusModel>("/swarm/status").then((res) => res.data),

  listNodes: (): Promise<SwarmNodeModel[]> =>
    api.get<SwarmNodeModel[]>("/swarm/nodes").then((res) => res.data),
}
