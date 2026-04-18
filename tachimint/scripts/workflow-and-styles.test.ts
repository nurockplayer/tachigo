import assert from 'node:assert/strict'
import { access } from 'node:fs/promises'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

async function hasCiWorkflow() {
  try {
    await access(new URL('../../.github/workflows/ci.yml', import.meta.url))
    return true
  } catch {
    return false
  }
}

async function readCiWorkflow() {
  return readFile(new URL('../../.github/workflows/ci.yml', import.meta.url), 'utf8')
}

async function readStylesIndex() {
  return readFile(new URL('../src/styles/index.css', import.meta.url), 'utf8')
}

test('backend integration job keeps an explicit pull_request CI gate', async (t) => {
  if (!(await hasCiWorkflow())) {
    t.skip('CI workflow file is not present in the frontend package build context')
    return
  }

  const workflow = await readCiWorkflow()

  assert.match(
    workflow,
    /backend-integration:[\s\S]*needs:\s*\[scope-gate,\s*backend\][\s\S]*if:\s*github\.event_name == 'push' \|\| \(needs\.scope-gate\.outputs\.run_ci == 'true' && needs\.scope-gate\.outputs\.run_backend_integration == 'true'\)/,
  )
})

test('game-pixel font stack falls back to Zpix CJK before generic monospace', async () => {
  const styles = await readStylesIndex()

  assert.match(
    styles,
    /\.game-pixel\s*\{[\s\S]*font-family:\s*var\(--pixel-font-family,\s*'Press Start 2P',\s*'Zpix CJK',\s*Zpix,\s*monospace\);/,
  )
})
