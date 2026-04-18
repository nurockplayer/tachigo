import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

async function readStylesIndex() {
  return readFile(new URL('../src/styles/index.css', import.meta.url), 'utf8')
}

test('game-pixel font stack falls back to Zpix CJK before generic monospace', async () => {
  const styles = await readStylesIndex()

  assert.match(
    styles,
    /\.game-pixel\s*\{[\s\S]*font-family:\s*var\(--pixel-font-family,\s*'Press Start 2P',\s*'Zpix CJK',\s*Zpix,\s*monospace\);/,
  )
})
