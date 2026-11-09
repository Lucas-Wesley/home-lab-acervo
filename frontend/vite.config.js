import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.js',
      injectManifest: {
        globPatterns: ['**/*.{js,css,html,png,woff2}'],
      },
      manifest: {
        name: 'Acervo',
        short_name: 'Acervo',
        lang: 'pt-BR',
        theme_color: '#ffffff',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        // Android: o Acervo aparece em "Compartilhar"; o link chega em "/compartilhar?text=…", uma
        // tela curta que manda o link e se fecha (Compartilhar.vue).
        share_target: {
          action: '/compartilhar',
          method: 'GET',
          params: { title: 'title', text: 'text', url: 'url' },
        },
        icons: [
          { src: 'icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
    }),
  ],
  server: {
    proxy: { '/api': process.env.API_URL || 'http://localhost:8080' },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.js'],
  },
})
