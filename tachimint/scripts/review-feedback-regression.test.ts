import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

async function readLoginScreen() {
  return readFile(new URL('../src/app/components/LoginScreen.tsx', import.meta.url), 'utf8')
}

async function readUseSound() {
  return readFile(new URL('../src/app/hooks/useSound.ts', import.meta.url), 'utf8')
}

test('login fields expose stable accessible names', async () => {
  const source = await readLoginScreen()

  assert.match(source, /aria-label=\{t\('login\.usernamePlaceholder'\)\}/)
  assert.match(source, /aria-label=\{t\('login\.passwordPlaceholder'\)\}/)
})

test('sound bridge falls back when content delivery reports false', async () => {
  const source = await readUseSound()

  assert.match(source, /const delivered = await sendToContentScript\(type, variant\)/)
  assert.match(source, /if \(!delivered\) \{\s*setBridgeStatus\('unsupported'\);\s*return false;\s*\}/)
})

test('background music tracks bridge and local playback reentry state', async () => {
  const source = await readUseSound()

  assert.match(source, /const bgPlayingRef = useRef\(false\)/)
  assert.match(source, /if \(bgPlayingRef\.current\) return/)
  assert.match(source, /bgPlayingRef\.current = true/)
  assert.match(source, /bgPlayingRef\.current = false/)
})
