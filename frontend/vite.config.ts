import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '..', '')

  return {
    envDir: '..',
    plugins: [react()],
    server: {
      proxy: {
        '/api/events': {
          target: env.VITE_API_TARGET || 'http://localhost:8080',
          ws: true,
          // Preserve Host so the backend can validate the browser's Origin.
          changeOrigin: false,
        },
        '/api': {
          target: env.VITE_API_TARGET || 'http://localhost:8080',
          changeOrigin: true,
        },
        '/health': {
          target: env.VITE_API_TARGET || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
  }
})
