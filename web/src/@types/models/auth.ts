import * as z from "zod"

export const loginSchema = z.object({
  email: z.email("Email không hợp lệ"),
  password: z.string().min(6, "Tối thiểu 6 ký tự"),
})

export type LoginRequestModel = z.infer<typeof loginSchema>

export type AuthTokenResponse = {
  access_token: string
}

// export type RegisterRequest = {
//   email: string
//   first_name: string
//   last_name: string
//   password: string
//   phone?: string
//   address?: string
// }

// export type RegisterResponse = UserModel

// export type CreateUserRequest = {
//   id: string
//   email: string
//   first_name: string
//   last_name: string
//   password: string
//   phone?: string
//   address?: string
// }

// export type CreateUserResponse = UserModel
