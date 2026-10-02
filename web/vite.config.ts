import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      // TODO(milestone 5): add PNG icons (192, 512, maskable) and fill out the manifest.
      manifest: {
        name: 'Résonance',
        short_name: 'Résonance',
        theme_color: '#ffffff',
        icons: [],
      },
      workbox: {
        // Never serve API responses from the service worker cache.
        navigateFallbackDenylist: [/^\/api\//],
      },
    }),
  ],
  server: {
    // Spotify only accepts loopback redirect URIs as 127.0.0.1, not localhost.
    host: '127.0.0.1',
    proxy: {
      // Mirrors the Ingress: /api goes to the Go service with the path unchanged.
      '/api': process.env.API_URL ?? 'http://localhost:8080',
    },
  },
})
