import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// 7001 default — macOS AirPlay Receiver squats on 7000 (ControlCenter).
const BACKEND = process.env.VITE_BACKEND_URL ?? 'http://localhost:7001'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: Number(process.env.MM_WEB_DEV_PORT ?? 5173),
    proxy: {
      '/api': { target: BACKEND, changeOrigin: true, ws: true },
      '/audit': { target: BACKEND, changeOrigin: true },
      '/demo': { target: BACKEND, changeOrigin: true },
      '/health': { target: BACKEND, changeOrigin: true },
      '/registry': { target: BACKEND, changeOrigin: true },
      '/sign': { target: BACKEND, changeOrigin: true },
      '/verify': { target: BACKEND, changeOrigin: true },
      '/webauthn': { target: BACKEND, changeOrigin: true },
      '/ws': { target: BACKEND, changeOrigin: true, ws: true },
    },
  },
})
