export type DocRelated = {
  related_issues?: number[]
  implemented_in?: number[]
}

export type ReverseIndexDocMatch = {
  path: string
  title: string
  mergedAt: string
  additions: number
  deletions: number
  weight: number
  reasons: string[]
  paths: string[]
}

export type ReverseIndexByPr = Record<
  number,
  {
    pr: number
    title: string
    mergedAt: string
    docs: ReverseIndexDocMatch[]
  }
>

export type RelatedLinksGlobalData = {
  docs?: Record<string, DocRelated>
}

export type ReverseIndexGlobalData = {
  byPr?: ReverseIndexByPr
}

export type DocMetadata = {
  id?: string
  source?: string
  frontMatter?: DocRelated
}

export type RelatedLinksResolvedData = {
  authoritative: {
    relatedIssues: number[]
    implementedIn: number[]
  }
  inferredPullRequests: Array<{
    pr: number
    title: string
    mergedAt: string
    weight: number
    reasons: string[]
  }>
}

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

export function resolveRelatedLinksData(options: {
  relatedLinks?: RelatedLinksGlobalData
  reverseIndex?: ReverseIndexGlobalData
  metadata: DocMetadata
}): RelatedLinksResolvedData {
  const related = resolveDocRelated(options.relatedLinks, options.metadata)
  const authoritative = {
    relatedIssues: uniquePositiveNumbers(related.related_issues),
    implementedIn: uniquePositiveNumbers(related.implemented_in),
  }

  const inferredByPr = new Map<number, RelatedLinksResolvedData['inferredPullRequests'][number]>()
  const docKeys = new Set(candidateDocKeys(options.metadata))

  for (const entry of Object.values(options.reverseIndex?.byPr ?? {})) {
    const matchingDoc = entry.docs.find((doc) => docKeys.has(normalizeSourceKey(doc.path)))

    if (!matchingDoc) {
      continue
    }

    if (authoritative.implementedIn.includes(entry.pr)) {
      continue
    }

    inferredByPr.set(entry.pr, {
      pr: entry.pr,
      title: entry.title,
      mergedAt: entry.mergedAt,
      weight: matchingDoc.weight,
      reasons: matchingDoc.reasons,
    })
  }

  return {
    authoritative,
    inferredPullRequests: [...inferredByPr.values()].sort(
      (a, b) => Date.parse(b.mergedAt) - Date.parse(a.mergedAt) || b.pr - a.pr,
    ),
  }
}
