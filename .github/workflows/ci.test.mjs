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

function workflowJobBlock(workflow, jobName) {
  const escapedJobName = jobName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const pattern = new RegExp(
    `^  ${escapedJobName}:\\n[\\s\\S]*?(?=^  [A-Za-z0-9_-]+:\\n|(?![\\s\\S]))`,
    'm',
  )
  const match = workflow.match(pattern)
  assert.ok(match, `expected workflow to include ${jobName} job`)
  return match[0]
}

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
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.match(
    backendJob,
    /- name: Run tests\n\s+working-directory: backend\n\s+run: go test \.\/\.\.\./,
  )

  assert.match(
    backendJob,
    /- name: Run vet\n\s+working-directory: backend\n\s+run: go vet \.\/\.\.\./,
  )
})

test('backend CI vet assertion does not match vet steps from later jobs', () => {
  const workflow = `  backend:
    steps:
      - name: Run tests
        run: docker compose run --pull never --no-deps --rm app go test ./...

  frontend:
    steps:
      - name: Run vet
        run: docker compose run --pull never --no-deps --rm app go vet ./...
`
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.doesNotMatch(
    backendJob,
    /- name: Run vet\n\s+run: docker compose run --pull never --no-deps --rm app go vet \.\/\.\.\./,
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
