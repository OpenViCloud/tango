import * as z from "zod"

export const loginSchema = z.object({
  email: z.email("Email không hợp lệ"),
  password: z.string().min(6, "Tối thiểu 6 ký tự"),
})

export type LoginRequestModel = z.infer<typeof loginSchema>

export const registerSchema = z.object({
  email: z.email("Email không hợp lệ"),
  first_name: z.string().min(1, "Bắt buộc"),
  last_name: z.string().min(1, "Bắt buộc"),
  password: z.string().min(6, "Tối thiểu 6 ký tự"),
})

export type RegisterRequestModel = z.infer<typeof registerSchema>

export type AuthTokenResponse = {
  access_token: string
}

export type SetupStatusResponse = {
  setup_required: boolean
}
