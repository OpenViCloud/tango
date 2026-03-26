import React from "react"
import ReactDOM from "react-dom/client"
import { RouterProvider } from "@tanstack/react-router"
import {
  MutationCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query"
import { Toaster, toast } from "sonner"
import { ThemeProvider } from "@/components/theme-provider"
import "@/i18n"
import { getErrorMessage } from "@/lib/get-error-message"
import { useAuthStore } from "./store/auth"
import "./index.css"
import { router } from "./router"

const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      const meta = mutation.options.meta as
        | { skipErrorToast?: boolean }
        | undefined

      if (meta?.skipErrorToast) {
        return
      }

      toast.error(getErrorMessage(error))
    },
  }),
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000 },
  },
})

async function bootstrap() {
  // Thử refresh token trước khi render
  // Nếu có cookie hợp lệ → lấy lại access token
  // Nếu không → để null, router tự redirect về login
  await useAuthStore.getState().refreshToken()

  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
          <Toaster position="top-right" richColors />
        </QueryClientProvider>
      </ThemeProvider>
    </React.StrictMode>
  )
}

bootstrap()
