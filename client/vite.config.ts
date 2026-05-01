import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/providers': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/output': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
