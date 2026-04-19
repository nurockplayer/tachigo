import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const workflowPath = path.join(currentDir, 'ci.yml')

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
