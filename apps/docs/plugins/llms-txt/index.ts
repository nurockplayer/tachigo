import fs from 'node:fs/promises'
import path from 'node:path'
import {fileURLToPath} from 'node:url'

import type {LoadContext, Plugin} from '@docusaurus/types'
import fg from 'fast-glob'
import matter from 'gray-matter'

type ManifestStatus = 'active' | 'proposed' | 'deprecated'

interface LlmFrontmatter {
  title?: string
  description?: string
  status?: ManifestStatus
  owner?: string
  source_of_truth?: boolean
  code_areas?: string[]
  related_issues?: number[]
  implemented_in?: number[]
  excluded_from_llms?: boolean
  internal?: boolean
}

export interface LlmsDoc {
  relativePath: string
  source: string
  content: string
  frontmatter: LlmFrontmatter
  title: string
  description: string
  url: string
  section: string
}

export interface LlmsManifest {
  version: '1.0'
  generated_at: string
  base_url: string
  docs: Array<{
    path: string
    title: string
    status?: ManifestStatus
    owner?: string
    code_areas?: string[]
    related_issues?: number[]
    implemented_in?: number[]
  }>
}

export interface LlmsArtifacts {
  llmsTxt: string
  llmsFullTxt: string
  manifest: LlmsManifest
}

export interface LlmsTxtOptions {
  docsRoot?: string | URL
  include?: string[]
  baseUrl?: string
}

interface CollectOptions extends LlmsTxtOptions {
  siteDir?: string
}

interface CollectResult {
  docsRoot: string
  docs: LlmsDoc[]
  warnings: string[]
}

const DEFAULT_BASE_URL = '/tachigo/'
const SOURCE_INDEX_RELATIVE_PATH = 'dev-portal/source-index.md'

function resolveDocsRoot(siteDir: string, docsRoot?: string | URL): string {
  if (docsRoot instanceof URL) {
    return path.resolve(fileURLToPath(docsRoot))
  }

  if (typeof docsRoot === 'string') {
    return path.isAbsolute(docsRoot) ? docsRoot : path.resolve(siteDir, docsRoot)
  }

  return path.resolve(siteDir, '../../docs')
}

function normalizeBaseUrl(baseUrl = DEFAULT_BASE_URL): string {
  const withLeadingSlash = baseUrl.startsWith('/') ? baseUrl : `/${baseUrl}`
  return withLeadingSlash.endsWith('/') ? withLeadingSlash : `${withLeadingSlash}/`
}

function relativePathToUrl(relativePath: string, baseUrl: string): string {
  const slug = relativePath.replace(/\.mdx?$/, '').replace(/(^|\/)index$/, '$1').replace(/\/$/, '')

  if (!slug) {
    return baseUrl
  }

  return `${baseUrl}${slug}`
}

function slugToRelativePath(slug: string): string {
  const normalized = slug.replace(/^\/+/, '').replace(/\/+$/, '')

  if (!normalized) {
    return 'index.md'
  }

  return normalized.endsWith('.md') ? normalized : `${normalized}.md`
}

function titleFromFilename(relativePath: string): string {
  const basename = path.basename(relativePath, path.extname(relativePath))
  return basename
    .split('-')
    .filter(Boolean)
    .map((part) => `${part[0]?.toUpperCase() ?? ''}${part.slice(1)}`)
    .join(' ')
}

function extractTitle(content: string, frontmatter: LlmFrontmatter, relativePath: string): string {
  if (frontmatter.title) {
    return frontmatter.title
  }

  const heading = content.match(/^#\s+(.+)$/m)
  return heading?.[1]?.trim() || titleFromFilename(relativePath)
}

function stripMarkdownInline(text: string): string {
  return text
    .replace(/^>\s*/, '')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[`*_~]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function extractDescription(content: string, frontmatter: LlmFrontmatter): string {
  if (frontmatter.description) {
    return frontmatter.description
  }

  const lines = content.split(/\r?\n/)
  let inFence = false

  for (const line of lines) {
    const trimmed = line.trim()

    if (trimmed.startsWith('```')) {
      inFence = !inFence
      continue
    }

    if (
      inFence ||
      trimmed.length === 0 ||
      trimmed.startsWith('#') ||
      trimmed.startsWith('|') ||
      trimmed.startsWith('---') ||
      trimmed.startsWith('<') ||
      trimmed.startsWith('![')
    ) {
      continue
    }

    return stripMarkdownInline(trimmed)
  }

  return 'tachigo Dev Portal document'
}

function sectionForDoc(relativePath: string): string {
  if (relativePath.startsWith('dev-portal/')) {
    return 'Dev Portal'
  }

  if (relativePath.startsWith('ai/')) {
    return 'AI Workflow'
  }

  if (relativePath.startsWith('history/')) {
    return 'History'
  }

  if (relativePath.startsWith('superpowers/')) {
    return 'Design Specs'
  }

  return 'Architecture'
}

function isExcluded(relativePath: string, frontmatter: LlmFrontmatter): boolean {
  if (relativePath.startsWith('superpowers/plans/')) {
    return true
  }

  if (
    relativePath.startsWith('superpowers/specs/') &&
    (frontmatter.status === 'proposed' || frontmatter.status === 'deprecated')
  ) {
    return true
  }

  return (
    frontmatter.status === 'deprecated' ||
    frontmatter.internal === true ||
    frontmatter.excluded_from_llms === true
  )
}

function isTierOne(relativePath: string, frontmatter: LlmFrontmatter): boolean {
  return (
    frontmatter.status === 'active' ||
    frontmatter.source_of_truth === true ||
    relativePath.startsWith('dev-portal/')
  )
}

export function parseRootSourceOfTruthLinks(source: string): Set<string> {
  const lines = source.split(/\r?\n/)
  const headingIndex = lines.findIndex((line) => /^##+\s+Root source of truth\s*$/i.test(line.trim()))

  if (headingIndex === -1) {
    throw new Error('Root source of truth table was not found')
  }

  const tableLines: string[] = []

  for (const line of lines.slice(headingIndex + 1)) {
    const trimmed = line.trim()

    if (trimmed.length === 0) {
      if (tableLines.length === 0) {
        continue
      }

      break
    }

    if (!trimmed.startsWith('|')) {
      if (tableLines.length === 0) {
        continue
      }

      break
    }

    tableLines.push(trimmed)
  }

  const rootPaths = new Set<string>()

  for (const line of tableLines) {
    if (/^\|\s*-+/.test(line)) {
      continue
    }

    const matches = line.matchAll(/\[[^\]]+\]\(([^)#?]+)[^)]*\)/g)

    for (const match of matches) {
      const href = match[1]
      const slug = href.startsWith('/tachigo/') ? href.slice('/tachigo/'.length) : href

      if (slug.startsWith('http://') || slug.startsWith('https://')) {
        continue
      }

      rootPaths.add(slugToRelativePath(slug))
    }
  }

  if (rootPaths.size === 0) {
    throw new Error('Root source of truth table did not contain markdown links')
  }

  return rootPaths
}

async function loadRootSourceOfTruthPaths(docsRoot: string): Promise<{
  rootPaths: Set<string>
  warnings: string[]
}> {
  try {
    const source = await fs.readFile(path.join(docsRoot, SOURCE_INDEX_RELATIVE_PATH), 'utf8')
    return {rootPaths: parseRootSourceOfTruthLinks(source), warnings: []}
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    return {
      rootPaths: new Set(),
      warnings: [`llms-txt: Root source of truth fallback disabled: ${message}`],
    }
  }
}

function toManifestDoc(doc: LlmsDoc): LlmsManifest['docs'][number] {
  return {
    path: doc.url,
    title: doc.title,
    ...(doc.frontmatter.status ? {status: doc.frontmatter.status} : {}),
    ...(doc.frontmatter.owner ? {owner: doc.frontmatter.owner} : {}),
    ...(doc.frontmatter.code_areas ? {code_areas: doc.frontmatter.code_areas} : {}),
    ...(doc.frontmatter.related_issues ? {related_issues: doc.frontmatter.related_issues} : {}),
    ...(doc.frontmatter.implemented_in ? {implemented_in: doc.frontmatter.implemented_in} : {}),
  }
}

export async function collectLlmsDocs(options: CollectOptions = {}): Promise<CollectResult> {
  const siteDir = options.siteDir ?? process.cwd()
  const docsRoot = resolveDocsRoot(siteDir, options.docsRoot)
  const baseUrl = normalizeBaseUrl(options.baseUrl)
  const include = options.include ?? ['**/*.md']
  const [relativePaths, rootSourceResult] = await Promise.all([
    fg(include, {cwd: docsRoot, onlyFiles: true}),
    loadRootSourceOfTruthPaths(docsRoot),
  ])
  const docs: LlmsDoc[] = []

  for (const relativePath of relativePaths.sort()) {
    const source = await fs.readFile(path.join(docsRoot, relativePath), 'utf8')
    const parsed = matter(source)
    const frontmatter = parsed.data as LlmFrontmatter

    if (isExcluded(relativePath, frontmatter)) {
      continue
    }

    if (!isTierOne(relativePath, frontmatter) && !rootSourceResult.rootPaths.has(relativePath)) {
      continue
    }

    docs.push({
      relativePath,
      source,
      content: parsed.content.trim(),
      frontmatter,
      title: extractTitle(parsed.content, frontmatter, relativePath),
      description: extractDescription(parsed.content, frontmatter),
      url: relativePathToUrl(relativePath, baseUrl),
      section: sectionForDoc(relativePath),
    })
  }

  return {
    docsRoot,
    docs,
    warnings: rootSourceResult.warnings,
  }
}

export function buildLlmsArtifacts(options: {
  docs: LlmsDoc[]
  baseUrl?: string
  generatedAt?: string
}): LlmsArtifacts {
  const baseUrl = normalizeBaseUrl(options.baseUrl)
  const generatedAt = options.generatedAt ?? new Date().toISOString()
  const llmsLines = [
    '# tachigo Dev Portal',
    '',
    '> tachigo / tachiya 專案導覽入口，供 AI agent 快速取得 source-of-truth 文件索引。',
    '',
  ]
  const currentSections = new Set<string>()

  for (const doc of options.docs) {
    if (!currentSections.has(doc.section)) {
      if (currentSections.size > 0) {
        llmsLines.push('')
      }

      llmsLines.push(`## ${doc.section}`)
      currentSections.add(doc.section)
    }

    llmsLines.push(`- [${doc.title}](${doc.url}): ${doc.description}`)
  }

  const llmsFullTxt = options.docs
    .map((doc) =>
      [
        '---',
        `Title: ${doc.title}`,
        `Source: ${doc.url}`,
        `Path: docs/${doc.relativePath}`,
        '---',
        '',
        doc.content,
      ].join('\n'),
    )
    .join('\n\n')

  return {
    llmsTxt: `${llmsLines.join('\n').trimEnd()}\n`,
    llmsFullTxt: `${llmsFullTxt.trimEnd()}\n`,
    manifest: {
      version: '1.0',
      generated_at: generatedAt,
      base_url: baseUrl,
      docs: options.docs.map(toManifestDoc),
    },
  }
}

export default function llmsTxtPlugin(
  context: LoadContext,
  options: LlmsTxtOptions = {},
): Plugin<void> {
  const baseUrl = normalizeBaseUrl(options.baseUrl ?? context.baseUrl ?? DEFAULT_BASE_URL)

  return {
    name: 'tachigo-llms-txt',

    async postBuild({outDir}) {
      const result = await collectLlmsDocs({
        siteDir: context.siteDir,
        docsRoot: options.docsRoot,
        include: options.include,
        baseUrl,
      })

      for (const warning of result.warnings) {
        console.warn(warning)
      }

      const artifacts = buildLlmsArtifacts({
        docs: result.docs,
        baseUrl,
      })

      await Promise.all([
        fs.writeFile(path.join(outDir, 'llms.txt'), artifacts.llmsTxt, 'utf8'),
        fs.writeFile(path.join(outDir, 'llms-full.txt'), artifacts.llmsFullTxt, 'utf8'),
        fs.writeFile(
          path.join(outDir, 'manifest.json'),
          `${JSON.stringify(artifacts.manifest, null, 2)}\n`,
          'utf8',
        ),
      ])
    },
  }
}
