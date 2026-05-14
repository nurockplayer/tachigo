import fs from 'node:fs/promises'
import path from 'node:path'
import type {LoadContext, Plugin} from '@docusaurus/types'

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

export interface ReverseIndexDocMatch extends ReverseIndexEntry {
  path: string
}

export interface ReverseIndexByPrEntry {
  pr: number
  title: string
  mergedAt: string
  docs: ReverseIndexDocMatch[]
}

export interface ReverseIndexGlobalData {
  generatedAt: string
  stale: boolean
  warning?: string
  source: {
    repository: string
    limit: number
  }
  entries: Record<string, ReverseIndexEntry[]>
  byPr: Record<number, ReverseIndexByPrEntry>
}

export interface ReverseIndexPluginOptions {
  cachePath?: string
}

function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function toStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value.filter((item): item is string => typeof item === 'string')
}

function normalizeEntry(value: unknown): ReverseIndexEntry | undefined {
  if (!isObject(value)) {
    return undefined
  }

  const pr = Number(value.pr)
  const weight = Number(value.weight)

  if (!Number.isInteger(pr) || pr <= 0 || !Number.isFinite(weight)) {
    return undefined
  }

  return {
    pr,
    title: typeof value.title === 'string' ? value.title : `PR #${pr}`,
    mergedAt: typeof value.mergedAt === 'string' ? value.mergedAt : '',
    additions: Number.isFinite(Number(value.additions)) ? Number(value.additions) : 0,
    deletions: Number.isFinite(Number(value.deletions)) ? Number(value.deletions) : 0,
    weight,
    reasons: toStringArray(value.reasons),
    paths: toStringArray(value.paths),
  }
}

export function normalizeReverseIndexCache(raw: unknown): ReverseIndexGlobalData {
  const cache = isObject(raw) ? raw : {}
  const rawEntries = isObject(cache.entries) ? cache.entries : {}
  const entries: ReverseIndexGlobalData['entries'] = {}

  for (const [docPath, rawDocEntries] of Object.entries(rawEntries)) {
    if (!Array.isArray(rawDocEntries)) {
      continue
    }

    const normalized = rawDocEntries.flatMap((entry) => {
      const item = normalizeEntry(entry)
      return item ? [item] : []
    })

    if (normalized.length > 0) {
      entries[docPath] = normalized
    }
  }

  return {
    generatedAt:
      typeof cache.generatedAt === 'string' ? cache.generatedAt : new Date(0).toISOString(),
    stale: Boolean(cache.stale),
    warning: typeof cache.warning === 'string' ? cache.warning : undefined,
    source: {
      repository:
        isObject(cache.source) && typeof cache.source.repository === 'string'
          ? cache.source.repository
          : 'nurockplayer/tachigo',
      limit:
        isObject(cache.source) && Number.isFinite(Number(cache.source.limit))
          ? Number(cache.source.limit)
          : 0,
    },
    entries,
    byPr: buildByPr(entries),
  }
}

function buildByPr(
  entries: Record<string, ReverseIndexEntry[]>,
): Record<number, ReverseIndexByPrEntry> {
  const byPr: Record<number, ReverseIndexByPrEntry> = {}

  for (const [docPath, docEntries] of Object.entries(entries)) {
    for (const entry of docEntries) {
      byPr[entry.pr] ??= {
        pr: entry.pr,
        title: entry.title,
        mergedAt: entry.mergedAt,
        docs: [],
      }

      byPr[entry.pr].docs.push({
        ...entry,
        path: docPath,
      })
    }
  }

  return byPr
}

export async function loadReverseIndexCache(cachePath: string): Promise<ReverseIndexGlobalData> {
  try {
    const source = await fs.readFile(cachePath, 'utf8')
    return normalizeReverseIndexCache(JSON.parse(source))
  } catch (error) {
    const warning =
      error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT'
        ? `reverse-index cache not found at ${cachePath}`
        : `reverse-index cache could not be loaded: ${
            error instanceof Error ? error.message : String(error)
          }`

    console.warn(`[reverse-index] ${warning}`)

    return {
      generatedAt: new Date(0).toISOString(),
      stale: true,
      warning,
      source: {
        repository: 'nurockplayer/tachigo',
        limit: 0,
      },
      entries: {},
      byPr: {},
    }
  }
}

export default function reverseIndexPlugin(
  context: LoadContext,
  options: ReverseIndexPluginOptions = {},
): Plugin<ReverseIndexGlobalData> {
  const cachePath = options.cachePath ?? path.join(context.siteDir, '.cache/pr-to-doc.json')

  return {
    name: 'tachigo-reverse-index',

    async loadContent() {
      return loadReverseIndexCache(cachePath)
    },

    async contentLoaded({content, actions}) {
      actions.setGlobalData(content)
    },
  }
}
