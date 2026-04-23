import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    watch: {
      usePolling: true,
      interval: 1000,
    },
    proxy: {
      // Shell API routes to Python Exp service
      '/api/v1/shell': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      // VShell API routes to Python Exp service
      '/api/v1/vshell': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      // Exploit API routes to Python Exp service
      '/api/v1/exploits': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      '/api/v1/check': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      '/api/v1/execute': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      '/api/v1/health': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
      // All other API routes to Go service
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
