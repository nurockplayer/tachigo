import React from 'react'
import {usePluginData} from '@docusaurus/useGlobalData'

import {formatLinkWeight} from './relatedLinksData'

type ReverseIndexEntry = {
  pr: number
  title: string
  mergedAt: string
  additions: number
  deletions: number
  weight: number
  reasons: string[]
  paths: string[]
}

type ReverseIndexData = {
  generatedAt?: string
  stale?: boolean
  warning?: string
  entries?: Record<string, ReverseIndexEntry[]>
}

type ChangelogItem = ReverseIndexEntry & {
  docPath: string
}

const GITHUB_REPO_URL = 'https://github.com/nurockplayer/tachigo'
const DAY_MS = 24 * 60 * 60 * 1000

function toTime(value: string | undefined): number {
  const timestamp = Date.parse(value ?? '')
  return Number.isFinite(timestamp) ? timestamp : 0
}

export function createChangelogItems(
  data: ReverseIndexData | undefined,
  windowDays = 90,
): ChangelogItem[] {
  const entries = data?.entries ?? {}
  const generatedAt = toTime(data?.generatedAt) || Date.now()
  const cutoff = generatedAt - windowDays * DAY_MS

  return Object.entries(entries)
    .flatMap(([docPath, docEntries]) =>
      docEntries.map((entry) => ({
        ...entry,
        docPath,
      })),
    )
    .filter((entry) => toTime(entry.mergedAt) >= cutoff)
    .sort((a, b) => toTime(b.mergedAt) - toTime(a.mergedAt) || b.pr - a.pr)
}

function formatDate(value: string): string {
  const timestamp = toTime(value)

  if (!timestamp) {
    return 'unknown date'
  }

  return new Date(timestamp).toISOString().slice(0, 10)
}

function formatDocPath(docPath: string): string {
  return docPath.replace(/^docs\//, '').replace(/\.md$/, '')
}

function formatReasons(reasons: string[]): string {
  if (reasons.length === 0) {
    return 'reason unavailable'
  }

  return reasons
    .map((reason) =>
      reason
        .replace(/^body:/, 'body: ')
        .replace(/^changedFiles:/, 'changedFiles: ')
        .replace('title:doc-slug', 'title: doc slug'),
    )
    .join(', ')
}

export default function ReverseIndexChangelog(): JSX.Element {
  const data = usePluginData('tachigo-reverse-index') as ReverseIndexData | undefined
  const items = createChangelogItems(data)

  if (items.length === 0) {
    return (
      <div className="tachigo-reverse-index-empty">
        committed cache 目前沒有可顯示的 reverse-index entries。
      </div>
    )
  }

  return (
    <section className="tachigo-reverse-index" aria-label="Reverse index changelog">
      {data?.stale ? (
        <p className="tachigo-reverse-index__warning">
          Cache is stale{data.warning ? `: ${data.warning}` : '。'}
        </p>
      ) : null}
      <div className="tachigo-reverse-index__list">
        {items.map((item) => (
          <article
            className="tachigo-reverse-index__item"
            key={`${item.docPath}-${item.pr}`}
          >
            <div className="tachigo-reverse-index__meta">
              <a href={`${GITHUB_REPO_URL}/pull/${item.pr}`} target="_blank" rel="noreferrer">
                #{item.pr}
              </a>
              <span>{formatDate(item.mergedAt)}</span>
              <span>weight {formatLinkWeight(item.weight)}</span>
            </div>
            <h3>{item.title}</h3>
            <p>
              <strong>{formatDocPath(item.docPath)}</strong>
              <span> — {formatReasons(item.reasons)}</span>
            </p>
          </article>
        ))}
      </div>
    </section>
  )
}
