import { z } from "zod"
import type { BaseRequestModel } from "./common"

export const createUserSchema = z.object({
  email: z.string().email("validation.email"),
  first_name: z.string().min(1, "validation.required"),
  last_name: z.string().min(1, "validation.required"),
  password: z.string().min(6, "validation.passwordMin"),
  phone: z.string().optional(),
  address: z.string().optional(),
})

export type CreateUserModel = z.infer<typeof createUserSchema>

export const updateUserSchema = createUserSchema.omit({ email: true, password: true })

export type UpdateUserModel = z.infer<typeof updateUserSchema>

export const userSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  nickname: z.string().optional().default(""),
  first_name: z.string(),
  last_name: z.string(),
  phone: z.string(),
  address: z.string(),
  status: z.string(),
  created_at: z.string(),
})

export type UserModel = z.infer<typeof userSchema>

export type GetUserRequestModel = {
  active?: boolean
} & BaseRequestModel
