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
const codexReviewRerequestWorkflowPath = path.join(currentDir, 'codex-review-rerequest.yml')
const closeIssueOnDevelopMergeWorkflowPath = path.join(currentDir, 'close-issue-on-develop-merge.yml')
const claudePath = path.join(repoRoot, 'CLAUDE.md')
const dependabotPolicyPath = path.join(repoRoot, 'docs', 'dependabot-update-policy.md')
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor
const developRequiredCheckRuns = [
  'Scope gate',
  'Frontend build',
  'Dashboard build',
  'Contracts build',
  'Backend CI (gate)',
].map((name, index) => ({
  name,
  status: 'completed',
  conclusion: 'success',
  app: { id: 15368 },
  completed_at: `2026-05-02T00:00:0${index}Z`,
}))

function successfulDevelopRequiredCheckRuns(...overrides) {
  return [...developRequiredCheckRuns, ...overrides]
}

const validStandardPrBody = `
refs #209

Source of truth: https://github.com/nurockplayer/tachigo/issues/209

Depends on PR: none

Backend contract already in develop:
- [x] yes
- [ ] no

本 PR 明確不做
- deploy workflow
`

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

function extractRequiredCheckSnapshots(script) {
  const marker = 'const requiredCheckSnapshots ='
  const markerIndex = script.indexOf(marker)
  assert.notEqual(markerIndex, -1, 'expected script to define requiredCheckSnapshots')

  const objectStart = script.indexOf('{', markerIndex)
  assert.notEqual(objectStart, -1, 'expected requiredCheckSnapshots to start with an object')

  let depth = 0
  let quote = null
  let escaping = false
  for (let index = objectStart; index < script.length; index += 1) {
    const char = script[index]

    if (quote) {
      if (escaping) {
        escaping = false
      } else if (char === '\\') {
        escaping = true
      } else if (char === quote) {
        quote = null
      }
      continue
    }

    if (char === "'" || char === '"' || char === '`') {
      quote = char
    } else if (char === '{') {
      depth += 1
    } else if (char === '}') {
      depth -= 1
      if (depth === 0) {
        const objectLiteral = script.slice(objectStart, index + 1)
        return Function(`'use strict'; return (${objectLiteral})`)()
      }
    }
  }

  assert.fail('expected requiredCheckSnapshots object to close')
}

async function runAutoReadyWorkflow({
  eventName = 'pull_request',
  checkRuns = [],
  statuses = [],
  checkRunsError = null,
  statusesError = null,
  graphqlError = null,
  prOverrides = {},
  livePrOverrides = {},
} = {}) {
  const parsedWorkflow = parseYaml(autoReadyWorkflowPath)
  const script = parsedWorkflow.jobs['auto-ready'].steps[0].with.script
  return runAutoReadyScript({
    script,
    eventName,
    checkRuns,
    statuses,
    checkRunsError,
    statusesError,
    graphqlError,
    prOverrides,
    livePrOverrides,
  })
}

async function runCiAutoReadyAfterCiWorkflow({
  env = {},
  ...options
} = {}) {
  const parsedWorkflow = parseYaml(workflowPath)
  const script = parsedWorkflow.jobs['auto-ready-after-ci'].steps[0].with.script
  return runAutoReadyScript({
    script,
    env: {
      SCOPE_GATE_RESULT: 'success',
      BACKEND_CI_RESULT: 'success',
      DEPENDENCY_REVIEW_RESULT: 'success',
      FRONTEND_RESULT: 'success',
      DASHBOARD_RESULT: 'success',
      CONTRACTS_RESULT: 'success',
      ...env,
    },
    ...options,
  })
}

async function runCiScopeGateWorkflow({ eventName = 'pull_request', prOverrides = {}, files = [] } = {}) {
  const parsedWorkflow = parseYaml(workflowPath)
  const script = parsedWorkflow.jobs['scope-gate'].steps[0].with.script
  const outputs = new Map()
  const notices = []
  const basePr = {
    number: 209,
    body: validStandardPrBody,
    title: '[infra] Path-aware / layered CI',
    base: { ref: 'develop' },
    head: { ref: 'chore/path-aware-layered-ci', repo: { full_name: 'nurockplayer/tachigo' } },
    labels: [],
  }
  const pr = {
    ...basePr,
    ...prOverrides,
    base: { ...basePr.base, ...(prOverrides.base || {}) },
    head: {
      ...basePr.head,
      ...(prOverrides.head || {}),
      repo: { ...basePr.head.repo, ...(prOverrides.head?.repo || {}) },
    },
    labels: prOverrides.labels || basePr.labels,
  }
  const github = {
    rest: {
      pulls: {
        listFiles: async () => ({ data: files }),
      },
    },
    paginate: async (fn, args) => {
      const result = await fn(args)
      return result.data || result
    },
  }
  const core = {
    notice: (message) => notices.push(message),
    setOutput: (name, value) => outputs.set(name, value),
  }
  const context = {
    repo: { owner: 'nurockplayer', repo: 'tachigo' },
    eventName,
    payload: eventName === 'pull_request' ? { pull_request: pr } : {},
  }

  await AsyncFunction('context', 'github', 'core', script)(context, github, core)
  return { notices, outputs: Object.fromEntries(outputs) }
}

async function runAutoReadyScript({
  script,
  eventName = 'pull_request',
  checkRuns = [],
  statuses = [],
  checkRunsError = null,
  statusesError = null,
  graphqlError = null,
  prOverrides = {},
  livePrOverrides = {},
  env = {},
} = {}) {
  const basePr = {
    number: 472,
    node_id: 'PR_node_id',
    base: { ref: 'develop' },
    draft: true,
    head: { sha: 'head_sha', repo: { full_name: 'nurockplayer/tachigo' } },
    labels: [{ name: 'auto-ready' }],
    user: { login: 'nurockplayer' },
  }

  const mergePr = (pr, overrides) => ({
    ...pr,
    ...overrides,
    base: { ...pr.base, ...(overrides.base || {}) },
    head: {
      ...pr.head,
      ...(overrides.head || {}),
      repo: { ...pr.head?.repo, ...(overrides.head?.repo || {}) },
    },
    user: { ...pr.user, ...(overrides.user || {}) },
    labels: overrides.labels || pr.labels,
  })

  const pr = mergePr(basePr, prOverrides)
  const livePr = mergePr(pr, livePrOverrides)
  const notices = []
  const warnings = []
  const mutations = []
  const autoMergeMutations = []
  const graphqlCalls = []
  const labelsAdded = []
  const labelsRemoved = []
  const labelsCreated = []
  const github = {
    rest: {
      checks: {
        listForRef: async () => {
          if (checkRunsError) throw checkRunsError
          return checkRuns
        },
      },
      pulls: {
        list: async () => ({ data: [pr] }),
        get: async () => ({ data: livePr }),
      },
      issues: {
        getLabel: async ({ name }) => ({ data: { name } }),
        createLabel: async (args) => {
          labelsCreated.push(args)
          return { data: { name: args.name } }
        },
        addLabels: async ({ issue_number, labels }) => {
          labelsAdded.push({ issue_number, labels })
          return { data: labels.map((name) => ({ name })) }
        },
        removeLabel: async ({ issue_number, name }) => {
          labelsRemoved.push({ issue_number, name })
          return { data: { name } }
        },
      },
      repos: {
        getCombinedStatusForRef: async () => {
          if (statusesError) throw statusesError
          return { data: { statuses } }
        },
        listPullRequestsAssociatedWithCommit: async () => ({ data: [pr] }),
      },
    },
    paginate: async (fn, args) => {
      const result = await fn(args)
      return result.data || result
    },
    graphql: async (query, variables) => {
      if (graphqlError) throw graphqlError
      if (query.includes('markPullRequestReadyForReview')) {
        graphqlCalls.push('ready')
        mutations.push(variables)
        return { markPullRequestReadyForReview: { pullRequest: { number: pr.number, isDraft: false } } }
      }
      if (query.includes('enablePullRequestAutoMerge')) {
        graphqlCalls.push('auto-merge')
        autoMergeMutations.push(variables)
        return {
          enablePullRequestAutoMerge: {
            pullRequest: {
              number: pr.number,
              autoMergeRequest: { mergeMethod: 'MERGE' },
            },
          },
        }
      }

      throw new Error('unexpected graphql query')
    },
  }
  const core = {
    info: () => {},
    notice: (message) => notices.push(message),
    warning: (message) => warnings.push(message),
  }
  const context = {
    repo: { owner: 'nurockplayer', repo: 'tachigo' },
    eventName,
    payload: { pull_request: pr },
  }

  const previousEnv = new Map()
  for (const [key, value] of Object.entries(env)) {
    previousEnv.set(key, process.env[key])
    process.env[key] = value
  }

  try {
    await AsyncFunction('context', 'github', 'core', script)(context, github, core)
  } finally {
    for (const [key, value] of previousEnv.entries()) {
      if (value === undefined) {
        delete process.env[key]
      } else {
        process.env[key] = value
      }
    }
  }
  return { autoMergeMutations, graphqlCalls, labelsAdded, labelsCreated, labelsRemoved, mutations, notices, warnings }
}

async function runCloseIssueOnDevelopMergeWorkflow({
  body = '',
  commits = [],
  issueStateByNumber = {},
} = {}) {
  const parsedWorkflow = parseYaml(closeIssueOnDevelopMergeWorkflowPath)
  const script = parsedWorkflow.jobs['close-linked-issues'].steps[0].with.script
  const comments = []
  const closed = []
  const notices = []
  const warnings = []
  const requestedCommits = []
  const errors = []
  const failures = []
  const context = {
    repo: { owner: 'nurockplayer', repo: 'tachigo' },
    payload: {
      pull_request: {
        number: 512,
        body,
      },
    },
  }
  const github = {
    rest: {
      issues: {
        get: async ({ issue_number }) => ({
          data: {
            number: issue_number,
            state: issueStateByNumber[issue_number] || 'open',
          },
        }),
        createComment: async (args) => {
          comments.push(args)
          return { data: { id: comments.length } }
        },
        update: async (args) => {
          closed.push(args)
          return { data: { number: args.issue_number, state: args.state } }
        },
      },
      pulls: {
        listCommits: async (args) => {
          requestedCommits.push(args)
          return { data: commits }
        },
      },
    },
    paginate: async (fn, args) => {
      const result = await fn(args)
      return result.data || result
    },
  }
  const core = {
    error: (message) => errors.push(message),
    info: () => {},
    notice: (message) => notices.push(message),
    setFailed: (message) => failures.push(message),
    warning: (message) => warnings.push(message),
  }

  await AsyncFunction('context', 'github', 'core', script)(context, github, core)
  return { closed, comments, errors, failures, notices, requestedCommits, warnings }
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
  assert.match(workflow, /run: bash infra\/scripts\/pr-open\.test\.sh/)
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

test('CI workflow validates Atlas migration tooling without applying migrations', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const atlasJob = workflowJobBlock(workflow, 'atlas-migration-tooling')

  assert.match(
    atlasJob,
    /uses: actions\/setup-go@v6\n\s+with:\n\s+go-version-file: services\/api\/go\.mod/,
  )
  assert.match(atlasJob, /uses: ariga\/setup-atlas@v0/)
  assert.match(atlasJob, /version: v1\.2\.0/)
  assert.match(
    atlasJob,
    /working-directory: services\/api\n\s+run: go run \.\/cmd\/loader\/main\.go > \/tmp\/tachigo-gorm-schema\.sql/,
  )
  assert.match(
    atlasJob,
    /atlas schema inspect --env gorm --url env:\/\/src --format '\{\{ sql \. \}\}' > \/tmp\/tachigo-atlas-inspect-schema\.sql/,
  )
  assert.match(atlasJob, /services\/api\/migrations\/.*\.sql/)
  assert.match(
    atlasJob,
    /atlas migrate lint --env gorm --git-base "origin\/\$\{\{ github\.base_ref \}\}"/,
  )
  assert.doesNotMatch(atlasJob, /atlas migrate apply|atlas schema apply|docker compose up/)
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

test('scope gate emits path-aware outputs for frontend-only PRs', async () => {
  const result = await runCiScopeGateWorkflow({
    files: [{ filename: 'apps/extension/src/App.tsx', additions: 12, deletions: 3, status: 'modified' }],
  })

  assert.deepEqual(result.outputs, {
    run_ci: 'true',
    run_backend: 'false',
    run_backend_integration: 'false',
    run_backend_scanners: 'false',
    run_dependency_review: 'false',
    run_frontend: 'true',
    run_dashboard: 'false',
    run_contracts: 'false',
  })
})

test('scope gate emits dependency review output only for dependency file PRs', async () => {
  const extensionLockfile = await runCiScopeGateWorkflow({
    files: [{ filename: 'apps/extension/pnpm-lock.yaml', additions: 9, deletions: 2, status: 'modified' }],
  })
  const rootWorkspace = await runCiScopeGateWorkflow({
    files: [{ filename: 'pnpm-workspace.yaml', additions: 1, deletions: 1, status: 'modified' }],
  })
  const frontendSource = await runCiScopeGateWorkflow({
    files: [{ filename: 'apps/extension/src/App.tsx', additions: 9, deletions: 2, status: 'modified' }],
  })

  assert.equal(extensionLockfile.outputs.run_dependency_review, 'true')
  assert.equal(rootWorkspace.outputs.run_dependency_review, 'true')
  assert.equal(frontendSource.outputs.run_dependency_review, 'false')
})

test('scope gate emits backend scanner outputs for backend PRs and scheduled scans', async () => {
  const backendPr = await runCiScopeGateWorkflow({
    files: [{ filename: 'services/api/internal/services/watch_service.go', additions: 3, deletions: 1, status: 'modified' }],
  })
  const scheduled = await runCiScopeGateWorkflow({ eventName: 'schedule' })

  assert.deepEqual(backendPr.outputs, {
    run_ci: 'true',
    run_backend: 'true',
    run_backend_integration: 'true',
    run_backend_scanners: 'true',
    run_dependency_review: 'false',
    run_frontend: 'false',
    run_dashboard: 'false',
    run_contracts: 'false',
  })
  assert.deepEqual(scheduled.outputs, {
    run_ci: 'true',
    run_backend: 'false',
    run_backend_integration: 'false',
    run_backend_scanners: 'true',
    run_dependency_review: 'false',
    run_frontend: 'false',
    run_dashboard: 'false',
    run_contracts: 'false',
  })
})

test('scope gate emits full CI outputs for push events and release promotion PRs', async () => {
  const push = await runCiScopeGateWorkflow({ eventName: 'push' })
  const releasePromotion = await runCiScopeGateWorkflow({
    prOverrides: {
      title: '[release] develop to main',
      base: { ref: 'main' },
      head: { ref: 'develop' },
      body: `
refs #209

Source of truth: https://github.com/nurockplayer/tachigo/issues/209

Depends on PR: none

Backend contract already in develop:
- [x] yes
- [ ] no

本 PR 明確不做
- production deploy
`,
    },
    files: [{ filename: 'docs/README.md', additions: 1, deletions: 0, status: 'modified' }],
  })

  const fullOutputs = {
    run_ci: 'true',
    run_backend: 'true',
    run_backend_integration: 'true',
    run_backend_scanners: 'true',
    run_dependency_review: 'false',
    run_frontend: 'true',
    run_dashboard: 'true',
    run_contracts: 'true',
  }
  assert.deepEqual(push.outputs, fullOutputs)
  assert.deepEqual(releasePromotion.outputs, fullOutputs)
  assert.equal(
    releasePromotion.notices.some((notice) => notice.includes('Skipping backend integration tests')),
    false,
  )
})

test('CI product jobs are gated by path-aware scope outputs', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const backendBuild = workflowJobBlock(workflow, 'backend-build')
  const backend = workflowJobBlock(workflow, 'backend')
  const atlas = workflowJobBlock(workflow, 'atlas-migration-tooling')
  const backendIntegration = workflowJobBlock(workflow, 'backend-integration')
  const backendSecurityScanners = workflowJobBlock(workflow, 'backend-security-scanners')
  const dependencyReview = workflowJobBlock(workflow, 'dependency-review')
  const frontend = workflowJobBlock(workflow, 'frontend')
  const dashboard = workflowJobBlock(workflow, 'dashboard')
  const contracts = workflowJobBlock(workflow, 'contracts')

  assert.match(backendBuild, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(backend, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(atlas, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(backendIntegration, /needs\.scope-gate\.outputs\.run_backend_integration == 'true'/)
  assert.match(backendSecurityScanners, /needs\.scope-gate\.outputs\.run_backend_scanners == 'true'/)
  assert.match(dependencyReview, /needs\.scope-gate\.outputs\.run_dependency_review == 'true'/)
  assert.match(frontend, /needs\.scope-gate\.outputs\.run_frontend == 'true'/)
  assert.match(dashboard, /needs\.scope-gate\.outputs\.run_dashboard == 'true'/)
  assert.match(contracts, /needs\.scope-gate\.outputs\.run_contracts == 'true'/)
})

test('backend security scanner job installs pinned staticcheck and govulncheck', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['backend-security-scanners']
  const jobBlock = workflowJobBlock(workflow, 'backend-security-scanners')
  const backendCi = parsedWorkflow.jobs['backend-ci']
  const backendCiBlock = workflowJobBlock(workflow, 'backend-ci')

  assert.deepEqual(parsedWorkflow.on.schedule, [{ cron: '17 2 * * 1' }])
  assert.equal(job.name, 'Backend security scanners')
  assert.equal(job.env.STATICCHECK_VERSION, 'v0.7.0')
  assert.equal(job.env.GOVULNCHECK_VERSION, 'v1.3.0')
  assert.match(jobBlock, /go-version: 1\.25\.9/)
  assert.match(jobBlock, /go install honnef\.co\/go\/tools\/cmd\/staticcheck@\$STATICCHECK_VERSION/)
  assert.match(jobBlock, /go install golang\.org\/x\/vuln\/cmd\/govulncheck@\$GOVULNCHECK_VERSION/)
  assert.match(jobBlock, /working-directory: services\/api\n\s+run: staticcheck \.\/\.\.\./)
  assert.match(jobBlock, /working-directory: services\/api\n\s+run: govulncheck \.\/\.\.\./)
  assert.deepEqual(backendCi.needs, [
    'backend-build',
    'backend',
    'backend-integration',
    'backend-security-scanners',
  ])
  assert.match(backendCiBlock, /BACKEND_SECURITY_SCANNERS_RESULT/)
  assert.match(backendCiBlock, /backend-security-scanners:\$BACKEND_SECURITY_SCANNERS_RESULT/)
})

test('dependency review CI job gates only frontend dependency files', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['dependency-review']
  const jobBlock = workflowJobBlock(workflow, 'dependency-review')

  assert.equal(job.name, 'Dependency Review')
  assert.equal(job['timeout-minutes'], 10)
  assert.equal(job.needs, 'scope-gate')
  assert.equal(job.if, "github.event_name == 'pull_request' && needs.scope-gate.outputs.run_dependency_review == 'true'")
  assert.match(jobBlock, /uses: actions\/checkout@v4/)
  assert.match(jobBlock, /uses: actions\/dependency-review-action@v4/)
  assert.match(jobBlock, /fail-on-severity: high/)
  assert.match(jobBlock, /fail-on-scopes: runtime/)
  assert.match(jobBlock, /vulnerability-check: true/)
  assert.match(jobBlock, /license-check: false/)
  assert.match(jobBlock, /comment-summary-in-pr: never/)
  assert.match(jobBlock, /show-openssf-scorecard: false/)
})

test('dependency review policy documents Dependabot split and waiver handling', async () => {
  const policy = await readFile(dependabotPolicyPath, 'utf8')

  assert.match(policy, /Dependency Review Gate/)
  assert.match(policy, /high\/critical production dependency vulnerabilities/)
  assert.match(policy, /development dependency findings are report-only/)
  assert.match(policy, /Dependabot opens routine version and security update PRs/)
  assert.match(policy, /False Positives And Waivers/)
  assert.match(policy, /Owner:/)
  assert.match(policy, /Expires on:/)
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

test('develop merge issue closer only runs after merged PRs close into develop', async () => {
  const parsedWorkflow = parseYaml(closeIssueOnDevelopMergeWorkflowPath)
  const job = parsedWorkflow.jobs['close-linked-issues']

  assert.deepEqual(parsedWorkflow.on.pull_request.types, ['closed'])
  assert.deepEqual(parsedWorkflow.on.pull_request.branches, ['develop'])
  assert.equal(parsedWorkflow.permissions.issues, 'write')
  assert.equal(parsedWorkflow.permissions['pull-requests'], 'read')
  assert.equal(job.if, 'github.event.pull_request.merged == true')
})

test('develop merge issue closer ignores template comments and code examples', async () => {
  const result = await runCloseIssueOnDevelopMergeWorkflow({
    body: [
      '## 為什麼',
      '<!-- 背景、需求，或關聯的 issue（e.g. closes #123） -->',
      '',
      '```',
      'closes #456',
      '```',
      '',
      '`fixes #789`',
      '',
      '實際完成 closes #494',
    ].join('\n'),
  })

  assert.deepEqual(
    result.closed.map((issue) => issue.issue_number),
    [494],
  )
  assert.deepEqual(
    result.comments.map((comment) => comment.issue_number),
    [494],
  )
})

test('develop merge issue closer reads closing keywords from PR commits', async () => {
  const result = await runCloseIssueOnDevelopMergeWorkflow({
    body: '## 為什麼\nrefs #111',
    commits: [
      { commit: { message: 'fix: old work\n\ncloses #333' } },
      { commit: { message: 'fix: prepare workflow\n\nrefs #222' } },
      { commit: { message: 'fix: close develop issue\n\ncloses #494' } },
    ],
  })

  assert.deepEqual(result.requestedCommits, [
    {
      owner: 'nurockplayer',
      repo: 'tachigo',
      pull_number: 512,
      per_page: 100,
    },
  ])
  assert.deepEqual(
    result.closed.map((issue) => issue.issue_number),
    [494],
  )
})

test('auto-ready workflow is opt-in for auto-ready PRs on protected base branches', async () => {
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
  assert.equal(parsedWorkflow.permissions.contents, 'write')
  assert.equal(parsedWorkflow.permissions.issues, 'write')
  assert.equal(parsedWorkflow.permissions.checks, 'read')
  assert.equal(parsedWorkflow.permissions.statuses, 'read')
  assert.match(workflow, /const autoReadyLabel = 'auto-ready'/)
  assert.match(workflow, /const markedReady = pr\.draft === true/)
  assert.match(workflow, /if \(markedReady\) \{[\s\S]*await readyForReview\(pr\)/)
  assert.match(workflow, /pr\.user\?\.login === 'dependabot\[bot\]'/)
  assert.match(workflow, /pr\.head\?\.repo\?\.full_name !== `\$\{owner\}\/\$\{repo\}`/)
  assert.match(workflow, /github\.rest\.pulls\.get/)
  assert.match(workflow, /pr\.head\?\.sha !== headSha/)
  assert.match(workflow, /targetBaseBranches\.has\(pr\.base\?\.ref\)/)
  assert.match(workflow, /const reviewLabel = 'needs-codex-review'/)
  assert.match(workflow, /const changesLabel = 'changes-requested'/)
})

test('auto-ready workflow checks required contexts and excludes its own run', async () => {
  const workflow = await readFile(autoReadyWorkflowPath, 'utf8')

  assert.match(workflow, /const requiredCheckSnapshots = \{/)
  assert.match(workflow, /develop: \[/)
  assert.match(workflow, /\{ context: 'Scope gate', appId: 15368 \}/)
  assert.match(workflow, /\{ context: 'Frontend build', appId: 15368 \}/)
  assert.match(workflow, /\{ context: 'Dashboard build', appId: 15368 \}/)
  assert.match(workflow, /\{ context: 'Contracts build', appId: 15368 \}/)
  assert.match(workflow, /\{ context: 'Backend CI \(gate\)', appId: 15368 \}/)
  assert.match(workflow, /main: \[/)
  assert.match(workflow, /\{ context: 'Scope police', appId: 15368 \}/)
  assert.doesNotMatch(workflow, /branchProtectionRule/)
  assert.doesNotMatch(workflow, /requiredStatusChecks/)
  assert.doesNotMatch(workflow, /requiredStatusCheckContexts/)
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
  assert.match(workflow, /enablePullRequestAutoMerge/)
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
  assert.match(workflow, /const readyMessage = markedReady \? 'marked ready for review, ' : ''/)
  assert.match(workflow, /core\.notice\(`PR #\$\{pr\.number\} \$\{readyMessage\}armed auto-merge, and flagged for Codex review\.`\)/)
})

test('CI workflow wakes auto-ready draft PRs after required CI jobs finish', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['auto-ready-after-ci']
  const jobBlock = workflowJobBlock(workflow, 'auto-ready-after-ci')

  assert.equal(job.name, 'Auto-ready draft PR after CI')
  assert.equal(job.if, "always() && github.event_name == 'pull_request'")
  assert.deepEqual(job.needs, ['scope-gate', 'backend-ci', 'dependency-review', 'frontend', 'dashboard', 'contracts'])
  assert.equal(job.permissions['pull-requests'], 'write')
  assert.equal(job.permissions.contents, 'write')
  assert.equal(job.permissions.issues, 'write')
  assert.equal(job.permissions.checks, 'read')
  assert.equal(job.permissions.statuses, 'read')

  assert.match(jobBlock, /const autoReadyLabel = 'auto-ready'/)
  assert.match(jobBlock, /const targetBaseBranches = new Set\(\['main', 'develop'\]\)/)
  assert.match(jobBlock, /const allowedJobResults = new Set\(\['success', 'skipped'\]\)/)
  assert.match(jobBlock, /const successConclusions = new Set\(\['success', 'neutral', 'skipped'\]\)/)
  assert.match(jobBlock, /SCOPE_GATE_RESULT/)
  assert.match(jobBlock, /BACKEND_CI_RESULT/)
  assert.match(jobBlock, /DEPENDENCY_REVIEW_RESULT/)
  assert.match(jobBlock, /FRONTEND_RESULT/)
  assert.match(jobBlock, /DASHBOARD_RESULT/)
  assert.match(jobBlock, /CONTRACTS_RESULT/)
  assert.match(jobBlock, /const markedReady = pr\.draft === true/)
  assert.match(jobBlock, /if \(markedReady\) \{[\s\S]*markPullRequestReadyForReview/)
  assert.match(jobBlock, /pr\.user\?\.login === 'dependabot\[bot\]'/)
  assert.match(jobBlock, /pr\.head\?\.repo\?\.full_name !== `\$\{owner\}\/\$\{repo\}`/)
  assert.match(jobBlock, /pr\.head\?\.sha !== currentHeadSha/)
  assert.match(jobBlock, /targetBaseBranches\.has\(pr\.base\?\.ref\)/)
  assert.match(jobBlock, /hasAutoReadyLabel\(pr\)/)
  assert.match(jobBlock, /const reviewLabel = 'needs-codex-review'/)
  assert.match(jobBlock, /const changesLabel = 'changes-requested'/)
  assert.match(jobBlock, /github\.rest\.issues\.addLabels/)
  assert.match(jobBlock, /github\.rest\.issues\.removeLabel/)
  assert.match(jobBlock, /github\.rest\.pulls\.get/)
  assert.match(jobBlock, /const requiredCheckSnapshots = \{/)
  assert.match(jobBlock, /\{ context: 'Scope gate', appId: 15368 \}/)
  assert.match(jobBlock, /\{ context: 'Frontend build', appId: 15368 \}/)
  assert.match(jobBlock, /\{ context: 'Dashboard build', appId: 15368 \}/)
  assert.match(jobBlock, /\{ context: 'Contracts build', appId: 15368 \}/)
  assert.match(jobBlock, /\{ context: 'Backend CI \(gate\)', appId: 15368 \}/)
  assert.match(jobBlock, /\{ context: 'Scope police', appId: 15368 \}/)
  assert.doesNotMatch(jobBlock, /branchProtectionRule/)
  assert.doesNotMatch(jobBlock, /requiredStatusChecks/)
  assert.doesNotMatch(jobBlock, /requiredStatusCheckContexts/)
  assert.match(jobBlock, /listForRef/)
  assert.match(jobBlock, /getCombinedStatusForRef/)
  assert.match(jobBlock, /markPullRequestReadyForReview/)
  assert.match(jobBlock, /enablePullRequestAutoMerge/)
})

test('CI auto-ready job marks ready before arming native auto-merge', async () => {
  const result = await runCiAutoReadyAfterCiWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
  })

  assert.deepEqual(result.graphqlCalls, ['ready', 'auto-merge'])
  assert.deepEqual(result.autoMergeMutations, [{ pullRequestId: 'PR_node_id' }])
  assert.deepEqual(result.labelsAdded, [
    { issue_number: 472, labels: ['needs-codex-review'] },
  ])
  assert.deepEqual(result.labelsRemoved, [
    { issue_number: 472, name: 'changes-requested' },
  ])
})

test('CI auto-ready job retries auto-merge for already-ready auto-ready PRs', async () => {
  const result = await runCiAutoReadyAfterCiWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
    prOverrides: { draft: false },
    livePrOverrides: { draft: false },
  })

  assert.deepEqual(result.graphqlCalls, ['auto-merge'])
  assert.equal(result.mutations.length, 0)
  assert.deepEqual(result.autoMergeMutations, [{ pullRequestId: 'PR_node_id' }])
  assert.deepEqual(result.labelsAdded, [
    { issue_number: 472, labels: ['needs-codex-review'] },
  ])
  assert.deepEqual(result.labelsRemoved, [
    { issue_number: 472, name: 'changes-requested' },
  ])
})

test('CI auto-ready job waits when dependency review fails', async () => {
  const result = await runCiAutoReadyAfterCiWorkflow({
    env: { DEPENDENCY_REVIEW_RESULT: 'failure' },
    checkRuns: successfulDevelopRequiredCheckRuns(),
  })

  assert.deepEqual(result.graphqlCalls, [])
  assert.deepEqual(result.labelsAdded, [])
})

test('auto-ready workflow arms native auto-merge after marking a PR ready', async () => {
  const result = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
  })

  assert.equal(result.mutations.length, 1)
  assert.deepEqual(result.graphqlCalls, ['ready', 'auto-merge'])
  assert.deepEqual(result.autoMergeMutations, [{ pullRequestId: 'PR_node_id' }])
})

test('auto-ready workflow retries auto-merge for already-ready auto-ready PRs', async () => {
  const result = await runAutoReadyWorkflow({
    eventName: 'schedule',
    checkRuns: successfulDevelopRequiredCheckRuns(),
    prOverrides: { draft: false },
    livePrOverrides: { draft: false },
  })

  assert.deepEqual(result.graphqlCalls, ['auto-merge'])
  assert.equal(result.mutations.length, 0)
  assert.deepEqual(result.autoMergeMutations, [{ pullRequestId: 'PR_node_id' }])
  assert.deepEqual(result.labelsAdded, [
    { issue_number: 472, labels: ['needs-codex-review'] },
  ])
  assert.deepEqual(result.labelsRemoved, [
    { issue_number: 472, name: 'changes-requested' },
  ])
})

test('auto-ready required-check snapshots stay aligned across workflows', () => {
  const standaloneScript =
    parseYaml(autoReadyWorkflowPath).jobs['auto-ready'].steps[0].with.script
  const ciScript =
    parseYaml(workflowPath).jobs['auto-ready-after-ci'].steps[0].with.script

  assert.deepEqual(
    extractRequiredCheckSnapshots(standaloneScript),
    extractRequiredCheckSnapshots(ciScript),
  )
})

test('auto-ready workflow serializes concurrent runs', async () => {
  const parsedWorkflow = parseYaml(autoReadyWorkflowPath)

  assert.equal(parsedWorkflow.concurrency.group, 'auto-ready-pr-${{ github.repository }}')
  assert.equal(parsedWorkflow.concurrency['cancel-in-progress'], false)
})

test('auto-ready workflow treats skipped required checks as passing', async () => {
  const result = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(
      {
        name: 'Scope gate',
        status: 'completed',
        conclusion: 'skipped',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
    ),
  })

  assert.equal(result.mutations.length, 1)
})

test('auto-ready workflow flags ready PRs for Codex review', async () => {
  const result = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
  })

  assert.equal(result.mutations.length, 1)
  assert.deepEqual(result.labelsAdded, [
    { issue_number: 472, labels: ['needs-codex-review'] },
  ])
  assert.deepEqual(result.labelsRemoved, [
    { issue_number: 472, name: 'changes-requested' },
  ])
})

test('auto-ready workflow refreshes live PR state before marking ready', async () => {
  const staleHead = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
    livePrOverrides: { head: { sha: 'new_head_sha' } },
  })
  const labelRemoved = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(),
    livePrOverrides: { labels: [] },
  })

  assert.equal(staleHead.mutations.length, 0)
  assert.equal(staleHead.labelsAdded.length, 0)
  assert.equal(labelRemoved.mutations.length, 0)
  assert.equal(labelRemoved.labelsAdded.length, 0)
})

test('auto-ready workflow does not let a successful status mask a failed check run with the same name', async () => {
  const result = await runAutoReadyWorkflow({
    statuses: [{ context: 'Scope gate', state: 'success', updated_at: '2026-05-03T00:00:00Z' }],
    checkRuns: successfulDevelopRequiredCheckRuns(
      {
        name: 'Scope gate',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ),
  })

  assert.equal(result.mutations.length, 0)
})

test('auto-ready workflow uses the latest rerun result for a required check', async () => {
  const failedThenPassed = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ),
  })
  const passedThenFailed = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:00:00Z',
      },
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'failure',
        app: { id: 15368 },
        completed_at: '2026-05-03T00:01:00Z',
      },
    ),
  })

  assert.equal(failedThenPassed.mutations.length, 1)
  assert.equal(passedThenFailed.mutations.length, 0)
})

test('auto-ready workflow requires matching app id for app-scoped checks', async () => {
  const wrongApp = await runAutoReadyWorkflow({
    checkRuns: [
      ...successfulDevelopRequiredCheckRuns().filter((run) => run.name !== 'Backend CI (gate)'),
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'success',
        app: { id: 99999 },
      },
    ],
  })
  const matchingApp = await runAutoReadyWorkflow({
    checkRuns: successfulDevelopRequiredCheckRuns(
      {
        name: 'Backend CI (gate)',
        status: 'completed',
        conclusion: 'success',
        app: { id: 15368 },
      },
    ),
  })

  assert.equal(wrongApp.mutations.length, 0)
  assert.equal(matchingApp.mutations.length, 1)
})

test('auto-ready workflow skips when check/status lookup fails', async () => {
  const result = await runAutoReadyWorkflow({
    checkRunsError: new Error('checks unavailable'),
  })

  assert.equal(result.mutations.length, 0)
  assert.equal(result.warnings.length, 1)
  assert.match(result.warnings[0], /failed to fetch checks\/statuses/)
})

test('Codex review re-request workflow requests reviewer and notifies Discord', async () => {
  const workflow = await readFile(codexReviewRerequestWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(codexReviewRerequestWorkflowPath)

  assert.equal(parsedWorkflow.name, 'Auto re-request Codex review')
  assert.equal(parsedWorkflow.permissions['pull-requests'], 'write')
  assert.match(workflow, /github\.rest\.pulls\.listReviews/)
  assert.match(workflow, /github\.rest\.pulls\.requestReviewers/)
  assert.match(workflow, /for \(const reviewer of reviewers\)/)
  assert.match(workflow, /reviewers: \[reviewer\]/)
  assert.match(workflow, /has_requested_reviewers/)
  assert.match(
    workflow,
    /if: steps\.dedup-cache\.outputs\.cache-hit != 'true' && steps\.rerequest-review\.outputs\.has_requested_reviewers == 'true'/,
  )
  assert.match(
    workflow,
    /PR #\$\{fullPr\.number\} has no previous reviewers to re-request\.`\)\n\s+return null/,
  )
  assert.match(workflow, /Previous reviewers/)
  assert.match(workflow, /DISCORD_CODEX_REVIEW_WEBHOOK_URL/)
  assert.match(workflow, /codex-review-rerequest/)
  assert.doesNotMatch(workflow, /nurockplayer/)
  assert.doesNotMatch(workflow, /Slack|SLACK|slack/)
})
