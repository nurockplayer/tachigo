import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.join(currentDir, '..', '..')
const workflowPath = path.join(currentDir, 'ci.yml')
const dockerComposePath = path.join(repoRoot, 'docker-compose.yml')
const dockerComposeOverridePath = path.join(repoRoot, 'docker-compose.override.yml')
const backendDockerfilePath = path.join(repoRoot, 'services', 'api', 'Dockerfile')
const backendMakefilePath = path.join(repoRoot, 'services', 'api', 'Makefile')
const backendDockerEntrypointPath = path.join(repoRoot, 'services', 'api', 'docker-entrypoint.sh')
const scopePolicePath = path.join(currentDir, 'pr-scope-police.yml')
const dependabotConfigPath = path.join(repoRoot, '.github', 'dependabot.yml')
const dependabotAutomergeWorkflowPath = path.join(currentDir, 'dependabot-automerge.yml')
const dependabotPnpmLockfileWorkflowPath = path.join(currentDir, 'dependabot-pnpm-lockfile.yml')
const autoMergeWorkflowPath = path.join(currentDir, 'auto-merge.yml')
const autoReadyWorkflowPath = path.join(currentDir, 'auto-ready-pr.yml')
const codexReviewRerequestWorkflowPath = path.join(currentDir, 'codex-review-rerequest.yml')
const closeIssueOnDevelopMergeWorkflowPath = path.join(currentDir, 'close-issue-on-develop-merge.yml')
const dependencyInventoryWorkflowPath = path.join(currentDir, 'dependency-inventory.yml')
const notifyRebaseNeededWorkflowPath = path.join(currentDir, 'notify-rebase-needed.yml')
const releasePrWorkflowPath = path.join(currentDir, 'release-pr.yml')
const claudePath = path.join(repoRoot, 'CLAUDE.md')
const prScopePolicyPath = path.join(repoRoot, 'docs', 'pr-scope-policy.md')
const dependabotPolicyPath = path.join(repoRoot, 'docs', 'dependabot-update-policy.md')
const securityScannerEvaluationPath = path.join(repoRoot, 'docs', 'security-scanner-evaluation.md')
const contractsGasSnapshotPolicyPath = path.join(repoRoot, 'docs', 'contracts-gas-snapshot-policy.md')
const dependencyInventoryPolicyPath = path.join(repoRoot, 'docs', 'dependency-inventory-policy.md')
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

function workflowJobStep(parsedWorkflow, jobName, stepName) {
  const job = parsedWorkflow.jobs[jobName]
  assert.ok(job, `expected workflow to include ${jobName} job`)
  const step = job.steps.find((candidate) => candidate.name === stepName)
  assert.ok(step, `expected ${jobName} to include ${stepName} step`)
  return step
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function pinnedActionRef(actionName, versionLabel) {
  return new RegExp(`uses: ${escapeRegExp(actionName)}@[0-9a-f]{40} # ${escapeRegExp(versionLabel)}`)
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
      CONTRACTS_SLITHER_RESULT: 'success',
      CONTRACTS_GAS_SNAPSHOT_RESULT: 'success',
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

async function runNotifyRebaseNeededWorkflow({
  mergedPrOverrides = {},
  openPRs = [],
  freshPRsByNumber = {},
  commentsByIssueNumber = {},
} = {}) {
  const parsedWorkflow = parseYaml(notifyRebaseNeededWorkflowPath)
  const script = parsedWorkflow.jobs.notify.steps[0].with.script
  const commentsCreated = []
  const commentsListed = []
  const infos = []
  const mergedPR = {
    number: 600,
    title: 'Merged backend fix',
    html_url: 'https://github.com/nurockplayer/tachigo/pull/600',
    ...mergedPrOverrides,
  }
  const github = {
    rest: {
      pulls: {
        list: async () => ({ data: openPRs }),
        get: async ({ pull_number }) => ({
          data: freshPRsByNumber[pull_number] || openPRs.find((pr) => pr.number === pull_number),
        }),
      },
      issues: {
        listComments: async (args) => {
          commentsListed.push(args)
          return { data: commentsByIssueNumber[args.issue_number] || [] }
        },
        createComment: async (args) => {
          commentsCreated.push(args)
          return { data: { id: commentsCreated.length } }
        },
      },
    },
    paginate: async (fn, args) => {
      const result = await fn(args)
      return result.data || result
    },
  }
  const core = {
    info: (message) => infos.push(message),
  }
  const context = {
    repo: { owner: 'nurockplayer', repo: 'tachigo' },
    payload: { pull_request: mergedPR },
  }

  await AsyncFunction('context', 'github', 'core', 'setTimeout', script)(
    context,
    github,
    core,
    (callback) => {
      callback()
      return 0
    },
  )

  return { commentsCreated, commentsListed, infos }
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

test('CI workflow pins action references to full commit SHAs', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const actionRefs = [...workflow.matchAll(/uses:\s+([^@\s#]+)@([^\s#]+)(?:\s+#\s+([^\n]+))?/g)]

  assert.ok(actionRefs.length > 0)

  for (const [, actionName, ref, versionLabel] of actionRefs) {
    assert.match(ref, /^[0-9a-f]{40}$/, `${actionName} must use a full 40-character SHA`)
    assert.ok(versionLabel?.startsWith('v'), `${actionName} must keep the original version tag as a comment`)
  }
})

test('backend CI job runs go test and go vet natively from services/api', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const backendJob = workflowJobBlock(workflow, 'backend')

  assert.match(
    backendJob,
    /uses: actions\/setup-go@[0-9a-f]{40} # v6\n\s+with:\n\s+go-version-file: services\/api\/go\.mod/,
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

test('CI workflow validates Atlas migrations against ephemeral PostgreSQL', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const atlasJob = workflowJobBlock(workflow, 'atlas-migration-tooling')

  assert.match(
    atlasJob,
    /uses: actions\/setup-go@[0-9a-f]{40} # v6\n\s+with:\n\s+go-version-file: services\/api\/go\.mod/,
  )
  assert.match(atlasJob, /services:\n\s+postgres:\n\s+image: postgres:16-alpine/)
  assert.match(atlasJob, /POSTGRES_DB: tachigo/)
  assert.match(atlasJob, /5432:5432/)
  assert.match(
    atlasJob,
    /--health-cmd "pg_isready -U postgres -d tachigo"/,
  )
  assert.match(atlasJob, pinnedActionRef('ariga/setup-atlas', 'v0'))
  assert.match(
    atlasJob,
    /working-directory: services\/api\n\s+run: go run \.\/cmd\/loader\/main\.go > \/tmp\/tachigo-gorm-schema\.sql/,
  )
  assert.match(
    atlasJob,
    /atlas schema inspect --env gorm --url env:\/\/src --format '\{\{ sql \. \}\}' > \/tmp\/tachigo-atlas-inspect-schema\.sql/,
  )
  assert.match(
    atlasJob,
    /Reject destructive migration statements without nolint/,
  )
  assert.match(
    atlasJob,
    /atlas migrate apply --dir file:\/\/migrations --url "postgres:\/\/postgres:postgres@localhost:5432\/tachigo\?sslmode=disable"/,
  )
  assert.match(
    atlasJob,
    /Apply migrations to GORM-shape baseline/,
  )
  assert.match(
    atlasJob,
    /GORM_BASELINE_DATABASE_URL: postgres:\/\/postgres:postgres@localhost:5432\/tachigo_gorm_baseline\?sslmode=disable/,
  )
  assert.match(
    atlasJob,
    /psql "postgres:\/\/postgres:postgres@localhost:5432\/postgres\?sslmode=disable" -v ON_ERROR_STOP=1 -c "CREATE DATABASE tachigo_gorm_baseline"/,
  )
  assert.match(
    atlasJob,
    /psql "\$GORM_BASELINE_DATABASE_URL" -v ON_ERROR_STOP=1 -f \/tmp\/tachigo-gorm-schema\.sql/,
  )
  assert.match(
    atlasJob,
    /atlas migrate apply --dir file:\/\/migrations --url "\$GORM_BASELINE_DATABASE_URL" --baseline 019/,
  )
  assert.doesNotMatch(atlasJob, /--community|\/tmp\/atlas-community|migrate lint|--git-base|docker:\/\/postgres\/15/)
  assert.doesNotMatch(atlasJob, /version: v0\.37\.0|version: v1\.2\.0/)
  assert.doesNotMatch(atlasJob, /atlas schema apply|docker compose up/)
})

test('Atlas destructive migration guard blocks high-risk schema rewrites without nolint', async () => {
  const parsedWorkflow = parseYaml(workflowPath)
  const step = workflowJobStep(parsedWorkflow, 'atlas-migration-tooling', 'Reject destructive migration statements without nolint')
  const tempDir = await mkdtemp(path.join(tmpdir(), 'tachigo-atlas-guard-'))
  const migrationsDir = path.join(tempDir, 'migrations')

  await mkdir(migrationsDir)
  try {
    const cases = [
      ['drop_index', 'DROP INDEX idx_users_email;'],
      ['drop_constraint', 'ALTER TABLE users DROP CONSTRAINT users_email_key;'],
      ['rename_column', 'ALTER TABLE users RENAME COLUMN email TO login_email;'],
      ['rename_table', 'ALTER TABLE users RENAME TO app_users;'],
      ['alter_column_type', 'ALTER TABLE users ALTER COLUMN score TYPE bigint;'],
      ['alter_column_set_data_type', 'ALTER TABLE users ALTER COLUMN score SET DATA TYPE bigint;'],
    ]

    await writeFile(path.join(migrationsDir, 'comment_only.sql'), '-- DROP CONSTRAINT is documentation only\n')
    assert.doesNotThrow(() => {
      execFileSync('sh', ['-c', step.run], { cwd: tempDir, encoding: 'utf8' })
    })

    // Mirrors the narrow CI allowlist so old applied migrations stay hash-stable.
    const legacyAuthProviderMigration = '014_auth_provider_partial_unique.sql'
    const legacyClaimStatusMigration = '017_claim_finalize_failed.sql'
    const legacyAuthProviderConstraint = 'auth_providers_provider_provider_id_key'
    const legacyClaimStatusConstraint = 'claims_status_check'
    const legacyClaimStatusCheckConstraint = 'chk_claim_status'
    const prefixedLegacyConstraint = `${legacyAuthProviderConstraint}_extra`

    await writeFile(
      path.join(migrationsDir, legacyAuthProviderMigration),
      `ALTER TABLE auth_providers\n    DROP CONSTRAINT IF EXISTS ${legacyAuthProviderConstraint};\n`,
    )
    await writeFile(
      path.join(migrationsDir, legacyClaimStatusMigration),
      [
        "EXECUTE format('ALTER TABLE claims DROP CONSTRAINT %I', constraint_name);",
        `ALTER TABLE claims DROP CONSTRAINT IF EXISTS ${legacyClaimStatusConstraint};`,
        `ALTER TABLE claims DROP CONSTRAINT IF EXISTS ${legacyClaimStatusCheckConstraint};`,
        '',
      ].join('\n'),
    )
    assert.doesNotThrow(() => {
      execFileSync('sh', ['-c', step.run], { cwd: tempDir, encoding: 'utf8' })
    })

    await writeFile(
      path.join(migrationsDir, legacyAuthProviderMigration),
      `ALTER TABLE auth_providers DROP CONSTRAINT IF EXISTS ${prefixedLegacyConstraint};\n`,
    )
    assert.throws(() => {
      execFileSync('sh', ['-c', step.run], { cwd: tempDir, encoding: 'utf8' })
    })
    await writeFile(
      path.join(migrationsDir, legacyAuthProviderMigration),
      `ALTER TABLE auth_providers\n    DROP CONSTRAINT IF EXISTS ${legacyAuthProviderConstraint};\n`,
    )

    for (const [name, sql] of cases) {
      const migrationPath = path.join(migrationsDir, `${name}.sql`)
      await writeFile(migrationPath, `${sql}\n`)

      let failure
      try {
        execFileSync('sh', ['-c', step.run], { cwd: tempDir, encoding: 'utf8' })
      } catch (error) {
        failure = error
      }

      assert.ok(failure, `${name} should require -- atlas:nolint`)
      assert.match(failure.stdout, /destructive migration requires -- atlas:nolint/)

      await writeFile(migrationPath, `-- atlas:nolint\n${sql}\n`)
      assert.doesNotThrow(() => {
        execFileSync('sh', ['-c', step.run], { cwd: tempDir, encoding: 'utf8' })
      })
    }
  } finally {
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('backend Docker runtime applies Atlas migrations before starting the API', async () => {
  const dockerfile = await readFile(backendDockerfilePath, 'utf8')
  const makefile = await readFile(backendMakefilePath, 'utf8')
  const entrypoint = await readFile(backendDockerEntrypointPath, 'utf8')
  const compose = await readFile(dockerComposePath, 'utf8')
  const composeOverride = await readFile(dockerComposeOverridePath, 'utf8')

  assert.match(dockerfile, /ARG ATLAS_VERSION=1\.2\.0\nFROM arigaio\/atlas:\$\{ATLAS_VERSION\} AS atlas/)
  assert.doesNotMatch(dockerfile, /FROM arigaio\/atlas:latest/)
  assert.equal(
    (dockerfile.match(/COPY --from=atlas \/atlas \/usr\/local\/bin\/atlas/g) ?? []).length,
    2,
    'both dev and runtime Docker stages must include the Atlas CLI',
  )
  assert.match(dockerfile, /COPY --from=builder \/app\/migrations \.\/migrations/)
  assert.equal(
    (dockerfile.match(/ENTRYPOINT \["\/docker-entrypoint\.sh"\]/g) ?? []).length,
    2,
    'both dev and runtime Docker stages must run the migration entrypoint',
  )
  assert.match(dockerfile, /mkdir -p \/home\/nonroot/)
  assert.match(dockerfile, /CMD \["\/tachigo"\]/)

  assert.match(entrypoint, /ATLAS_DATABASE_URL is required to apply database migrations/)
  assert.match(entrypoint, /case "\$\{1:-\}" in/)
  assert.match(entrypoint, /air\|\/tachigo\|tachigo\)/)
  const migrateIndex = entrypoint.indexOf('atlas migrate apply')
  const execIndex = entrypoint.indexOf('exec "$@"')
  const commandGateIndex = entrypoint.indexOf('case "${1:-}" in')
  assert.ok(commandGateIndex >= 0, 'entrypoint must gate migration by startup command')
  assert.ok(commandGateIndex < migrateIndex, 'entrypoint must decide whether to migrate before applying migrations')
  assert.ok(migrateIndex >= 0, 'entrypoint must apply Atlas migrations')
  assert.ok(execIndex > migrateIndex, 'entrypoint must start the API only after migrations apply')

  assert.match(makefile, /^migrate:\n\tatlas migrate apply --dir file:\/\/migrations --url "\$\(ATLAS_DATABASE_URL\)"/m)
  assert.match(compose, /ATLAS_DATABASE_URL: postgres:\/\/postgres:postgres@postgres:5432\/tachigo\?sslmode=disable/)
  assert.match(composeOverride, /target: dev/)
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
  assert.match(workflow, /bodyValidForCi &&\n\s+!isDocsTemplateOrMetadataOnly &&/)
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
    run_contracts_slither: 'false',
    run_contracts_gas_report: 'false',
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
    run_contracts_slither: 'false',
    run_contracts_gas_report: 'false',
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
    run_contracts_slither: 'true',
    run_contracts_gas_report: 'false',
  })
})

test('scope gate emits contracts report outputs for contracts PRs', async () => {
  const result = await runCiScopeGateWorkflow({
    prOverrides: { title: '[contract] Update TachiToken' },
    files: [{ filename: 'contracts/src/TachiToken.sol', additions: 4, deletions: 1, status: 'modified' }],
  })

  assert.deepEqual(result.outputs, {
    run_ci: 'true',
    run_backend: 'false',
    run_backend_integration: 'true',
    run_backend_scanners: 'false',
    run_dependency_review: 'false',
    run_frontend: 'false',
    run_dashboard: 'false',
    run_contracts: 'true',
    run_contracts_slither: 'true',
    run_contracts_gas_report: 'true',
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
    run_contracts_slither: 'true',
    run_contracts_gas_report: 'true',
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
  const parsedWorkflow = parseYaml(workflowPath)
  const backendBuild = workflowJobBlock(workflow, 'backend-build')
  const backend = workflowJobBlock(workflow, 'backend')
  const atlas = workflowJobBlock(workflow, 'atlas-migration-tooling')
  const backendIntegration = workflowJobBlock(workflow, 'backend-integration')
  const backendIntegrationJob = parsedWorkflow.jobs['backend-integration']
  const backendSecurityScanners = workflowJobBlock(workflow, 'backend-security-scanners')
  const dependencyReview = workflowJobBlock(workflow, 'dependency-review')
  const frontend = workflowJobBlock(workflow, 'frontend')
  const dashboard = workflowJobBlock(workflow, 'dashboard')
  const contracts = workflowJobBlock(workflow, 'contracts')
  const contractsSlither = workflowJobBlock(workflow, 'contracts-slither')
  const contractsGasSnapshot = workflowJobBlock(workflow, 'contracts-gas-snapshot')

  assert.match(backendBuild, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(backend, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(atlas, /needs\.scope-gate\.outputs\.run_backend == 'true'/)
  assert.match(backendIntegration, /needs\.scope-gate\.outputs\.run_backend_integration == 'true'/)
  assert.match(backendSecurityScanners, /needs\.scope-gate\.outputs\.run_backend_scanners == 'true'/)
  assert.match(dependencyReview, /needs\.scope-gate\.outputs\.run_dependency_review == 'true'/)
  assert.match(frontend, /needs\.scope-gate\.outputs\.run_frontend == 'true'/)
  assert.match(dashboard, /needs\.scope-gate\.outputs\.run_dashboard == 'true'/)
  assert.match(contracts, /needs\.scope-gate\.outputs\.run_contracts == 'true'/)
  assert.match(contractsSlither, /needs\.scope-gate\.outputs\.run_contracts_slither == 'true'/)
  assert.match(contractsGasSnapshot, /needs\.scope-gate\.outputs\.run_contracts_gas_report == 'true'/)
  assert.deepEqual(backendIntegrationJob.needs, ['scope-gate', 'check-cache-wiring'])
})

test('PR commit message check skips formal release promotion PRs', () => {
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['pr-commit-messages']

  assert.match(job.if, /github\.event\.pull_request\.user\.login != 'dependabot\[bot\]'/)
  assert.match(job.if, /github\.event\.pull_request\.head\.repo\.full_name != github\.repository/)
  assert.match(job.if, /github\.event\.pull_request\.base\.ref != 'main'/)
  assert.match(job.if, /github\.event\.pull_request\.head\.ref != 'develop'/)
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
  assert.match(jobBlock, /go-version: 1\.26\.3/)
  assert.match(jobBlock, /go install honnef\.co\/go\/tools\/cmd\/staticcheck@\$STATICCHECK_VERSION/)
  assert.match(jobBlock, /go install golang\.org\/x\/vuln\/cmd\/govulncheck@\$GOVULNCHECK_VERSION/)
  assert.match(jobBlock, /working-directory: services\/api\n\s+run: staticcheck \.\/\.\.\./)
  assert.match(jobBlock, /working-directory: services\/api\n\s+run: govulncheck \.\/\.\.\./)
  assert.deepEqual(backendCi.needs, [
    'backend-build',
    'backend',
    'atlas-migration-tooling',
    'backend-integration',
    'backend-security-scanners',
  ])
  assert.match(backendCiBlock, /ATLAS_MIGRATION_TOOLING_RESULT: \$\{\{ needs\.atlas-migration-tooling\.result \}\}/)
  assert.match(backendCiBlock, /"atlas-migration-tooling:\$ATLAS_MIGRATION_TOOLING_RESULT"/)
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
  assert.match(jobBlock, pinnedActionRef('actions/checkout', 'v4'))
  assert.match(jobBlock, pinnedActionRef('actions/dependency-review-action', 'v4'))
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
  assert.match(policy, /Dependabot opens routine version update PRs for the configured update levels/)
  assert.match(policy, /dependabot-pnpm-lockfile\.yml/)
  assert.match(policy, /shared-workspace-lockfile=false/)
  assert.match(policy, /security update PRs for alert-triggered fixes/)
  assert.match(policy, /Production security update\s+PRs remain manual-review changes/)
  assert.match(policy, /False Positives And Waivers/)
  assert.match(policy, /Owner:/)
  assert.match(policy, /Expires on:/)
})

test('Dependabot pnpm version updates skip routine production patch releases', () => {
  const config = parseYaml(dependabotConfigPath)
  const pnpmUpdates = config.updates.filter((update) => update['package-ecosystem'] === 'npm')

  assert.equal(pnpmUpdates.length, 2)
  for (const update of pnpmUpdates) {
    assert.deepEqual(update.allow, [
      {
        'dependency-type': 'production',
        'update-types': [
          'version-update:semver-minor',
          'version-update:semver-major',
        ],
      },
      {
        'dependency-type': 'development',
        'update-types': [
          'version-update:semver-patch',
          'version-update:semver-minor',
          'version-update:semver-major',
        ],
      },
    ])
  }
})

test('Dependabot pnpm lockfile repair is scoped to same-repo Dependabot PRs', async () => {
  const workflow = await readFile(dependabotPnpmLockfileWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(dependabotPnpmLockfileWorkflowPath)
  const job = parsedWorkflow.jobs['repair-lockfiles']
  const jobBlock = workflowJobBlock(workflow, 'repair-lockfiles')

  assert.equal(job.name, 'Repair pnpm lockfiles')
  assert.equal(job.permissions.contents, 'write')
  assert.equal(job.permissions.actions, 'write')
  assert.equal(job.permissions['pull-requests'], 'read')
  assert.match(job.if, /github\.event\.pull_request\.user\.login == 'dependabot\[bot\]'/)
  assert.match(job.if, /github\.event\.pull_request\.head\.repo\.full_name == github\.repository/)
  assert.match(jobBlock, /ref: \$\{\{ github\.event\.pull_request\.head\.ref \}\}/)
  assert.match(jobBlock, /repository: \$\{\{ github\.event\.pull_request\.head\.repo\.full_name \}\}/)
  assert.match(jobBlock, /corepack prepare pnpm@10\.33\.0 --activate/)
  assert.match(
    jobBlock,
    /working-directory: apps\/dashboard\n\s+run: pnpm install --lockfile-only --ignore-scripts --config\.shared-workspace-lockfile=false/,
  )
  assert.match(
    jobBlock,
    /working-directory: apps\/extension\n\s+run: pnpm install --lockfile-only --ignore-scripts --config\.shared-workspace-lockfile=false/,
  )
  assert.match(jobBlock, /git add pnpm-lock\.yaml apps\/dashboard\/pnpm-lock\.yaml apps\/extension\/pnpm-lock\.yaml/)
  assert.match(jobBlock, /git commit -m "chore\(deps\): repair pnpm lockfiles"/)
  assert.match(jobBlock, /git push/)
})

test('Dependabot auto-merge keeps production dependency updates on manual review', async () => {
  const workflow = await readFile(dependabotAutomergeWorkflowPath, 'utf8')
  const jobBlock = workflowJobBlock(workflow, 'automerge')

  assert.doesNotMatch(jobBlock, /BASE_REF: \$\{\{ github\.event\.pull_request\.base\.ref \}\}/)
  assert.doesNotMatch(jobBlock, /DEFAULT_BRANCH: \$\{\{ github\.event\.repository\.default_branch \}\}/)
  assert.doesNotMatch(jobBlock, /default-branch production patch update \(security-alert path\)/)
  assert.match(jobBlock, /\[\[ "\$DEPENDENCY_TYPE" != "direct:development" \]\]/)
  assert.match(jobBlock, /reason="production dependency requires manual review"/)
})

test('CI scope gate runs product validation for Dependabot maintenance PRs', async () => {
  const workflow = await readFile(workflowPath, 'utf8')

  assert.match(workflow, /const isDependabotPr = pr\.user\?\.login === 'dependabot\[bot\]'/)
  assert.match(workflow, /const bodyValidForCi = isDependabotPr \|\| standardBodyValid/)
  assert.match(
    workflow,
    /const runCi =\n\s+\(isReleasePromotionPr && releaseBodyValid\) \|\|\n\s+bodyValidForCi &&/,
  )
})

test('Dependabot auto-merge uses repository-supported merge commits', async () => {
  const workflow = await readFile(dependabotAutomergeWorkflowPath, 'utf8')

  assert.match(workflow, /gh pr merge --auto --merge "\$PR_URL"/)
  assert.doesNotMatch(workflow, /gh pr merge --auto --squash/)
})

test('contracts Slither report job uploads SARIF and keeps findings report-only', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['contracts-slither']
  const jobBlock = workflowJobBlock(workflow, 'contracts-slither')

  assert.equal(job.name, 'Contracts Slither report')
  assert.equal(job['timeout-minutes'], 20)
  assert.deepEqual(job.needs, ['scope-gate'])
  assert.equal(job.if, "needs.scope-gate.outputs.run_contracts_slither == 'true'")
  assert.equal(job.permissions.contents, 'read')
  assert.equal(job.permissions['security-events'], 'write')
  assert.match(jobBlock, pinnedActionRef('actions/checkout', 'v4'))
  assert.match(jobBlock, pinnedActionRef('foundry-rs/foundry-toolchain', 'v1'))
  assert.match(jobBlock, /working-directory: contracts\n\s+run: forge install OpenZeppelin\/openzeppelin-contracts@v5\.6\.1 --no-git/)
  assert.match(jobBlock, pinnedActionRef('crytic/slither-action', 'v0.4.2'))
  assert.match(jobBlock, /id: slither/)
  assert.match(jobBlock, /target: contracts/)
  assert.match(jobBlock, /slither-version: 0\.11\.5/)
  assert.match(jobBlock, /sarif: slither\.sarif/)
  assert.match(jobBlock, /fail-on: none/)
  assert.match(jobBlock, pinnedActionRef('github/codeql-action/upload-sarif', 'v3'))
  assert.match(jobBlock, /sarif_file: \$\{\{ steps\.slither\.outputs\.sarif \}\}/)
  assert.match(jobBlock, pinnedActionRef('actions/upload-artifact', 'v4'))
  assert.match(jobBlock, /name: slither-report/)
  assert.match(jobBlock, /path: \$\{\{ steps\.slither\.outputs\.sarif \}\}/)
  assert.doesNotMatch(jobBlock, /continue-on-error: true/)
})

test('contracts Slither policy documents baseline triage and waiver handling', async () => {
  const policy = await readFile(securityScannerEvaluationPath, 'utf8')

  assert.match(policy, /Contracts Slither Report/)
  assert.match(policy, /fail-on: none/)
  assert.match(policy, /slither-report/)
  assert.match(policy, /Owner:/)
  assert.match(policy, /Accepted on:/)
  assert.match(policy, /Expires on:/)
  assert.match(policy, /GitHub code scanning/)
})

test('contracts gas snapshot job publishes a report-only artifact', async () => {
  const workflow = await readFile(workflowPath, 'utf8')
  const parsedWorkflow = parseYaml(workflowPath)
  const job = parsedWorkflow.jobs['contracts-gas-snapshot']
  const jobBlock = workflowJobBlock(workflow, 'contracts-gas-snapshot')

  assert.equal(job.name, 'Contracts gas snapshot report')
  assert.equal(job['timeout-minutes'], 20)
  assert.deepEqual(job.needs, ['scope-gate'])
  assert.equal(job.if, "needs.scope-gate.outputs.run_contracts_gas_report == 'true'")
  assert.match(jobBlock, pinnedActionRef('actions/checkout', 'v4'))
  assert.match(jobBlock, pinnedActionRef('foundry-rs/foundry-toolchain', 'v1'))
  assert.match(jobBlock, /working-directory: contracts\n\s+run: forge install OpenZeppelin\/openzeppelin-contracts@v5\.6\.1 --no-git/)
  assert.match(jobBlock, /working-directory: contracts\n\s+run: forge snapshot --snap gas-snapshot\.report/)
  assert.match(jobBlock, /cat gas-snapshot\.report/)
  assert.match(jobBlock, pinnedActionRef('actions/upload-artifact', 'v4'))
  assert.match(jobBlock, /name: contracts-gas-snapshot-report/)
  assert.match(jobBlock, /path: contracts\/gas-snapshot\.report/)
  assert.doesNotMatch(jobBlock, /--check/)
  assert.doesNotMatch(jobBlock, /continue-on-error: true/)
})

test('contracts gas snapshot policy documents baseline and reviewer handling', async () => {
  const policy = await readFile(contractsGasSnapshotPolicyPath, 'utf8')

  assert.match(policy, /Gas Snapshot Policy/)
  assert.match(policy, /contracts-gas-snapshot-report/)
  assert.match(policy, /`.gas-snapshot` is not committed/)
  assert.match(policy, /Tolerance/)
  assert.match(policy, /Intentional gas changes checklist/)
  assert.match(policy, /Reviewer accepted the gas impact/)
  assert.match(policy, /forge snapshot --check/)
})

test('dependency inventory workflow publishes report-only OSV scans by surface', async () => {
  const workflow = await readFile(dependencyInventoryWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(dependencyInventoryWorkflowPath)

  assert.equal(parsedWorkflow.name, 'Dependency inventory scan')
  assert.deepEqual(parsedWorkflow.on.schedule, [{ cron: '41 3 * * 2' }])
  assert.ok(Object.hasOwn(parsedWorkflow.on, 'workflow_dispatch'))
  assert.equal(parsedWorkflow.permissions.contents, 'read')
  assert.equal(parsedWorkflow.permissions.actions, 'read')
  assert.equal(parsedWorkflow.permissions['security-events'], 'write')
  assert.equal(parsedWorkflow.concurrency.group, 'dependency-inventory-${{ github.repository }}')
  assert.equal(parsedWorkflow.concurrency['cancel-in-progress'], false)

  const go = parsedWorkflow.jobs['osv-go-modules']
  const pnpm = parsedWorkflow.jobs['osv-pnpm-lockfiles']
  const containerArchives = parsedWorkflow.jobs['build-container-inventory-archives']
  const backendImage = parsedWorkflow.jobs['osv-container-backend-image']
  const extensionImage = parsedWorkflow.jobs['osv-container-extension-image']
  const dashboardImage = parsedWorkflow.jobs['osv-container-dashboard-image']
  const notifyDiscord = parsedWorkflow.jobs['notify-discord']

  for (const job of [go, pnpm, backendImage, extensionImage, dashboardImage]) {
    assert.equal(job.uses, 'google/osv-scanner-action/.github/workflows/osv-scanner-reusable.yml@v2.3.5')
    assert.equal(job.with['fail-on-vuln'], false)
    assert.equal(job.with['upload-sarif'], true)
  }

  assert.match(go.with['scan-args'], /--lockfile=services\/api\/go\.mod/)
  assert.equal(go.with['results-file-name'], 'osv-go-modules.sarif')
  assert.equal(go.with['matrix-property'], 'go-')

  assert.match(pnpm.with['scan-args'], /--lockfile=pnpm-lock\.yaml/)
  assert.match(pnpm.with['scan-args'], /--lockfile=apps\/extension\/pnpm-lock\.yaml/)
  assert.match(pnpm.with['scan-args'], /--lockfile=apps\/dashboard\/pnpm-lock\.yaml/)
  assert.equal(pnpm.with['results-file-name'], 'osv-pnpm-lockfiles.sarif')
  assert.equal(pnpm.with['matrix-property'], 'pnpm-')

  assert.equal(containerArchives.name, 'Build dependency inventory image archives')
  assert.equal(containerArchives['timeout-minutes'], 30)
  assert.match(workflowJobBlock(workflow, 'build-container-inventory-archives'), /docker build -t tachigo-backend-inventory:latest \.\/services\/api/)
  assert.match(workflowJobBlock(workflow, 'build-container-inventory-archives'), /docker save tachigo-backend-inventory:latest -o tachigo-backend-image\.tar/)
  assert.match(workflowJobBlock(workflow, 'build-container-inventory-archives'), /name: dependency-inventory-backend-image/)
  assert.match(workflowJobBlock(workflow, 'build-container-inventory-archives'), /name: dependency-inventory-extension-image/)
  assert.match(workflowJobBlock(workflow, 'build-container-inventory-archives'), /name: dependency-inventory-dashboard-image/)

  assert.equal(backendImage.needs, 'build-container-inventory-archives')
  assert.equal(backendImage.with['download-artifact'], 'dependency-inventory-backend-image')
  assert.equal(backendImage.with['results-file-name'], 'osv-container-backend-image.sarif')
  assert.equal(backendImage.with['matrix-property'], 'container-backend-')
  assert.match(backendImage.with['scan-args'], /scan\nimage\n--archive\ntachigo-backend-image\.tar/)

  assert.equal(extensionImage.with['download-artifact'], 'dependency-inventory-extension-image')
  assert.equal(extensionImage.with['results-file-name'], 'osv-container-extension-image.sarif')
  assert.equal(extensionImage.with['matrix-property'], 'container-extension-')
  assert.match(extensionImage.with['scan-args'], /scan\nimage\n--archive\ntachigo-extension-image\.tar/)

  assert.equal(dashboardImage.with['download-artifact'], 'dependency-inventory-dashboard-image')
  assert.equal(dashboardImage.with['results-file-name'], 'osv-container-dashboard-image.sarif')
  assert.equal(dashboardImage.with['matrix-property'], 'container-dashboard-')
  assert.match(dashboardImage.with['scan-args'], /scan\nimage\n--archive\ntachigo-dashboard-image\.tar/)

  assert.equal(notifyDiscord.name, 'Notify Discord on dependency inventory failure')
  assert.deepEqual(notifyDiscord.needs, [
    'osv-go-modules',
    'osv-pnpm-lockfiles',
    'build-container-inventory-archives',
    'osv-container-backend-image',
    'osv-container-extension-image',
    'osv-container-dashboard-image',
  ])
  assert.equal(notifyDiscord.if, 'failure()')
  assert.match(workflowJobBlock(workflow, 'notify-discord'), /secrets\.DISCORD_CI_WEBHOOK_URL/)
  assert.match(workflowJobBlock(workflow, 'notify-discord'), /Dependency inventory scan failed/)
})

test('dependency inventory policy documents OSV triage ownership and non-blocking rollout', async () => {
  const policy = await readFile(dependencyInventoryPolicyPath, 'utf8')

  assert.match(policy, /Scheduled Dependency Inventory/)
  assert.match(policy, /Source of truth.*#508/)
  assert.match(policy, /Report owner/)
  assert.match(policy, /Triage SLA/)
  assert.match(policy, /Go module manifest/)
  assert.match(policy, /pnpm lockfiles/)
  assert.match(policy, /Container images/)
  assert.match(policy, /Dependabot alerts/)
  assert.match(policy, /Dependency Review/)
  assert.match(policy, /not a required check/)
  assert.match(policy, /weekly/)
  assert.match(policy, /workflow_dispatch/)
  assert.match(policy, /False positives and waivers/)
  assert.match(policy, /Surface: Go module manifest \/ pnpm lockfiles \/ Container images/)
  assert.match(policy, /Owner:/)
  assert.match(policy, /Expires on:/)
})

test('release PR workflow gates automated creation by age and merged PR volume', async () => {
  const workflow = await readFile(releasePrWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(releasePrWorkflowPath)
  const job = parsedWorkflow.jobs['create-release-pr']
  const jobBlock = workflowJobBlock(workflow, 'create-release-pr')

  assert.equal(parsedWorkflow.name, 'Release PR')
  assert.deepEqual(parsedWorkflow.on.schedule, [{ cron: '0 2 * * *' }])
  assert.ok(Object.hasOwn(parsedWorkflow.on, 'workflow_dispatch'))
  assert.equal(parsedWorkflow.permissions.contents, 'read')
  assert.equal(parsedWorkflow.permissions['pull-requests'], 'write')
  assert.equal(parsedWorkflow.permissions.actions, 'write')

  assert.equal(job['timeout-minutes'], 10)
  assert.match(jobBlock, pinnedActionRef('actions/checkout', 'v4'))
  assert.match(jobBlock, /const minElapsedHours = 72/)
  assert.match(jobBlock, /const minMergedPrs = 10/)
  assert.match(jobBlock, /const maxElapsedHours = 168/)
  assert.match(jobBlock, /process\.env\.GITHUB_EVENT_NAME === 'workflow_dispatch'/)
  assert.match(jobBlock, /const shouldOpen = isManual \|\| \(enoughAge && \(enoughPrs \|\| staleEnough\)\)/)
  assert.match(jobBlock, /gh pr list --base main --head develop --state open/)
  assert.match(jobBlock, /gh pr list --base main --head develop --state merged/)
  assert.match(jobBlock, /gh pr list --base develop --state merged/)
  assert.match(jobBlock, /gh pr create/)
  assert.doesNotMatch(jobBlock, /--draft/)
  assert.doesNotMatch(jobBlock, /--label auto-ready/)
})

test('release PR workflow includes grouped PR summaries in generated body', async () => {
  const workflow = await readFile(releasePrWorkflowPath, 'utf8')
  const jobBlock = workflowJobBlock(workflow, 'create-release-pr')

  assert.match(jobBlock, /gh pr list --base develop --state merged --limit 300 --json number,title,mergedAt,url,author,mergeCommit/)
  assert.match(jobBlock, /gh pr list --base develop --state open --limit 300 --json number,title,url,labels/)
  assert.match(jobBlock, /const titleGroups = \[/)
  assert.match(jobBlock, /\/tmp\/release-pr-summary\.md/)
  assert.match(jobBlock, /## Included PRs/)
  assert.match(jobBlock, /title: 'Backend'/)
  assert.match(jobBlock, /title: 'Frontend'/)
  assert.match(jobBlock, /title: 'Contracts'/)
  assert.match(jobBlock, /title: 'Infrastructure'/)
  assert.match(jobBlock, /title: 'Maintenance'/)
  assert.match(jobBlock, /title: 'Discussion'/)
  assert.match(jobBlock, /\[backend\]/i)
  assert.match(jobBlock, /\[frontend\]/i)
  assert.match(jobBlock, /\[contract\]/i)
  assert.match(jobBlock, /\[infra\]/i)
  assert.match(jobBlock, /\[chore\]/i)
  assert.match(jobBlock, /\[discussion\]/i)
  assert.match(jobBlock, /Unprefixed \/ Other/)
  assert.match(jobBlock, /## Release Warnings/)
  assert.match(jobBlock, /changes-requested/)
  assert.match(jobBlock, /scope-violation/)
  assert.match(jobBlock, /const escapeMarkdown = /)
  assert.match(jobBlock, /const safeTitle = escapeMarkdown\(cleanTitle\(pr\.title\)\)/)
  assert.match(jobBlock, /by \$\{author\}/)
  assert.match(jobBlock, /merge commit: \\\`\$\{mergeCommit\}\\\`/)
  assert.match(jobBlock, /\$\(cat \/tmp\/release-pr-summary\.md\)/)
})

test('release cadence documentation matches the automated gate', async () => {
  const claude = await readFile(claudePath, 'utf8')
  const policy = await readFile(prScopePolicyPath, 'utf8')

  for (const doc of [claude, policy]) {
    assert.match(doc, /72 小時/)
    assert.match(doc, /10 個 PR/)
    assert.match(doc, /7 天/)
  }

  assert.doesNotMatch(claude, /每兩週由 `develop` 開一張正式 release PR 到 `main`/)
  assert.doesNotMatch(policy, /預設 cadence 為每兩週一次 `develop -> main` release PR/)
})

test('global auto-merge workflow excludes Dependabot and workflow-file PRs', async () => {
  const workflow = await readFile(autoMergeWorkflowPath, 'utf8')
  const parsedWorkflow = parseYaml(autoMergeWorkflowPath)

  assert.deepEqual(parsedWorkflow.on.pull_request.types, [
    'opened',
    'reopened',
    'ready_for_review',
  ])
  assert.equal(
    parsedWorkflow.jobs['enable-auto-merge'].if,
    "github.event.pull_request.draft == false && github.event.pull_request.user.login != 'dependabot[bot]' && github.event.pull_request.base.ref == 'develop'",
  )
  assert.match(workflow, /Skipping auto-merge for workflow-file PR/)
  assert.ok(
    workflow.includes("--jq '.[] | .filename, (.previous_filename // empty)'"),
    'workflow-file PR detection must include renamed-away workflow files',
  )
  assert.ok(workflow.includes("grep -Eq '^\\.github/workflows/'"))
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

test('notify rebase workflow uses read-only pull request permission', async () => {
  const parsedWorkflow = parseYaml(notifyRebaseNeededWorkflowPath)

  assert.equal(parsedWorkflow.permissions.issues, 'write')
  assert.equal(parsedWorkflow.permissions['pull-requests'], 'read')
})

test('notify rebase workflow skips duplicate notification for the same PR head', async () => {
  const result = await runNotifyRebaseNeededWorkflow({
    openPRs: [
      { number: 601 },
    ],
    freshPRsByNumber: {
      601: {
        number: 601,
        mergeable_state: 'dirty',
        head: { sha: 'dirty-head-sha' },
      },
    },
    commentsByIssueNumber: {
      601: [
        {
          body: '<!-- notify-rebase-needed:head=dirty-head-sha -->\n> existing notification',
        },
      ],
    },
  })

  assert.equal(result.commentsListed.length, 1)
  assert.deepEqual(result.commentsCreated, [])
  assert.match(result.infos.join('\n'), /already has a rebase notification/)
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
  assert.deepEqual(job.needs, ['scope-gate', 'backend-ci', 'dependency-review', 'frontend', 'dashboard', 'contracts', 'contracts-slither', 'contracts-gas-snapshot'])
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
  assert.match(jobBlock, /CONTRACTS_SLITHER_RESULT/)
  assert.match(jobBlock, /CONTRACTS_GAS_SNAPSHOT_RESULT/)
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

test('CI auto-ready job waits when contracts Slither report fails', async () => {
  const result = await runCiAutoReadyAfterCiWorkflow({
    env: { CONTRACTS_SLITHER_RESULT: 'failure' },
    checkRuns: successfulDevelopRequiredCheckRuns(),
  })

  assert.deepEqual(result.graphqlCalls, [])
  assert.deepEqual(result.labelsAdded, [])
})

test('CI auto-ready job waits when contracts gas snapshot report fails', async () => {
  const result = await runCiAutoReadyAfterCiWorkflow({
    env: { CONTRACTS_GAS_SNAPSHOT_RESULT: 'failure' },
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
