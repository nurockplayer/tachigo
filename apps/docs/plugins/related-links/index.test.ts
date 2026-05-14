import test from 'node:test'
import assert from 'node:assert/strict'
import {fileURLToPath} from 'node:url'

import * as relatedLinksData from '../../src/components/relatedLinksData.ts'
import {
  collectRelatedLinks,
  default as relatedLinksPlugin,
} from './index.ts'
import {validateFrontmatterDocs} from '../frontmatter-validator/index.ts'
import {resolveRelatedLinksData} from '../../src/components/relatedLinksData.ts'

const currentDir = fileURLToPath(new URL('.', import.meta.url))
const fixtureDocsRoot = new URL('./__fixtures__/docs/', import.meta.url)

test('collects only docs with related issues or implemented PRs', async () => {
  const validation = await validateFrontmatterDocs({
    docsRoot: fixtureDocsRoot,
    allowWarnings: true,
  })

  const relatedLinks = collectRelatedLinks(validation)

  assert.deepEqual(relatedLinks.docs, {
    'with-links.md': {
      related_issues: [674, 695],
      implemented_in: [679],
    },
    'pr-only.md': {
      implemented_in: [680],
    },
  })
})

test('plugin lifecycle publishes related links as global data', async () => {
  const plugin = relatedLinksPlugin(
    {siteDir: currentDir},
    {docsRoot: fixtureDocsRoot},
  )
  const content = await plugin.loadContent()
  let globalData

  await plugin.contentLoaded({
    content,
    actions: {
      setGlobalData(data) {
        globalData = data
      },
    },
  })

  assert.equal(globalData, content)
  assert.deepEqual(content.docs['with-links.md'].related_issues, [674, 695])
  assert.equal(content.docs['warning-only.md'], undefined)
})

test('plugin allows frontmatter warnings by default', async () => {
  const plugin = relatedLinksPlugin(
    {siteDir: currentDir},
    {docsRoot: fixtureDocsRoot},
  )

  const content = await plugin.loadContent()

  assert.deepEqual(content.docs['with-links.md'].related_issues, [674, 695])
})

test('resolved related links keep authoritative and inferred PRs separate', () => {
  const resolved = resolveRelatedLinksData({
    metadata: {
      id: 'watch-to-points-design',
      source: '@site/docs/watch-to-points-design.md',
      frontMatter: {
        related_issues: [696],
        implemented_in: [709],
      },
    },
    reverseIndex: {
      byPr: {
        709: {
          pr: 709,
          title: 'Authoritative PR should not duplicate inferred section',
          mergedAt: '2026-05-14T08:00:00Z',
          docs: [
            {
              path: 'docs/watch-to-points-design.md',
              title: 'Authoritative PR should not duplicate inferred section',
              mergedAt: '2026-05-14T08:00:00Z',
              additions: 12,
              deletions: 2,
              weight: 1,
              reasons: ['body:docs/watch-to-points-design.md'],
              paths: ['docs/watch-to-points-design.md'],
            },
          ],
        },
        710: {
          pr: 710,
          title: 'Infer from watchtime scope',
          mergedAt: '2026-05-15T08:00:00Z',
          docs: [
            {
              path: 'docs/watch-to-points-design.md',
              title: 'Infer from watchtime scope',
              mergedAt: '2026-05-15T08:00:00Z',
              additions: 18,
              deletions: 3,
              weight: 0.5,
              reasons: ['changedFiles:services/api/internal/watchtime'],
              paths: ['services/api/internal/watchtime/session.go'],
            },
          ],
        },
        711: {
          pr: 711,
          title: 'Below-threshold match should stay hidden',
          mergedAt: '2026-05-15T09:00:00Z',
          docs: [
            {
              path: 'docs/watch-to-points-design.md',
              title: 'Below-threshold match should stay hidden',
              mergedAt: '2026-05-15T09:00:00Z',
              additions: 8,
              deletions: 1,
              weight: 0.49,
              reasons: ['changedFiles:services/api/internal/low-signal'],
              paths: ['services/api/internal/low-signal/example.go'],
            },
          ],
        },
      },
    },
  })

  assert.deepEqual(resolved.authoritative, {
    relatedIssues: [696],
    implementedIn: [709],
  })
  assert.deepEqual(resolved.inferredPullRequests, [
    {
      pr: 710,
      title: 'Infer from watchtime scope',
      mergedAt: '2026-05-15T08:00:00Z',
      weight: 0.5,
      reasons: ['changedFiles:services/api/internal/watchtime'],
    },
  ])
})

test('formats missing or non-number link weight without throwing', () => {
  assert.equal(typeof relatedLinksData.formatLinkWeight, 'function')
  assert.equal(relatedLinksData.formatLinkWeight?.(undefined), '0.00')
  assert.equal(relatedLinksData.formatLinkWeight?.('not-a-number'), '0.00')
  assert.equal(relatedLinksData.formatLinkWeight?.(0.625), '0.63')
})
