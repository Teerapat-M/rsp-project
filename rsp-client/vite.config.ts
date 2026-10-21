import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'
import { hostedClientURLError, shouldRequireHostedClientURLs } from './hostedClientURLs.ts'

export default defineConfig(({ mode }) => {
  const env = { ...loadEnv(mode, process.cwd(), ''), ...process.env }
  if (shouldRequireHostedClientURLs(env)) {
    const err = hostedClientURLError(env.VITE_API_BASE, env.VITE_WS_URL)
    if (err) throw new Error(err)
  }
  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
        },
        '/ws': {
          target: 'http://localhost:8080',
          ws: true,
        },
      },
    },
  }
})
