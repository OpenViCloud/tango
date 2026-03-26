import { create } from "zustand"
import axios from "axios"
import type { ApiResponse } from "@/@types/models/common"
import type { AuthTokenResponse } from "@/@types/models"
import { unwrapApiResponse } from "@/lib/api-response"

interface AuthState {
  accessToken: string | null
  isLoading: boolean
  error: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<string | null>
  clearError: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  isLoading: false,
  error: null,

  // ── Đăng nhập ──────────────────────────────────
  login: async (email, password) => {
    set({ isLoading: true, error: null })
    try {
      const res = await axios.post<ApiResponse<AuthTokenResponse>>("/api/auth/login", {
        email,
        password,
      })
      const token = unwrapApiResponse(res.data).access_token
      // access_token lưu trong memory (không localStorage — bảo mật hơn)
      // refresh_token server set httpOnly cookie tự động
      set({ accessToken: token, isLoading: false })
    } catch (err: any) {
      const msg = err.response?.data?.error || "Đăng nhập thất bại"
      set({ error: msg, isLoading: false })
      throw err
    }
  },

  // ── Đăng xuất ──────────────────────────────────
  logout: async () => {
    try {
      // Xoá refresh token cookie phía server
      await axios.post("/api/auth/logout")
    } finally {
      set({ accessToken: null })
    }
  },

  // ── Refresh access token ────────────────────────
  // Gọi khi access token hết hạn (401)
  // Server đọc httpOnly cookie refresh_token → trả access_token mới
  refreshToken: async () => {
    try {
      const res = await axios.post<ApiResponse<AuthTokenResponse>>(
        "/api/auth/refresh",
        {},
        {
          withCredentials: true, // gửi kèm cookie
        }
      )
      const newToken = unwrapApiResponse(res.data).access_token
      set({ accessToken: newToken })
      return newToken
    } catch {
      // Refresh token hết hạn → logout
      set({ accessToken: null })
      return null
    }
  },

  clearError: () => set({ error: null }),
}))
