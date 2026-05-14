import React from 'react'
import {useDoc} from '@docusaurus/plugin-content-docs/client'
import {usePluginData} from '@docusaurus/useGlobalData'

import {
  formatLinkWeight,
  resolveRelatedLinksData,
  type DocMetadata,
  type RelatedLinksGlobalData,
  type RelatedLinksResolvedData,
  type ReverseIndexGlobalData,
} from './relatedLinksData'

const GITHUB_REPO_URL = 'https://github.com/nurockplayer/tachigo'

function RelatedChip({href, number}: {href: string; number: number}): JSX.Element {
  return (
    <a
      className="tachigo-related-links__chip"
      href={href}
      target="_blank"
      rel="noreferrer"
    >
      #{number}
    </a>
  )
}

function InferredPrChip({
  entry,
}: {
  entry: RelatedLinksResolvedData['inferredPullRequests'][number]
}): JSX.Element {
  const formattedWeight = formatLinkWeight(entry.weight)
  const title = `${entry.title} · weight ${formattedWeight} · ${entry.reasons.join(', ')}`

  return (
    <a
      className="tachigo-related-links__chip tachigo-related-links__chip--inferred"
      href={`${GITHUB_REPO_URL}/pull/${entry.pr}`}
      target="_blank"
      rel="noreferrer"
      title={title}
    >
      #{entry.pr}
      <span>{formattedWeight}</span>
    </a>
  )
}

function RelatedGroup({
  title,
  hrefBase,
  numbers,
}: {
  title: string
  hrefBase: string
  numbers: number[]
}): JSX.Element | null {
  if (numbers.length === 0) {
    return null
  }

  return (
    <div className="tachigo-related-links__group">
      <h3>{title}</h3>
      <div className="tachigo-related-links__chips">
        {numbers.map((number) => (
          <RelatedChip key={number} href={`${hrefBase}/${number}`} number={number} />
        ))}
      </div>
    </div>
  )
}

function InferredPrGroup({
  entries,
}: {
  entries: RelatedLinksResolvedData['inferredPullRequests']
}): JSX.Element | null {
  if (entries.length === 0) {
    return null
  }

  return (
    <div className="tachigo-related-links__group">
      <h3>推論關聯 PR</h3>
      <div className="tachigo-related-links__chips">
        {entries.map((entry) => (
          <InferredPrChip key={entry.pr} entry={entry} />
        ))}
      </div>
    </div>
  )
}

export default function RelatedLinks(): JSX.Element | null {
  const {metadata} = useDoc()
  const pluginData = usePluginData('tachigo-related-links') as
    | RelatedLinksGlobalData
    | undefined
  const reverseIndexData = usePluginData('tachigo-reverse-index') as
    | ReverseIndexGlobalData
    | undefined
  const related = resolveRelatedLinksData({
    relatedLinks: pluginData,
    reverseIndex: reverseIndexData,
    metadata: metadata as DocMetadata,
  })
  const relatedIssues = related.authoritative.relatedIssues
  const implementedIn = related.authoritative.implementedIn
  const inferredPrs = related.inferredPullRequests

  if (relatedIssues.length === 0 && implementedIn.length === 0 && inferredPrs.length === 0) {
    return null
  }

  return (
    <section className="tachigo-related-links" aria-label="相關連結">
      <RelatedGroup
        title="相關 issue"
        hrefBase={`${GITHUB_REPO_URL}/issues`}
        numbers={relatedIssues}
      />
      <RelatedGroup
        title="實作 PR"
        hrefBase={`${GITHUB_REPO_URL}/pull`}
        numbers={implementedIn}
      />
      <InferredPrGroup entries={inferredPrs} />
    </section>
  )
}
