import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.join(currentDir, '..', '..')
const workflowPath = path.join(currentDir, 'ci.yml')
const scopePolicePath = path.join(currentDir, 'pr-scope-police.yml')
const autoMergeWorkflowPath = path.join(currentDir, 'auto-merge.yml')
const claudePath = path.join(repoRoot, 'CLAUDE.md')

function parseYaml(filePath) {
  const script = `
require 'yaml'
require 'json'

content = File.read(ARGV[0])
data = YAML.safe_load(content, permitted_classes: [], aliases: false)
if data.is_a?(Hash) && data.key?(true) && !data.key?('on')
  data['on'] = data.delete(true)
end
puts JSON.generate(data)
`
  const output = execFileSync('ruby', ['-e', script, filePath], {
    encoding: 'utf8',
  })
  return JSON.parse(output)
}

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

test('backend CI job runs go test and go vet via docker compose prebuilt image', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.match(
    backendJob,
    /- name: Run tests\n\s+run: docker compose run --pull never --no-deps --rm app go test \.\/\.\.\./,
  )

  assert.match(
    backendJob,
    /- name: Run vet\n\s+run: docker compose run --pull never --no-deps --rm app go vet \.\/\.\.\./,
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

test('scope gate and scope police use rename-aware allFilePaths for touches', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const scopePolice = await readFile(scopePolicePath, 'utf8')

  const renameAwarePattern =
    /allFilePaths = files\.flatMap\(\(f\) =>\s*\n\s+f\.status === 'renamed' && f\.previous_filename \? \[f\.filename, f\.previous_filename\] : \[f\.filename\]/

  assert.match(workflow, renameAwarePattern, 'ci.yml scope-gate must define allFilePaths with previous_filename support')
  assert.match(scopePolice, renameAwarePattern, 'pr-scope-police.yml must define allFilePaths with previous_filename support')

  assert.match(workflow, /touches = \{[\s\S]*?allFilePaths\.some/, 'ci.yml touches must use allFilePaths')
  assert.match(scopePolice, /touches = \{[\s\S]*?allFilePaths\.some/, 'pr-scope-police.yml touches must use allFilePaths')
  assert.match(scopePolice, /isDocsTemplateOrMetadataOnly[\s\S]*?allFilePaths\.every/, 'pr-scope-police.yml isDocsTemplateOrMetadataOnly must use allFilePaths')
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

test('docs/template-only PRs skip heavy product CI in scope gate', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(workflow, /const isDocsTemplateOrMetadataOnly =/)
  assert.match(workflow, /standardBodyValid &&\n\s+!isDocsTemplateOrMetadataOnly &&/)
  assert.match(
    workflow,
    /Skipping heavy product CI because this PR only changes docs\/templates\/metadata\./,
  )
})

test('global auto-merge workflow excludes Dependabot PRs', async () => {
  const workflow = await readFile(autoMergeWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(autoMergeWorkflowPath)

  assert.deepEqual(parsedWorkflow.on.pull_request.types, [
    'opened',
    'reopened',
    'ready_for_review',
  ])
  assert.equal(
    parsedWorkflow.jobs['enable-auto-merge'].if,
    "github.event.pull_request.draft == false && github.event.pull_request.user.login != 'dependabot[bot]'",
  )
  assert.doesNotMatch(workflow, /if: github\.event\.pull_request\.draft == false\s*$/m)
})
