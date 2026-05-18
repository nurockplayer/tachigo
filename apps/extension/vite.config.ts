import { defineConfig } from 'vite'
import type { Plugin } from 'vite'
import fs from 'node:fs'
import path from 'path'
import react from '@vitejs/plugin-react'

const manifestTemplates = {
  dev: path.resolve(__dirname, 'manifests/dev.json'),
  production: path.resolve(__dirname, 'manifests/production.json'),
} as const

type ManifestTarget = keyof typeof manifestTemplates

function getManifestTarget(): ManifestTarget {
  const rawTarget = process.env.TACHIGO_EXTENSION_MANIFEST_TARGET ?? 'production'

  if (rawTarget === 'dev' || rawTarget === 'production') {
    return rawTarget
  }

  throw new Error(
    `Unsupported TACHIGO_EXTENSION_MANIFEST_TARGET=${rawTarget}. Use "dev" or "production".`,
  )
}

function extensionManifestPlugin(): Plugin {
  const manifestTarget = getManifestTarget()
  let outDir = path.resolve(__dirname, 'dist')

  return {
    name: 'tachigo-extension-manifest',
    apply: 'build',
    configResolved(config) {
      outDir = path.isAbsolute(config.build.outDir)
        ? config.build.outDir
        : path.resolve(config.root, config.build.outDir)
    },
    closeBundle() {
      fs.mkdirSync(outDir, { recursive: true })
      fs.copyFileSync(manifestTemplates[manifestTarget], path.join(outDir, 'manifest.json'))
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), extensionManifestPlugin()],
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
