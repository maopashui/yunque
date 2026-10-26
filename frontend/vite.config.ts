import path from 'node:path'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const rootDir = path.dirname(fileURLToPath(import.meta.url))

/** Vite `base` must end with `/`. Empty / `/` → site root. */
function normalizeViteBase(raw: string | undefined): string {
  const p = (raw ?? '').trim()
  if (!p || p === '/') return '/'
  const withSlash = p.startsWith('/') ? p : `/${p}`
  return withSlash.endsWith('/') ? withSlash : `${withSlash}/`
}

const base = normalizeViteBase(process.env.YUNQUE_BASE_PATH)
const apiProxyKey = base === '/' ? '/api' : `${base.replace(/\/$/, '')}/api`

export default defineConfig({
  base,
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(rootDir, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      [apiProxyKey]: {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
