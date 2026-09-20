import { mkdirSync, writeFileSync } from 'node:fs'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

// In development the Go backend runs on :8080 and Vite proxies API and auth calls to it.
const backend = process.env.SBRIGO_BACKEND_URL ?? 'http://localhost:8080'

/** web/dist is embedded by the Go binary and must exist even before the first frontend build. */
function keepDist(): Plugin {
  return {
    name: 'sbrigo-keep-dist',
    closeBundle() {
      mkdirSync('dist', { recursive: true })
      writeFileSync('dist/.gitkeep', '')
    },
  }
}

export default defineConfig({
  plugins: [
    vue(),
    keepDist(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg', 'icons/*.png'],
      manifest: {
        name: 'Sbrigo',
        short_name: 'Sbrigo',
        description: 'Liste e attività di casa, condivise in famiglia',
        lang: 'it',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        orientation: 'portrait',
        background_color: '#f6f5f1',
        theme_color: '#2f7d5b',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: '/icons/icon-512-maskable.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,webmanifest}'],
        navigateFallback: '/index.html',
        // API, auth and the SSE stream must never be answered by the service worker.
        navigateFallbackDenylist: [/^\/api\//, /^\/auth\//, /^\/healthz/],
        cleanupOutdatedCaches: true,
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/auth': { target: backend, changeOrigin: false },
    },
  },
  build: { sourcemap: false },
})
