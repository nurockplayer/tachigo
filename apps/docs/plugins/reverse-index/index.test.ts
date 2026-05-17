import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'

import reverseIndexPlugin, {
  loadReverseIndexCache,
  normalizeReverseIndexCache,
} from './index.ts'

test('normalizes cache entries and drops malformed rows', () => {
  const cache = normalizeReverseIndexCache({
    generatedAt: '2026-05-14T00:00:00.000Z',
    stale: false,
    source: {
      repository: 'nurockplayer/tachigo',
      limit: 200,
    },
    entries: {
      'docs/architecture.md': [
        {
          pr: 648,
          title: '[chore] Align dashboard architecture state',
          mergedAt: '2026-05-13T07:50:30Z',
          additions: 5,
          deletions: 4,
          weight: 1,
          reasons: ['body:docs/architecture.md'],
          paths: ['docs/architecture.md'],
        },
        {pr: 'nope'},
      ],
    },
  })

  assert.equal(cache.entries['docs/architecture.md'].length, 1)
  assert.equal(cache.entries['docs/architecture.md'][0].pr, 648)
  assert.equal(cache.byPr[648].docs[0].path, 'docs/architecture.md')
  assert.equal(cache.source.limit, 200)
})

test('missing cache is published as stale empty data instead of failing build', async () => {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'tachigo-reverse-index-'))
  const cache = await loadReverseIndexCache(path.join(tempDir, 'missing.json'))

  assert.equal(cache.stale, true)
  assert.deepEqual(cache.entries, {})
  assert.deepEqual(cache.byPr, {})
  assert.match(cache.warning ?? '', /not found/)
})

test('plugin lifecycle publishes reverse index cache as global data', async () => {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'tachigo-reverse-index-'))
  const cachePath = path.join(tempDir, 'pr-to-doc.json')
  await fs.writeFile(
    cachePath,
    JSON.stringify({
      generatedAt: '2026-05-14T00:00:00.000Z',
      stale: false,
      source: {
        repository: 'nurockplayer/tachigo',
        limit: 1,
      },
      entries: {
        'docs/watch-to-points-design.md': [
          {
            pr: 374,
            title: '[backend] T-point docs cleanup',
            mergedAt: '2026-04-27T08:01:32Z',
            additions: 7,
            deletions: 5,
            weight: 1,
            reasons: ['body:docs/watch-to-points-design.md'],
            paths: ['docs/watch-to-points-design.md'],
          },
        ],
      },
    }),
  )

  const plugin = reverseIndexPlugin({siteDir: tempDir}, {cachePath})
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
  assert.equal(content.entries['docs/watch-to-points-design.md'][0].pr, 374)
  assert.equal(content.byPr[374].docs[0].path, 'docs/watch-to-points-design.md')
})
