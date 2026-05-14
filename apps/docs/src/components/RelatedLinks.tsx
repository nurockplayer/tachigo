import React from 'react'
import {useDoc} from '@docusaurus/plugin-content-docs/client'
import {usePluginData} from '@docusaurus/useGlobalData'

type DocRelated = {
  related_issues?: number[]
  implemented_in?: number[]
}

type RelatedLinksGlobalData = {
  docs?: Record<string, DocRelated>
}

type DocMetadata = {
  id?: string
  source?: string
  frontMatter?: DocRelated
}

const GITHUB_REPO_URL = 'https://github.com/nurockplayer/tachigo'

function uniquePositiveNumbers(values: number[] | undefined): number[] {
  if (!values) {
    return []
  }

  return [...new Set(values.filter((value) => Number.isInteger(value) && value > 0))]
}

function normalizeSourceKey(source: string): string {
  return source
    .replace(/^@site\//, '')
    .replace(/^(\.\.\/)+docs\//, '')
    .replace(/^docs\//, '')
}

function candidateDocKeys(metadata: DocMetadata): string[] {
  const candidates = new Set<string>()

  if (metadata.source) {
    candidates.add(normalizeSourceKey(metadata.source))
  }

  if (metadata.id) {
    candidates.add(`${metadata.id}.md`)
    candidates.add(`${metadata.id}/index.md`)
  }

  return [...candidates]
}

function findPluginRelated(
  data: RelatedLinksGlobalData | undefined,
  metadata: DocMetadata,
): DocRelated | undefined {
  const docs = data?.docs

  if (!docs) {
    return undefined
  }

  for (const key of candidateDocKeys(metadata)) {
    if (docs[key]) {
      return docs[key]
    }
  }

  return undefined
}

export function resolveDocRelated(
  data: RelatedLinksGlobalData | undefined,
  metadata: DocMetadata,
): DocRelated {
  return findPluginRelated(data, metadata) ?? metadata.frontMatter ?? {}
}

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

export default function RelatedLinks(): JSX.Element | null {
  const {metadata} = useDoc()
  const pluginData = usePluginData('tachigo-related-links') as
    | RelatedLinksGlobalData
    | undefined
  const related = resolveDocRelated(pluginData, metadata as DocMetadata)
  const relatedIssues = uniquePositiveNumbers(related.related_issues)
  const implementedIn = uniquePositiveNumbers(related.implemented_in)

  if (relatedIssues.length === 0 && implementedIn.length === 0) {
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
    </section>
  )
}
