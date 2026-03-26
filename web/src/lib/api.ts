import axios from "axios"
import { useAuthStore } from "../store/auth"

export const api = axios.create({
  baseURL: "/api",
  withCredentials: true, // always send cookies
})

// No request interceptor needed — access_token cookie is sent automatically.

// ── Response interceptor: auto-refresh on 401 ───────────────────────────────
let isRefreshing = false
let failedQueue: Array<{ resolve: () => void; reject: (err: any) => void }> = []

const processQueue = (error: any) => {
  failedQueue.forEach(({ resolve, reject }) => (error ? reject(error) : resolve()))
  failedQueue = []
}

api.interceptors.response.use(
  (r) => r,
  async (error) => {
    const original = error.config
    if (error.response?.status === 401 && !original._retry) {
      if (isRefreshing) {
        return new Promise<void>((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then(() => api(original))
          .catch((err) => Promise.reject(err))
      }

      original._retry = true
      isRefreshing = true

      try {
        const ok = await useAuthStore.getState().refreshToken()
        if (!ok) {
          processQueue(new Error("Session expired"))
          return Promise.reject(error)
        }
        processQueue(null)
        return api(original) // cookie is refreshed — retry works automatically
      } catch (err) {
        processQueue(err)
        return Promise.reject(err)
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)
