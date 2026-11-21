import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// Video.js 10 player tags are custom elements, not Vue components.
const videojsTags = new Set(['video-player', 'video-skin'])

export default defineConfig({
  plugins: [
    vue({
      template: {
        transformAssetUrls: {
          includeAbsolute: false,
        },
        compilerOptions: {
          isCustomElement: tag => videojsTags.has(tag),
        },
      },
    }),
  ],
  base: '/ui/',
  build: {
    outDir: 'dist',
    sourcemap: false,
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
  },
  server: {
    port: 8080,
    host: true,
    watch: {
      usePolling: true,
      interval: 500,
    },
    hmr: {
      host: 'localhost',
      port: 8080,
    },
    proxy: {
      '/api': { target: 'http://localhost:9999', changeOrigin: true },
      '/ws': { target: 'ws://localhost:9998', ws: true, changeOrigin: true },
      '/img': { target: 'http://localhost:9999', changeOrigin: true },
      '/imghm': { target: 'http://localhost:9999', changeOrigin: true },
      '/download': { target: 'http://localhost:9999', changeOrigin: true },
      '/myfiles': { target: 'http://localhost:9999', changeOrigin: true },
    },
  },
})
