import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'node',
    env: {
      VITE_TACHIGO_API_URL: 'http://localhost:8080',
    },
    include: ['src/**/*.test.{ts,tsx}', 'scripts/*.test.{ts,tsx}'],
  },
})
