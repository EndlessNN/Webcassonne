/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    // dev-режим: API і WebSocket обслуговує Go-сервер
    proxy: { '/api': { target: 'http://localhost:8090', ws: true } },
  },
  // Vitest (ADR-0006): лише чисті функції web/src/game, без DOM
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
})
