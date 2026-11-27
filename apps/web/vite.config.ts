import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

const root = path.dirname(fileURLToPath(import.meta.url))

function apiHostPort(): string {
  if (process.env.API_HOST_PORT) {
    return process.env.API_HOST_PORT
  }
  try {
    const envFile = readFileSync(path.resolve(root, '../../.env'), 'utf8')
    const match = envFile.match(/^API_HOST_PORT=(.+)$/m)
    if (match?.[1]) {
      return match[1].trim()
    }
  } catch {
    // The example port is used when the local env file is absent.
  }
  return '8080'
}

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(root, 'src'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${apiHostPort()}`,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
