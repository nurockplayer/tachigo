import fs from 'node:fs/promises'
import path from 'node:path'
import {fileURLToPath} from 'node:url'

type ReverseIndexMode = 'inferred' | 'explicit_only' | 'disabled'

export interface DocFrontmatter {
  title?: string
  code_areas?: string[]
  reverse_index_mode?: ReverseIndexMode
  reverse_index_scope?: string[]
}

export interface DocCandidate {
  path: string
  frontmatter: DocFrontmatter
}

export interface PullRequestCandidate {
  pr: number
  title: string
  body: string
  mergedAt: string
  additions: number
  deletions: number
  changedFiles: string[]
}

export interface ReverseIndexEntry {
  pr: number
  title: string
  mergedAt: string
  additions: number
  deletions: number
  weight: number
  reasons: string[]
  paths: string[]
}

export interface ReverseIndexCache {
  generatedAt: string
  stale: boolean
  warning?: string
  source: {
    repository: string
    limit: number
  }
  entries: Record<string, ReverseIndexEntry[]>
}

const explicitOnlyDefaults = [/^docs\/index\.md$/, /^docs\/dev-portal\/[^/]+\.md$/]
const coreDocCaps = new Map([['docs/architecture.md', 5]])

function normalizePath(value: string): string {
  return value.replaceAll('\\', '/').replace(/^\.?\//, '')
}

function defaultReverseIndexMode(docPath: string): ReverseIndexMode {
  return explicitOnlyDefaults.some((pattern) => pattern.test(docPath)) ? 'explicit_only' : 'inferred'
}

function getReverseIndexMode(doc: DocCandidate): ReverseIndexMode {
  return doc.frontmatter.reverse_index_mode ?? defaultReverseIndexMode(doc.path)
}

function includesDocPath(body: string, docPath: string): boolean {
  const normalizedBody = body.toLowerCase()
  const normalizedPath = normalizePath(docPath).toLowerCase()

  return normalizedBody.includes(normalizedPath)
}

function docSlug(docPath: string): string {
  return path.basename(docPath, path.extname(docPath)).toLowerCase()
}

function titleMentionsDocSlug(title: string, docPath: string): boolean {
  const slug = docSlug(docPath)
  const normalizedTitle = title.toLowerCase()

  return normalizedTitle.includes(slug) || normalizedTitle.includes(slug.replaceAll('-', ' '))
}

function scopeDepth(scope: string): number {
  return normalizePath(scope)
    .split('/')
    .filter((part) => part.length > 0).length
}

function fileMatchesScope(filePath: string, scope: string): boolean {
  const file = normalizePath(filePath)
  const normalizedScope = normalizePath(scope)

  return file === normalizedScope || file.startsWith(`${normalizedScope}/`)
}

function firstChangedFileScopeReason(
  changedFiles: string[],
  scopes: string[],
): string | undefined {
  const sortedScopes = [...scopes].map(normalizePath).sort((a, b) => b.length - a.length)

  for (const scope of sortedScopes) {
    if (scopeDepth(scope) < 2) {
      continue
    }

    if (changedFiles.some((filePath) => fileMatchesScope(filePath, scope))) {
      return `changedFiles:${scope}`
    }
  }

  return undefined
}

export function scorePullRequestForDoc(
  pullRequest: PullRequestCandidate,
  doc: DocCandidate,
): {weight: number; reasons: string[]; paths: string[]} | undefined {
  const mode = getReverseIndexMode(doc)

  if (mode === 'disabled') {
    return undefined
  }

  const reasons: string[] = []
  const paths = pullRequest.changedFiles.map(normalizePath)

  if (includesDocPath(pullRequest.body ?? '', doc.path)) {
    reasons.push(`body:${normalizePath(doc.path)}`)
  }

  if (mode !== 'explicit_only' && titleMentionsDocSlug(pullRequest.title, doc.path)) {
    reasons.push('title:doc-slug')
  }

  const scopes = doc.frontmatter.reverse_index_scope ?? doc.frontmatter.code_areas ?? []
  const changedFileReason =
    mode !== 'explicit_only' ? firstChangedFileScopeReason(paths, scopes) : undefined

  if (changedFileReason) {
    reasons.push(changedFileReason)
  }

  const weight = reasons.reduce((total, reason) => {
    if (reason.startsWith('body:')) {
      return total + 1
    }

    if (reason === 'title:doc-slug') {
      return total + 0.6
    }

    if (reason.startsWith('changedFiles:')) {
      return total + 0.5
    }

    return total
  }, 0)

  if (weight < 0.5) {
    return undefined
  }

  return {
    weight: Number(Math.min(weight, 1).toFixed(2)),
    reasons,
    paths,
  }
}

export function capCoreDocMatches(
  docPath: string,
  matches: ReverseIndexEntry[],
): ReverseIndexEntry[] {
  const cap = coreDocCaps.get(normalizePath(docPath))

  if (!cap || matches.length <= cap) {
    return matches
  }

  return [...matches]
    .sort((a, b) => b.weight - a.weight || Date.parse(b.mergedAt) - Date.parse(a.mergedAt))
    .slice(0, cap)
}

export function buildReverseIndex(options: {
  docs: DocCandidate[]
  pullRequests: PullRequestCandidate[]
  generatedAt?: string
  stale?: boolean
  warning?: string
  repository?: string
  limit?: number
}): ReverseIndexCache {
  const entries: Record<string, ReverseIndexEntry[]> = {}

  for (const doc of options.docs) {
    const matches: ReverseIndexEntry[] = []

    for (const pullRequest of options.pullRequests) {
      const score = scorePullRequestForDoc(pullRequest, doc)

      if (!score) {
        continue
      }

      matches.push({
        pr: pullRequest.pr,
        title: pullRequest.title,
        mergedAt: pullRequest.mergedAt,
        additions: pullRequest.additions,
        deletions: pullRequest.deletions,
        weight: score.weight,
        reasons: score.reasons,
        paths: score.paths,
      })
    }

    const capped = capCoreDocMatches(doc.path, matches).sort(
      (a, b) => Date.parse(b.mergedAt) - Date.parse(a.mergedAt) || b.pr - a.pr,
    )

    if (capped.length > 0) {
      entries[normalizePath(doc.path)] = capped
    }
  }

  return {
    generatedAt: options.generatedAt ?? new Date().toISOString(),
    stale: options.stale ?? false,
    warning: options.warning,
    source: {
      repository: options.repository ?? 'nurockplayer/tachigo',
      limit: options.limit ?? options.pullRequests.length,
    },
    entries,
  }
}

async function readDocs(docsRoot: string): Promise<DocCandidate[]> {
  const relativePaths = (await listMarkdownFiles(docsRoot)).sort()

  return Promise.all(
    relativePaths.map(async (relativePath) => {
      const absolutePath = path.join(docsRoot, relativePath)
      const source = await fs.readFile(absolutePath, 'utf8')

      return {
        path: normalizePath(path.join('docs', relativePath)),
        frontmatter: parseFrontmatter(source),
      }
    }),
  )
}

async function listMarkdownFiles(root: string): Promise<string[]> {
  const files: string[] = []

  async function walk(directory: string) {
    const entries = await fs.readdir(directory, {withFileTypes: true})

    for (const entry of entries) {
      const absolutePath = path.join(directory, entry.name)

      if (entry.isDirectory()) {
        await walk(absolutePath)
      } else if (entry.isFile() && entry.name.endsWith('.md')) {
        files.push(normalizePath(path.relative(root, absolutePath)))
      }
    }
  }

  await walk(root)
  return files
}

export function parseFrontmatter(source: string): DocFrontmatter {
  if (!source.startsWith('---\n')) {
    return {}
  }

  const end = source.indexOf('\n---', 4)

  if (end === -1) {
    return {}
  }

  const frontmatter: DocFrontmatter = {}
  const lines = source.slice(4, end).split('\n')

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    const match = line.match(/^([A-Za-z0-9_]+):\s*(.*)$/)

    if (!match) {
      continue
    }

    const [, key, rawValue] = match

    if (key === 'title') {
      frontmatter.title = rawValue.trim()
    } else if (key === 'reverse_index_mode') {
      frontmatter.reverse_index_mode = rawValue.trim() as ReverseIndexMode
    } else if (key === 'code_areas' || key === 'reverse_index_scope') {
      const inlineValues = parseInlineArray(rawValue)

      if (inlineValues) {
        if (key === 'code_areas') {
          frontmatter.code_areas = inlineValues
        } else {
          frontmatter.reverse_index_scope = inlineValues
        }

        continue
      }

      const values: string[] = []
      let cursor = index + 1

      while (cursor < lines.length) {
        const item = lines[cursor].match(/^\s*-\s+(.+)$/)

        if (!item) {
          break
        }

        values.push(item[1].trim())
        cursor += 1
      }

      if (key === 'code_areas') {
        frontmatter.code_areas = values
      } else {
        frontmatter.reverse_index_scope = values
      }

      index = cursor - 1
    }
  }

  return frontmatter
}

function parseInlineArray(rawValue: string): string[] | undefined {
  const trimmed = rawValue.trim()

  if (!trimmed.startsWith('[') || !trimmed.endsWith(']')) {
    return undefined
  }

  return trimmed
    .slice(1, -1)
    .split(',')
    .map((item) => item.trim().replace(/^['"]|['"]$/g, ''))
    .filter(Boolean)
}

function parseArgs(argv: string[]): Record<string, string | boolean> {
  const parsed: Record<string, string | boolean> = {}

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]

    if (!arg.startsWith('--')) {
      continue
    }

    const [rawKey, inlineValue] = arg.slice(2).split('=', 2)
    const next = argv[index + 1]

    if (inlineValue !== undefined) {
      parsed[rawKey] = inlineValue
    } else if (next && !next.startsWith('--')) {
      parsed[rawKey] = next
      index += 1
    } else {
      parsed[rawKey] = true
    }
  }

  return parsed
}

async function fetchJson<T>(url: string, token: string): Promise<T> {
  const response = await fetch(url, {
    headers: {
      accept: 'application/vnd.github+json',
      authorization: `Bearer ${token}`,
      'x-github-api-version': '2022-11-28',
    },
  })

  if (!response.ok) {
    throw new Error(`GitHub API ${response.status} for ${url}`)
  }

  return response.json() as Promise<T>
}

async function fetchMergedPullRequests(options: {
  repository: string
  limit: number
  token: string
}): Promise<PullRequestCandidate[]> {
  const [owner, repo] = options.repository.split('/')
  const pullRequests: PullRequestCandidate[] = []
  let page = 1

  while (pullRequests.length < options.limit) {
    const listUrl = new URL(`https://api.github.com/repos/${owner}/${repo}/pulls`)
    listUrl.searchParams.set('state', 'closed')
    listUrl.searchParams.set('sort', 'updated')
    listUrl.searchParams.set('direction', 'desc')
    listUrl.searchParams.set('per_page', '100')
    listUrl.searchParams.set('page', String(page))

    const pageItems = await fetchJson<Array<{number: number; merged_at?: string | null}>>(
      listUrl.toString(),
      options.token,
    )

    if (pageItems.length === 0) {
      break
    }

    for (const item of pageItems) {
      if (!item.merged_at || pullRequests.length >= options.limit) {
        continue
      }

      const details = await fetchJson<{
        number: number
        title: string
        body?: string | null
        merged_at: string
        additions: number
        deletions: number
      }>(`https://api.github.com/repos/${owner}/${repo}/pulls/${item.number}`, options.token)
      const files = await fetchPullRequestFiles({
        owner,
        repo,
        pullRequestNumber: item.number,
        token: options.token,
      })

      pullRequests.push({
        pr: details.number,
        title: details.title,
        body: details.body ?? '',
        mergedAt: details.merged_at,
        additions: details.additions,
        deletions: details.deletions,
        changedFiles: files.map((file) => file.filename),
      })
    }

    page += 1
  }

  return pullRequests
}

async function fetchPullRequestFiles(options: {
  owner: string
  repo: string
  pullRequestNumber: number
  token: string
}): Promise<Array<{filename: string}>> {
  const files: Array<{filename: string}> = []
  let page = 1

  while (true) {
    const pageItems = await fetchJson<Array<{filename: string}>>(
      `https://api.github.com/repos/${options.owner}/${options.repo}/pulls/${options.pullRequestNumber}/files?per_page=100&page=${page}`,
      options.token,
    )

    files.push(...pageItems)

    if (pageItems.length < 100) {
      return files
    }

    page += 1
  }
}

async function readExistingCache(cachePath: string): Promise<ReverseIndexCache | undefined> {
  try {
    const source = await fs.readFile(cachePath, 'utf8')
    return JSON.parse(source) as ReverseIndexCache
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT') {
      return undefined
    }

    throw error
  }
}

async function writeCache(cachePath: string, cache: ReverseIndexCache): Promise<void> {
  await fs.mkdir(path.dirname(cachePath), {recursive: true})
  await fs.writeFile(cachePath, `${JSON.stringify(cache)}\n`, 'utf8')
}

function resolveRepository(input?: string | boolean): string {
  if (typeof input === 'string' && input.includes('/')) {
    return input
  }

  return process.env.GITHUB_REPOSITORY ?? 'nurockplayer/tachigo'
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const scriptDir = path.dirname(fileURLToPath(import.meta.url))
  const appDir = path.resolve(scriptDir, '..')
  const docsRoot = path.resolve(appDir, String(args['docs-root'] ?? '../../docs'))
  const cachePath = path.resolve(appDir, String(args.cache ?? '.cache/pr-to-doc.json'))
  const repository = resolveRepository(args.repo)
  const limit = Number(args.limit ?? process.env.REVERSE_INDEX_LIMIT ?? 200)
  const token = process.env.GH_TOKEN ?? process.env.GITHUB_TOKEN
  const docs = await readDocs(docsRoot)

  if (!token) {
    const existing = await readExistingCache(cachePath)
    const warning = 'GH_TOKEN/GITHUB_TOKEN is not set; using existing cache or writing stale empty cache.'
    const cache = existing
      ? {...existing, stale: true, warning}
      : buildReverseIndex({
          docs,
          pullRequests: [],
          stale: true,
          warning,
          repository,
          limit,
        })

    await writeCache(cachePath, cache)
    console.warn(`[reverse-index] ${warning}`)
    return
  }

  try {
    const pullRequests = await fetchMergedPullRequests({repository, limit, token})
    await writeCache(
      cachePath,
      buildReverseIndex({
        docs,
        pullRequests,
        repository,
        limit,
      }),
    )
  } catch (error) {
    const warning = `GitHub API failed; using existing cache or writing stale empty cache. ${
      error instanceof Error ? error.message : String(error)
    }`
    const existing = await readExistingCache(cachePath)
    const cache = existing
      ? {...existing, stale: true, warning}
      : buildReverseIndex({
          docs,
          pullRequests: [],
          stale: true,
          warning,
          repository,
          limit,
        })

    await writeCache(cachePath, cache)
    console.warn(`[reverse-index] ${warning}`)
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  await main()
}
