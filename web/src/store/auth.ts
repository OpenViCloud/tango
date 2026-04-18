import { create } from "zustand"
import axios from "axios"
import { getErrorMessage } from "@/lib/get-error-message"
import type { RegisterRequestModel } from "@/@types/models/auth"
import { authService } from "@/services/api/auth-service"

interface AuthState {
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  init: () => Promise<boolean>
  login: (email: string, password: string) => Promise<void>
  register: (payload: RegisterRequestModel) => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<boolean>
  clearError: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: false,
  error: null,

  // Called on page load — tries to restore session from refresh cookie
  init: async () => {
    try {
      await axios.post("/api/auth/refresh", {}, { withCredentials: true })
      set({ isAuthenticated: true })
      return true
    } catch {
      set({ isAuthenticated: false })
      return false
    }
  },

  register: async (payload) => {
    set({ isLoading: true, error: null })
    try {
      await authService.register(payload)
      set({ isAuthenticated: true, isLoading: false })
    } catch (err: any) {
      set({
        error: getErrorMessage(err),
        isAuthenticated: false,
        isLoading: false,
      })
      throw err
    }
  },

  login: async (email, password) => {
    set({ isLoading: true, error: null })
    try {
      await axios.post("/api/auth/login", { email, password }, { withCredentials: true })
      set({ isAuthenticated: true, isLoading: false })
    } catch (err: any) {
      set({
        error: getErrorMessage(err),
        isAuthenticated: false,
        isLoading: false,
      })
      throw err
    }
  },

  logout: async () => {
    try {
      await axios.post("/api/auth/logout", {}, { withCredentials: true })
    } finally {
      set({ isAuthenticated: false })
    }
  },

  // Called by the 401 interceptor — renews the access_token cookie
  refreshToken: async () => {
    try {
      await axios.post("/api/auth/refresh", {}, { withCredentials: true })
      set({ isAuthenticated: true })
      return true
    } catch {
      set({ isAuthenticated: false })
      return false
    }
  },

  clearError: () => set({ error: null }),
}))
