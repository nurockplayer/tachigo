#!/usr/bin/env node

import { readdir, readFile, stat } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const ignoredDirs = new Set([
  '.git',
  '.pnpm-store',
  '.worktrees',
  'build',
  'coverage',
  'dist',
  'node_modules',
])

const packageFiles = new Set(['package.json'])
const lockfileNames = new Set([
  'bun.lock',
  'bun.lockb',
  'npm-shrinkwrap.json',
  'package-lock.json',
  'pnpm-lock.yaml',
  'yarn.lock',
])

const disallowedLifecycleScripts = new Set([
  'preinstall',
  'install',
  'postinstall',
  'prepare',
])

const disallowedDynamicExecPatterns = [
  { label: 'npx', pattern: /(^|[\s;&|()])npx($|[\s;&|()])/ },
  { label: 'pnpm dlx', pattern: /\bpnpm\s+dlx\b/ },
  { label: 'npm exec', pattern: /\bnpm\s+exec\b/ },
  { label: 'curl piped to shell', pattern: /\bcurl\b[\s\S]*\|\s*(?:bash|sh)\b/ },
  { label: 'wget piped to shell', pattern: /\bwget\b[\s\S]*\|\s*(?:bash|sh)\b/ },
]

const miniShaiHuludIndicators = [
  /router_init\.js/i,
  /router_runtime\.js/i,
  /tanstack_runner\.js/i,
  /git-tanstack/i,
  /getsession\.org/i,
  /83\.142\.209\.194/i,
  /gh-token-monitor/i,
  /IfYouRevokeThisTokenItWillWipeTheComputerOfTheOwner/i,
  /Shai-Hulud: Here We Go Again/i,
  /79ac49eedf774dd4b0cfa308722bc463cfe5885c/i,
  /transformers\.pyz/i,
  /pgmonitor\.py/i,
  /pgsql-monitor\.service/i,
]

const tanstackRouterStartAdvisoryPattern =
  /@tanstack\/(?:arktype-adapter|eslint-plugin-router|eslint-plugin-start|history|nitro-v2-vite-plugin|react-router|react-router-devtools|react-router-ssr-query|react-start|react-start-client|react-start-rsc|react-start-server|router-cli|router-core|router-devtools|router-devtools-core|router-generator|router-plugin|router-ssr-query-core|router-utils|router-vite-plugin|solid-router|solid-router-devtools|solid-router-ssr-query|solid-start|solid-start-client|solid-start-server|start-client-core|start-fn-stubs|start-plugin-core|start-server-core|start-static-server-functions|start-storage-context|valibot-adapter|virtual-file-routes|vue-router|vue-router-devtools|vue-router-ssr-query|vue-start|vue-start-client|vue-start-server|zod-adapter)(?:@|['":/\s])/i

const otherAdvisoryPackagePatterns = [
  /@mistralai\/(?:mistralai|mistralai-azure|mistralai-gcp)(?:@|['":/\s])/i,
  /@opensearch-project\/opensearch(?:@|['":/\s])/i,
  /@uipath\//i,
  /@squawk\//i,
  /@tallyui\//i,
  /@beproduct\/nestjs-auth(?:@|['":/\s])/i,
  /@cap-js\/(?:db-service|postgres|sqlite)(?:@|['":/\s])/i,
  /@dirigible-ai\/sdk(?:@|['":/\s])/i,
  /@draftauth\/(?:client|core)(?:@|['":/\s])/i,
  /@draftlab\/(?:auth|auth-router|db)(?:@|['":/\s])/i,
  /@mesadev\/(?:rest|saguaro|sdk)(?:@|['":/\s])/i,
  /@ml-toolkit-ts\/(?:preprocessing|xgboost)(?:@|['":/\s])/i,
  /@supersurkhet\/(?:cli|sdk)(?:@|['":/\s])/i,
  /@taskflow-corp\/cli(?:@|['":/\s])/i,
  /@tolka\/cli(?:@|['":/\s])/i,
  /\b(?:agentwork-cli|cmux-agent-mcp|cross-stitch|git-branch-selector|git-git-git|guardrails-ai|intercom-client|mistralai|ml-toolkit-ts|nextmove-mcp|safe-action|ts-dna|wot-api)(?:@|['":/\s])/i,
  /\blightning@2\.6\.[23]\b/i,
]

function parseArgs(argv) {
  const options = { root: process.cwd() }
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--root') {
      const value = argv[index + 1]
      if (!value) {
        throw new Error('--root requires a path')
      }
      options.root = value
      index += 1
    } else {
      throw new Error(`unknown argument: ${arg}`)
    }
  }
  return options
}

async function walk(root) {
  const files = []
  async function visit(current) {
    const entries = await readdir(current, { withFileTypes: true })
    for (const entry of entries) {
      if (entry.isDirectory()) {
        if (!ignoredDirs.has(entry.name)) {
          await visit(path.join(current, entry.name))
        }
        continue
      }
      if (entry.isFile()) {
        files.push(path.join(current, entry.name))
      }
    }
  }
  await visit(root)
  return files
}

function relative(root, filePath) {
  return path.relative(root, filePath) || path.basename(filePath)
}

async function checkPackageJson(root, filePath) {
  const problems = []
  let parsed
  try {
    parsed = JSON.parse(await readFile(filePath, 'utf8'))
  } catch (error) {
    problems.push(`${relative(root, filePath)}: invalid package.json: ${error.message}`)
    return problems
  }

  const scripts = parsed.scripts || {}
  for (const [scriptName, command] of Object.entries(scripts)) {
    if (disallowedLifecycleScripts.has(scriptName)) {
      problems.push(`${relative(root, filePath)}: disallowed lifecycle script "${scriptName}"`)
    }
    if (typeof command !== 'string') {
      continue
    }
    for (const { label, pattern } of disallowedDynamicExecPatterns) {
      if (pattern.test(command)) {
        problems.push(`${relative(root, filePath)}: disallowed dynamic package execution in "${scriptName}" (${label})`)
      }
    }
  }

  return problems
}

async function checkAdvisoryContent(root, filePath) {
  const problems = []
  const content = await readFile(filePath, 'utf8')
  for (const pattern of miniShaiHuludIndicators) {
    if (pattern.test(content)) {
      problems.push(`${relative(root, filePath)}: Mini Shai-Hulud indicator matched ${pattern}`)
    }
  }
  if (tanstackRouterStartAdvisoryPattern.test(content)) {
    problems.push(`${relative(root, filePath)}: TanStack Router/Start advisory package requires review`)
  }
  for (const pattern of otherAdvisoryPackagePatterns) {
    if (pattern.test(content)) {
      problems.push(`${relative(root, filePath)}: advisory package pattern requires review: ${pattern}`)
    }
  }
  return problems
}

async function main() {
  const { root } = parseArgs(process.argv.slice(2))
  const resolvedRoot = path.resolve(root)
  const rootStat = await stat(resolvedRoot)
  if (!rootStat.isDirectory()) {
    throw new Error(`root is not a directory: ${resolvedRoot}`)
  }

  const files = await walk(resolvedRoot)
  const problems = []
  let tanstackReactQueryObserved = false

  for (const filePath of files) {
    const name = path.basename(filePath)
    if (packageFiles.has(name)) {
      problems.push(...await checkPackageJson(resolvedRoot, filePath))
      problems.push(...await checkAdvisoryContent(resolvedRoot, filePath))
    } else if (lockfileNames.has(name)) {
      const content = await readFile(filePath, 'utf8')
      tanstackReactQueryObserved ||= /@tanstack\/react-query@5\.100\.6/.test(content)
      problems.push(...await checkAdvisoryContent(resolvedRoot, filePath))
    }
  }

  if (problems.length > 0) {
    console.error('Supply-chain guardrails failed:')
    for (const problem of problems) {
      console.error(`- ${problem}`)
    }
    process.exitCode = 1
    return
  }

  console.log('Supply-chain guardrails passed')
  if (tanstackReactQueryObserved) {
    console.log('TanStack check: @tanstack/react-query@5.100.6 observed; documented as not listed in current Mini Shai-Hulud Router/Start advisories on 2026-05-13.')
  }
}

main().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
