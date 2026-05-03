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
const autoReadyWorkflowPath = path.join(currentDir, 'auto-ready-pr.yml')
const claudePath = path.join(repoRoot, 'CLAUDE.md')
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor

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

async function runAutoReadyWorkflow({
  requiredStatusChecks = [],
  requiredStatusCheckContexts = [],
  checkRuns = [],
  statuses = [],
  graphqlError = null,
} = {}) {
  const parsedWorkflow = parseYaml(autoReadyWorkflowPath)
  const script = parsedWorkflow.jobs['auto-ready'].steps[0].with.script
  const pr = {
    number: 472,
    node_id: 'PR_node_id',
    base: { ref: 'develop' },
    draft: true,
    head: { sha: 'head_sha' },
    labels: [{ name: 'auto-ready' }],
    user: { login: 'nurockplayer' },
  }
  const notices = []
  const warnings = []
  const mutations = []
  const github = {
    rest: {
      checks: {
        listForRef: async () => checkRuns,
      },
      pulls: {
        list: async () => ({ data: [pr] }),
      },
      repos: {
        getCombinedStatusForRef: async () => ({ data: { statuses } }),
        listPullRequestsAssociatedWithCommit: async () => ({ data: [pr] }),
      },
    },
    paginate: async (fn, args) => {
      const result = await fn(args)
      return result.data || result
    },
    graphql: async (query, variables) => {
      if (query.includes('markPullRequestReadyForReview')) {
        mutations.push(variables)
        return { markPullRequestReadyForReview: { pullRequest: { number: pr.number, isDraft: false } } }
      }

      if (graphqlError) throw graphqlError
      return {
        repository: {
          ref: {
            branchProtectionRule: {
              requiredStatusChecks,
              requiredStatusCheckContexts,
            },
          },
        },
      }
    },
  }
  const core = {
    info: () => {},
    notice: (message) => notices.push(message),
    warning: (message) => warnings.push(message),
  }
  const context = {
    repo: { owner: 'nurockplayer', repo: 'tachigo' },
    eventName: 'pull_request',
    payload: { pull_request: pr },
  }

  await AsyncFunction('context', 'github', 'core', script)(context, github, core)
  return { mutations, notices, warnings }
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

test('CI workflow uses infra script entrypoints', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(workflow, /run: bash infra\/scripts\/check-backend-ci-cache\.sh/)
  assert.match(workflow, /run: bash infra\/scripts\/commit-message-check\.test\.sh/)
  assert.match(workflow, /run: bash infra\/scripts\/check-pr-commit-messages\.sh/)
  assert.doesNotMatch(workflow, /run: bash scripts\//)
})

test('backend CI job runs go test and go vet natively from services/api', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.match(
    backendJob,
    /uses: actions\/setup-go@v6\n\s+with:\n\s+go-version-file: services\/api\/go\.mod/,
  )

  assert.match(
    backendJob,
    /- name: Run tests\n\s+working-directory: services\/api\n\s+run: go test \.\/\.\.\./,
  )

  assert.match(
    backendJob,
    /- name: Run vet\n\s+working-directory: services\/api\n\s+run: go vet \.\/\.\.\./,
  )

  assert.doesNotMatch(backendJob, /actions\/download-artifact|docker load|backend-image/)
})

test('backend CI vet assertion does not match vet steps from later jobs', () => {
  const workflow = `  backend:
    steps:
      - name: Run tests
        working-directory: services/api
        run: go test ./...

  frontend:
    steps:
      - name: Run vet
        working-directory: services/api
        run: go vet ./...
`
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.doesNotMatch(
    backendJob,
    /- name: Run vet\n\s+working-directory: services\/api\n\s+run: go vet \.\/\.\.\./,
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

test('scope gate and scope police recognize legacy and monorepo frontend/backend paths', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const scopePolice = await readFile(scopePolicePath, 'utf8')

  assert.match(
    workflow,
    /frontend: allFilePaths\.some\(\(name\) =>[\s\S]*name\.startsWith\('dashboard\/'\)[\s\S]*name\.startsWith\('tachimint\/'\)[\s\S]*name\.startsWith\('apps\/dashboard\/'\)[\s\S]*name\.startsWith\('apps\/extension\/'\)/,
    'ci.yml must treat legacy and future frontend paths as one frontend surface',
  )
  assert.match(
    scopePolice,
    /frontend: allFilePaths\.some\(\(name\) =>[\s\S]*name\.startsWith\('dashboard\/'\)[\s\S]*name\.startsWith\('tachimint\/'\)[\s\S]*name\.startsWith\('apps\/dashboard\/'\)[\s\S]*name\.startsWith\('apps\/extension\/'\)/,
    'pr-scope-police.yml must treat legacy and future frontend paths as one frontend surface',
  )

  assert.match(
    workflow,
    /backend: allFilePaths\.some\(\(name\) =>[\s\S]*name\.startsWith\('backend\/'\)[\s\S]*name\.startsWith\('services\/api\/'\)/,
    'ci.yml must recognize legacy and future backend paths',
  )
  assert.match(
    scopePolice,
    /backend: allFilePaths\.some\(\(name\) =>[\s\S]*name\.startsWith\('backend\/'\)[\s\S]*name\.startsWith\('services\/api\/'\)/,
    'pr-scope-police.yml must recognize legacy and future backend paths',
  )

  assert.match(
    workflow,
    /name\.startsWith\('packages\/'\)/,
    'ci.yml must recognize future packages paths',
  )
  assert.match(
    scopePolice,
    /name\.startsWith\('packages\/'\)/,
    'pr-scope-police.yml must recognize future packages paths',
  )
})

test('backend CI uses services/api as the Go service root', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(
    workflow,
    /context: \.\/services\/api/,
    'backend image build context must use services/api',
  )
  assert.match(
    workflow,
    /go-version-file: services\/api\/go\.mod/,
    'backend integration setup-go must read services/api/go.mod',
  )
  assert.match(
    workflow,
    /working-directory: services\/api/,
    'backend integration tests must run from services/api',
  )
})

test('scope gate backend contract regex accepts full-width and half-width colons', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const scopePolice = await readFile(scopePolicePath, 'utf8')
  const backendContractYesPattern =
    String.raw`Backend contract already in develop\s*[：:][\s\S]*?- \[[xX]\] yes`
  const backendContractNoPattern =
    String.raw`Backend contract already in develop\s*[：:][\s\S]*?- \[[xX]\] no`

  assert.ok(
    workflow.includes(backendContractYesPattern),
    'ci.yml Backend contract regex must accept full-width and half-width colons',
  )
  assert.ok(
    workflow.includes(backendContractNoPattern),
    'ci.yml Backend contract regex must accept full-width and half-width colons',
  )
  assert.ok(
    scopePolice.includes(backendContractYesPattern),
    'pr-scope-police.yml Backend contract regex must accept full-width and half-width colons',
  )
  assert.ok(
    scopePolice.includes(backendContractNoPattern),
    'pr-scope-police.yml Backend contract regex must accept full-width and half-width colons',
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

test('docs/template-only PRs skip heavy product CI in scope gate', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(workflow, /const isDocsTemplateOrMetadataOnly =/)
  assert.match(
    workflow,
    /const isDocsTemplateOrMetadataOnly =\n\s+allFilePaths\.length > 0 && allFilePaths\.every\(isNonProductMetadataPath\)/,
    'ci.yml metadata-only detection must stay rename-aware via allFilePaths.every(...)',
  )
  assert.match(workflow, /standardBodyValid &&\n\s+!isDocsTemplateOrMetadataOnly &&/)
  assert.match(
    workflow,
    /Skipping heavy product CI because this PR only changes docs\/templates\/metadata\./,
  )
  assert.doesNotMatch(
    workflow,
    /name\.startsWith\('\.github\/workflows\/'\)/,
    'workflow changes must not be classified as metadata-only because they need CI validation',
  )
  assert.match(
    workflow,
    /\^\\.github\\\/workflows\\\/ci\\.yml\$/,
    'ci.yml changes must request backend integration validation',
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

test('auto-ready workflow is opt-in for draft PRs on protected base branches', async () => {
  const workflow = await readFile(autoReadyWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(autoReadyWorkflowPath)

  assert.deepEqual(parsedWorkflow.on.pull_request.branches, ['main', 'develop'])
  assert.deepEqual(parsedWorkflow.on.pull_request.types, [
    'opened',
    'synchronize',
    'reopened',
    'labeled',
  ])
  assert.deepEqual(parsedWorkflow.on.check_suite.types, ['completed'])
  assert.equal(parsedWorkflow.permissions['pull-requests'], 'write')
  assert.equal(parsedWorkflow.permissions.checks, 'read')
  assert.equal(parsedWorkflow.permissions.statuses, 'read')
  assert.match(workflow, /const autoReadyLabel = 'auto-ready'/)
  assert.match(workflow, /pr\.draft !== true/)
  assert.match(workflow, /pr\.user\?\.login === 'dependabot\[bot\]'/)
  assert.match(workflow, /targetBaseBranches\.has\(pr\.base\?\.ref\)/)
})

test('auto-ready workflow checks required contexts and excludes its own run', async () => {
  const workflow = await readFile(autoReadyWorkflowPath, 'utf8')

  assert.match(workflow, /branchProtectionRule/)
  assert.match(workflow, /requiredStatusChecks/)
  assert.match(workflow, /requiredStatusCheckContexts/)
  assert.doesNotMatch(workflow, /getBranchProtection/)
  assert.match(workflow, /listForRef/)
  assert.match(workflow, /getCombinedStatusForRef/)
  assert.match(workflow, /const ownCheckNames = new Set\(\[/)
  assert.match(workflow, /requiredChecks/)
  assert.match(workflow, /const successConclusions = new Set\(\['success', 'neutral', 'skipped'\]\)/)
  assert.doesNotMatch(workflow, /requiredSuccessConclusions/)
  assert.doesNotMatch(workflow, /fallbackSuccessConclusions/)
  assert.match(workflow, /const checkNameCandidates = \(run\) => \[run\.name\]/)
  assert.doesNotMatch(workflow, /workflow_run\?\.name/)
  assert.match(workflow, /const requiredCheckKey = \(contextName, appId\) =>/)
  assert.match(workflow, /appId \? `check:\$\{contextName\}:\$\{appId\}` : `context:\$\{contextName\}`/)
  assert.match(workflow, /const observedAt = \(item\) =>/)
  assert.match(workflow, /const recordObservedContext = \(observed, key, passed, item\) =>/)
  assert.match(workflow, /const appKey = requiredCheckKey\(name, run\.app\?\.id\)/)
  assert.match(workflow, /observed\.get\(requiredCheckKey\(requiredCheck\.context, requiredCheck\.appId\)\)\?\.passed/)
  assert.match(workflow, /try \{\s+const fetchedChecks = await Promise\.all\(\[/)
  assert.match(workflow, /core\.warning\(\s+`Skipping PR #\$\{pr\.number\} at \$\{headSha\}: failed to fetch checks\/statuses/)
  assert.match(workflow, /markPullRequestReadyForReview/)
  assert.doesNotMatch(workflow, /gh pr ready/)
})

test('auto-ready workflow uses routine logs for skipped PRs', async () => {
  const workflow = await readFile(autoReadyWorkflowPath, 'utf8')

  assert.doesNotMatch(workflow, /core\.notice\(\s*`Required status checks are not readable/)
  assert.doesNotMatch(workflow, /core\.notice\(`Required context is not successful yet:/)
  assert.doesNotMatch(workflow, /core\.notice\('No visible checks found yet\.'\)/)
  assert.doesNotMatch(workflow, /core\.notice\('At least one visible check or status is not successful yet\.'\)/)
  assert.doesNotMatch(workflow, /core\.notice\(`Skipping PR #/)
  assert.doesNotMatch(workflow, /core\.notice\(`PR #\$\{pr\.number\} remains draft until checks pass\.`\)/)
  assert.match(workflow, /core\.notice\(`PR #\$\{pr\.number\} marked ready for review\.`\)/)
})

test('auto-ready workflow serializes concurrent runs', async () => {
  const parsedWorkflow = parseYaml(autoReadyWorkflowPath)

  assert.equal(parsedWorkflow.concurrency.group, 'auto-ready-pr-${{ github.repository }}')
  assert.equal(parsedWorkflow.concurrency['cancel-in-progress'], false)
})

test('auto-ready workflow treats skipped required checks as passing', async () => {
  const result = await runAutoReadyWorkflow({
    requiredStatusChecks: [{ context: 'Docs only', app: { databaseId: 15368 } }],
    checkRuns: [
      {
        name: 'Docs only',
        status: 'completed',
        conclusion: 'skipped',
        app: { id: 15368 },
      },
    ],
  })

  assert.equal(result.mutations.length, 1)
})

test('auto-ready workflow does not let a successful status mask a failed check run with the same name', async () => {
  const result = await runAutoReadyWorkflow({
    requiredStatusCheckContexts: ['Deploy'],
    statuses: [{ context: 'Deploy', state: 'success', updated_at: '2026-05-03T00:00:00Z' }],
    checkRuns: [
      {
        name: 'Deploy',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ],
  })

  assert.equal(result.mutations.length, 0)
})

test('auto-ready workflow uses the latest rerun result for a required check', async () => {
  const failedThenPassed = await runAutoReadyWorkflow({
    requiredStatusChecks: [{ context: 'CI gate', app: { databaseId: 15368 } }],
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ],
  })
  const passedThenFailed = await runAutoReadyWorkflow({
    requiredStatusChecks: [{ context: 'CI gate', app: { databaseId: 15368 } }],
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ],
  })

  assert.equal(failedThenPassed.mutations.length, 1)
  assert.equal(passedThenFailed.mutations.length, 0)
})

test('auto-ready workflow uses the latest rerun result when no required checks are configured', async () => {
  const failedThenPassed = await runAutoReadyWorkflow({
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ],
  })
  const passedThenFailed = await runAutoReadyWorkflow({
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ],
  })

  assert.equal(failedThenPassed.mutations.length, 1)
  assert.equal(passedThenFailed.mutations.length, 0)
})

test('auto-ready workflow requires matching app id for app-scoped checks', async () => {
  const wrongApp = await runAutoReadyWorkflow({
    requiredStatusChecks: [{ context: 'CI gate', app: { databaseId: 15368 } }],
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 99999 },
      },
    ],
  })
  const matchingApp = await runAutoReadyWorkflow({
    requiredStatusChecks: [{ context: 'CI gate', app: { databaseId: 15368 } }],
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
      },
    ],
  })

  assert.equal(wrongApp.mutations.length, 0)
  assert.equal(matchingApp.mutations.length, 1)
})

test('auto-ready workflow skips instead of falling back when branch protection fetch fails', async () => {
  const result = await runAutoReadyWorkflow({
    graphqlError: new Error('Resource not accessible by integration'),
    checkRuns: [
      {
        name: 'CI gate',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
      },
    ],
  })

  assert.equal(result.mutations.length, 0)
  assert.equal(result.warnings.length, 1)
  assert.match(result.warnings[0], /failed to fetch checks\/statuses/)
})
