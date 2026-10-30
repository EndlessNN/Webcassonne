import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    // dev-режим: API і WebSocket обслуговує Go-сервер
    proxy: { '/api': { target: 'http://localhost:8090', ws: true } },
  },
})
