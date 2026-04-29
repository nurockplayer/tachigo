import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

async function readManifest() {
  const raw = await readFile(new URL('../../public/manifest.json', import.meta.url), 'utf8')
  return JSON.parse(raw) as {
    host_permissions?: string[]
  }
}

async function readEnvExample() {
  return readFile(new URL('../../.env.example', import.meta.url), 'utf8')
}

test('manifest allows requests to the default local API base url', async () => {
  const manifest = await readManifest()
  const envExample = await readEnvExample()
  const apiUrl = envExample.match(/^VITE_TACHIGO_API_URL=(.+)$/m)?.[1]

  assert.equal(apiUrl, 'http://localhost:8080')
  assert.ok(manifest.host_permissions?.includes('http://localhost:8080/*'))
})
