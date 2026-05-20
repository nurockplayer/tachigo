import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'

const dashboardRoot = process.cwd()
const distRoot = join(dashboardRoot, 'dist')

function writeDistWithJavascript(contents: string) {
  rmSync(distRoot, { force: true, recursive: true })
  mkdirSync(`${distRoot}/assets`, { recursive: true })
  writeFileSync(`${distRoot}/index.html`, '<script type="module" src="/assets/app.js"></script>')
  writeFileSync(`${distRoot}/assets/app.js`, contents)
}

describe('dashboard production artifact readback', () => {
  afterEach(() => {
    rmSync(distRoot, { force: true, recursive: true })
  })

  it('rejects local API URLs without an explicit port', () => {
    writeDistWithJavascript('const api = "https://api.tachigo.io"; const fallback = "https://localhost";')

    const result = spawnSync('node', ['--experimental-strip-types', 'scripts/verify-production-artifact.ts'], {
      cwd: dashboardRoot,
      encoding: 'utf8',
    })

    expect(result.status).not.toBe(0)
    expect(`${result.stdout}\n${result.stderr}`).toContain('https://localhost')
  })

  it('allows the bare localhost token bundled by browser libraries', () => {
    writeDistWithJavascript('const api = "https://api.tachigo.io"; const fallback = "http://localhost";')

    const result = spawnSync('node', ['--experimental-strip-types', 'scripts/verify-production-artifact.ts'], {
      cwd: dashboardRoot,
      encoding: 'utf8',
    })

    expect(result.status).toBe(0)
  })
})
