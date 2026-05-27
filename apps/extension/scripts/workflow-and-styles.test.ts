import assert from 'node:assert/strict'
import { test } from 'vitest'
import { readFile } from 'node:fs/promises'

async function readStylesIndex() {
  return readFile(new URL('../src/styles/index.css', import.meta.url), 'utf8')
}

async function readFontStyles() {
  return readFile(new URL('../src/styles/fonts.css', import.meta.url), 'utf8')
}

test('game-pixel font stack falls back to Zpix CJK before generic monospace', async () => {
  const styles = await readStylesIndex()

  assert.match(
    styles,
    /\.game-pixel\s*\{[\s\S]*font-family:\s*var\(--pixel-font-family,\s*'Press Start 2P',\s*'Zpix CJK',\s*Zpix,\s*monospace\);/,
  )
})

test('pixel font faces load from pinned CDN without local LFS font fallback', async () => {
  const styles = await readFontStyles()

  assert.match(
    styles,
    /font-family:\s*'Press Start 2P';[\s\S]*?src:\s*url\('https:\/\/cdn\.jsdelivr\.net\/npm\/@fontsource\/press-start-2p@5\.2\.7\/files\/press-start-2p-latin-400-normal\.woff2'\)\s*format\('woff2'\);/,
  )
  assert.match(
    styles,
    /font-family:\s*Zpix;[\s\S]*?src:\s*url\('https:\/\/cdn\.jsdelivr\.net\/gh\/SolidZORO\/zpix-pixel-font@v3\.1\.11\/dist\/zpix\.ttf'\)\s*format\('truetype'\);/,
  )
  assert.match(
    styles,
    /font-family:\s*'Zpix CJK';[\s\S]*?src:\s*url\('https:\/\/cdn\.jsdelivr\.net\/gh\/SolidZORO\/zpix-pixel-font@v3\.1\.11\/dist\/zpix\.ttf'\)\s*format\('truetype'\);/,
  )
  assert.doesNotMatch(styles, /\.\.\/assets\/fonts\//)
})
