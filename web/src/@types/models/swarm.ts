import { z } from "zod"

export const swarmStatusSchema = z.object({
  is_manager: z.boolean(),
})

export const swarmNodeSchema = z.object({
  id: z.string(),
  hostname: z.string(),
  role: z.string(),
  state: z.string(),
  availability: z.string(),
  manager_addr: z.string().optional(),
})

export type SwarmStatusModel = z.infer<typeof swarmStatusSchema>
export type SwarmNodeModel = z.infer<typeof swarmNodeSchema>
