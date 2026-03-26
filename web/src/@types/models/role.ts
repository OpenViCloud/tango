import { z } from "zod"

import type { BaseRequestModel } from "./common"

export const createRoleSchema = z.object({
  name: z.string().min(1, "validation.required"),
  description: z.string(),
})

export type CreateRoleModel = z.infer<typeof createRoleSchema>

export const updateRoleSchema = createRoleSchema

export type UpdateRoleModel = z.infer<typeof updateRoleSchema>

export const roleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  is_system: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
})

export type RoleModel = z.infer<typeof roleSchema>

export type GetRoleRequestModel = BaseRequestModel
