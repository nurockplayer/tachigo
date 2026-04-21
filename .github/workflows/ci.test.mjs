import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.join(currentDir, '..', '..')
const workflowPath = path.join(currentDir, 'ci.yml')
const scopePolicePath = path.join(currentDir, 'pr-scope-police.yml')
const claudePath = path.join(repoRoot, 'CLAUDE.md')

test('frontend CI job runs the frontend test command', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(
    workflow,
    /frontend:\n[\s\S]*?- name: Test\n\s+run: docker compose run --no-deps --rm frontend pnpm test/,
  )

  assert.match(
    workflow,
    /workflow-regression:\n[\s\S]*?- name: Verify CI workflow assertions\n\s+run: node --test \.github\/workflows\/ci\.test\.mjs/,
  )
})

test('backend CI job runs go test and go vet', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(
    workflow,
    /backend:\n[\s\S]*?- name: Run tests\n\s+run: docker compose run --pull never --no-deps --rm app go test \.\/\.\.\./,
  )

  assert.match(
    workflow,
    /backend:\n[\s\S]*?- name: Run vet\n\s+run: docker compose run --pull never --no-deps --rm app go vet \.\/\.\.\./,
  )
})

test('PR size thresholds match CLAUDE.md', async () => {
  const claude = await readFile(claudePath, 'utf8')
  const workflow = await readFile(workflowPath, 'utf8')
  const scopePolice = await readFile(scopePolicePath, 'utf8')

  assert.match(claude, /\|\s+\*\*警告門檻\*\*\s+\|\s+600\+\s+\|/)
  assert.match(claude, /\|\s+\*\*硬限制\*\*\s+\|\s+1000\+\s+\|/)
  assert.match(claude, /\|\s+\*\*例外上限\*\*\s+\|\s+1500\s+\|/)
  assert.match(workflow, /const hardMaxDiffLines = 1000/)
  assert.match(scopePolice, /const warningDiffLines = 600/)
  assert.match(scopePolice, /const hardMaxDiffLines = 1000/)
  assert.match(scopePolice, /const exceptionMaxDiffLines = 1500/)
})
