import { defineConfig } from 'vite'
import path from 'path'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },

  build: {
    rollupOptions: {
      input: {
        sidepanel: path.resolve(__dirname, 'sidepanel.html'),
        background: path.resolve(__dirname, 'src/extension/background.ts'),
      },
      output: {
        entryFileNames: (chunkInfo) => (
          chunkInfo.name === 'background'
            ? 'assets/background.js'
            : 'assets/[name]-[hash].js'
        ),
      },
    },
  },

  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
  },
  assetsInclude: ['**/*.svg', '**/*.csv'],
})
