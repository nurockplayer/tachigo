import { defineConfig } from 'vite'
import path from 'path'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 350,
    rollupOptions: {
      input: {
        sidepanel: path.resolve(__dirname, 'sidepanel.html'),
        popup: path.resolve(__dirname, 'index.html'),
        background: path.resolve(__dirname, 'src/extension/background.ts'),
        content: path.resolve(__dirname, 'src/extension/content.ts'),
      },
      output: {
        entryFileNames: (chunkInfo) => (
          chunkInfo.name === 'background' || chunkInfo.name === 'content'
            ? 'assets/[name].js'
            : 'assets/[name]-[hash].js'
        ),
      },
    },
  },
  server: {
    host: true,
    port: 5173,
  },
})
