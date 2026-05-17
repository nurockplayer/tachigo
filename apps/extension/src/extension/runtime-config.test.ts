import assert from 'node:assert/strict'
import { test } from 'vitest'
import { readFile } from 'node:fs/promises'

interface ExtensionManifest {
  background?: {
    service_worker?: string
  }
  content_scripts?: Array<{
    matches?: string[]
    js?: string[]
  }>
  host_permissions?: string[]
  side_panel?: {
    default_path?: string
  }
}

async function readManifest(target: 'dev' | 'production') {
  const raw = await readFile(new URL(`../../../manifests/${target}.json`, import.meta.url), 'utf8')
  return JSON.parse(raw) as ExtensionManifest
}

async function readDevEnvExample() {
  return readFile(new URL('../../.env.example', import.meta.url), 'utf8')
}

async function readProductionEnvExample() {
  return readFile(new URL('../../.env.production.example', import.meta.url), 'utf8')
}

function readEnvValue(envExample: string, name: string) {
  return envExample.match(new RegExp(`^${name}=(.+)$`, 'm'))?.[1]
}

function manifestUrls(manifest: ExtensionManifest) {
  return [
    ...(manifest.host_permissions ?? []),
    ...(manifest.content_scripts ?? []).flatMap((script) => script.matches ?? []),
  ]
}

function assertManifestEntrypoints(manifest: ExtensionManifest) {
  assert.equal(manifest.background?.service_worker, 'assets/background.js')
  assert.equal(manifest.side_panel?.default_path, 'sidepanel.html')
  assert.ok(
    manifest.content_scripts?.some((script) => script.js?.includes('assets/content.js')),
    'manifest should include the content script bundle entry',
  )
}

test('dev manifest allows requests to the default local API base url', async () => {
  const manifest = await readManifest('dev')
  const envExample = await readDevEnvExample()
  const apiUrl = readEnvValue(envExample, 'VITE_TACHIGO_API_URL')

  assert.equal(apiUrl, 'http://localhost:8080')
  assert.ok(manifest.host_permissions?.includes('http://localhost:8080/*'))
  assert.ok(manifest.content_scripts?.some((script) => script.matches?.includes('http://localhost:3000/*')))
  assertManifestEntrypoints(manifest)
})

test('production manifest targets Twitch and tachigo API without localhost permissions', async () => {
  const manifest = await readManifest('production')
  const envExample = await readProductionEnvExample()
  const apiUrl = readEnvValue(envExample, 'VITE_TACHIGO_API_URL')

  assert.equal(apiUrl, 'https://api.tachigo.io')
  assert.ok(manifest.host_permissions?.includes('https://api.tachigo.io/*'))
  assert.ok(manifest.content_scripts?.some((script) => script.matches?.includes('https://www.twitch.tv/*')))
  assertManifestEntrypoints(manifest)

  const urls = manifestUrls(manifest)
  assert.equal(
    urls.some((url) => /localhost|127\.0\.0\.1|0\.0\.0\.0/.test(url)),
    false,
    `production manifest should not include local-only URLs: ${urls.join(', ')}`,
  )
})
