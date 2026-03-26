import path from "path"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig, loadEnv } from "vite"

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, "")
  const frontendPort = Number(env.VITE_PORT || "5173")
  const apiProxyTarget = env.VITE_API_PROXY_TARGET || "http://localhost:8080"

  return {
    plugins: [tanstackRouter(), react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: frontendPort,
      host: true,
      // Dev: proxy API to Go server
      proxy: {
        "/api": {
          target: apiProxyTarget,
          ws: true,
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: "dist", // Go will serve files from dist/
    },
  }
})
