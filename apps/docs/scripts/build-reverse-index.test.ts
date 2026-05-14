import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildReverseIndex,
  capCoreDocMatches,
  parseFrontmatter,
  scorePullRequestForDoc,
} from './build-reverse-index.ts'

const servicePr = {
  pr: 701,
  title: 'fix: watch heartbeat points ledger',
  body: 'Tightens watch session accounting.',
  mergedAt: '2026-05-14T00:00:00Z',
  additions: 12,
  deletions: 3,
  changedFiles: [
    'services/api/internal/watchtime/session.go',
    'services/api/internal/points/ledger.go',
  ],
}

test('broad docs default to explicit_only and are not inferred from changed service files', () => {
  const indexDoc = {
    path: 'docs/index.md',
    frontmatter: {
      code_areas: ['services/api', 'apps/extension', 'apps/dashboard'],
    },
  }
  const portalDoc = {
    path: 'docs/dev-portal/domain-maps.md',
    frontmatter: {
      code_areas: ['services/api', 'apps/extension', 'apps/dashboard'],
    },
  }

  assert.equal(scorePullRequestForDoc(servicePr, indexDoc), undefined)
  assert.equal(scorePullRequestForDoc(servicePr, portalDoc), undefined)
})

test('explicit doc path in PR body matches explicit_only docs', () => {
  const result = scorePullRequestForDoc(
    {
      ...servicePr,
      body: 'See docs/dev-portal/domain-maps.md for the affected area.',
    },
    {
      path: 'docs/dev-portal/domain-maps.md',
      frontmatter: {
        code_areas: ['services/api', 'apps/extension', 'apps/dashboard'],
      },
    },
  )

  assert.equal(result?.weight, 1)
  assert.deepEqual(result?.reasons, ['body:docs/dev-portal/domain-maps.md'])
})

test('reverse_index_scope narrows changed-file inference below broad code_areas', () => {
  const watchDoc = {
    path: 'docs/watch-to-points-design.md',
    frontmatter: {
      code_areas: ['services/api'],
      reverse_index_scope: ['services/api/internal/watchtime', 'services/api/internal/points'],
    },
  }

  const matching = scorePullRequestForDoc(servicePr, watchDoc)
  const nonMatching = scorePullRequestForDoc(
    {
      ...servicePr,
      changedFiles: ['services/api/internal/auth/service.go'],
    },
    watchDoc,
  )

  assert.equal(matching?.weight, 0.5)
  assert.deepEqual(matching?.reasons, ['changedFiles:services/api/internal/watchtime'])
  assert.equal(nonMatching, undefined)
})

test('buildReverseIndex only emits entries with weight at least 0.5', () => {
  const result = buildReverseIndex({
    docs: [
      {
        path: 'docs/watch-to-points-design.md',
        frontmatter: {
          code_areas: ['services/api'],
          reverse_index_scope: ['services/api/internal/watchtime'],
        },
      },
      {
        path: 'docs/dev-portal/start-here.md',
        frontmatter: {
          code_areas: ['services/api', 'apps/extension'],
        },
      },
    ],
    pullRequests: [servicePr],
  })

  assert.deepEqual(Object.keys(result.entries), ['docs/watch-to-points-design.md'])
  assert.equal(result.entries['docs/watch-to-points-design.md'][0].weight, 0.5)
})

test('architecture core helper caps inferred matches at five', () => {
  const matches = Array.from({length: 8}, (_, index) => ({
    pr: 700 + index,
    title: `PR ${index}`,
    mergedAt: `2026-05-${String(index + 1).padStart(2, '0')}T00:00:00Z`,
    additions: 1,
    deletions: 0,
    weight: 0.5,
    reasons: ['changedFiles:services/api/internal'],
    paths: ['services/api/internal/file.go'],
  }))

  assert.equal(capCoreDocMatches('docs/architecture.md', matches).length, 5)
})

test('frontmatter parser accepts inline arrays used by design docs', () => {
  const frontmatter = parseFrontmatter(`---
code_areas: [services/api]
reverse_index_scope: [services/api/internal/watchtime, services/api/internal/points]
---

# Inline arrays
`)

  assert.deepEqual(frontmatter.code_areas, ['services/api'])
  assert.deepEqual(frontmatter.reverse_index_scope, [
    'services/api/internal/watchtime',
    'services/api/internal/points',
  ])
})
