import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ command, mode }) => {
  const environment = loadEnv(mode, '..', '')
  const proxyTarget = environment.VITE_BACKEND_PROXY_TARGET
  const apiBaseURL = environment.VITE_API_BASE_URL

  if (command === 'serve' && !proxyTarget) {
    throw new Error('VITE_BACKEND_PROXY_TARGET environment variable is required')
  }
  if (!apiBaseURL) {
    throw new Error('VITE_API_BASE_URL environment variable is required')
  }

  return {
    envDir: '..',
    plugins: [react(), tailwindcss()],
    server:
      command === 'serve'
        ? {
            proxy: {
              [apiBaseURL]: {
                target: proxyTarget,
                changeOrigin: true,
              },
            },
          }
        : undefined,
  }
})
