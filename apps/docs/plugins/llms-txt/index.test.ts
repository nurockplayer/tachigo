import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'

import {
  buildLlmsArtifacts,
  collectLlmsDocs,
  default as llmsTxtPlugin,
  parseRootSourceOfTruthLinks,
} from './index.ts'

const doc = (body: string, fm = '') => `${fm ? `---\n${fm}---\n\n` : ''}${body.trimStart()}`
const sourceIndex = (table: string, heading = 'Root source of truth') =>
  doc(`# Source Index\n\n## ${heading}\n\n| 文件 | 定位 |\n|---|---|\n${table}`, 'title: Source Index\nstatus: active\n')

async function fixture(files: Record<string, string>): Promise<string> {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'tachigo-llms-docs-'))

  for (const [name, source] of Object.entries(files)) {
    await fs.mkdir(path.dirname(path.join(root, name)), {recursive: true})
    await fs.writeFile(path.join(root, name), source, 'utf8')
  }

  return root
}

test('filters docs by tier rules', async () => {
  const docsRoot = await fixture({
    'dev-portal/start-here.md': doc('# Start Here\n\nDev Portal entry point.', 'title: Start Here\nstatus: active\n'),
    'dev-portal/source-index.md': sourceIndex(
      '| [architecture.md](/tachigo/architecture) | System map |\n| [tokenomics.md](/tachigo/tokenomics) | Token model |',
    ),
    'architecture.md': doc('# System Architecture\n\nSystem map content.'),
    'tokenomics.md': doc('# Tokenomics\n\nToken model content.'),
    'feature-discussion.md': doc('# Feature Discussion\n\nBackground only.'),
    'superpowers/specs/proposed-design.md': doc(
      '# Proposed Design',
      'title: Proposed Design\nstatus: proposed\nsource_of_truth: true\n',
    ),
    'superpowers/plans/implementation-plan.md': doc('# Plan', 'title: Plan\nstatus: active\n'),
    'internal.md': doc('# Internal Note', 'title: Internal Note\nstatus: active\ninternal: true\n'),
    'excluded.md': doc('# Excluded Note', 'title: Excluded Note\nstatus: active\nexcluded_from_llms: true\n'),
  })

  const result = await collectLlmsDocs({docsRoot, baseUrl: '/tachigo/'})

  assert.deepEqual(
    result.docs.map((item) => item.relativePath),
    ['architecture.md', 'dev-portal/source-index.md', 'dev-portal/start-here.md', 'tokenomics.md'],
  )
  assert.equal(result.warnings.length, 0)
})

test('falls back to tier 1 only when the root source table cannot be parsed', async () => {
  const docsRoot = await fixture({
    'dev-portal/start-here.md': doc('# Start Here', 'title: Start Here\nstatus: active\n'),
    'dev-portal/source-index.md': sourceIndex('| [architecture.md](/tachigo/architecture) | System map |', 'Other table'),
    'architecture.md': doc('# System Architecture'),
  })

  const result = await collectLlmsDocs({docsRoot, baseUrl: '/tachigo/'})

  assert.deepEqual(
    result.docs.map((item) => item.relativePath),
    ['dev-portal/source-index.md', 'dev-portal/start-here.md'],
  )
  assert.match(result.warnings.join('\n'), /Root source of truth/)
})

test('parses root source links from the first Root source of truth table', () => {
  assert.deepEqual(
    parseRootSourceOfTruthLinks(
      sourceIndex(
        '| [architecture.md](/tachigo/architecture) | System map |\n| [auth-architecture.md](/tachigo/auth-architecture) | Auth map |\n\n## Other table\n\n| 文件 | 定位 |\n|---|---|\n| [ignored.md](/tachigo/ignored) | Ignore me |',
      ),
    ),
    new Set(['architecture.md', 'auth-architecture.md']),
  )
})

test('builds llms text, full text, manifest, and postBuild files', async () => {
  const docsRoot = await fixture({
    'dev-portal/start-here.md': doc(
      '# Start Here\n\nDev Portal entry point.',
      'title: Start Here\nstatus: active\nowner: engineering\nrelated_issues:\n  - 697\nimplemented_in:\n  - 700\n',
    ),
    'dev-portal/source-index.md': sourceIndex('| [architecture.md](/tachigo/architecture) | System map |'),
    'architecture.md': doc('# System Architecture\n\nSystem map content.'),
  })
  const result = await collectLlmsDocs({docsRoot, baseUrl: '/tachigo/'})
  const artifacts = buildLlmsArtifacts({
    docs: result.docs,
    baseUrl: '/tachigo/',
    generatedAt: '2026-05-14T00:00:00.000Z',
  })

  assert.match(artifacts.llmsTxt, /^# tachigo Dev Portal\n\n> /)
  assert.match(artifacts.llmsTxt, /## Architecture\n- \[System Architecture\]\(\/tachigo\/architecture\): System map content\./)
  assert.match(artifacts.llmsFullTxt, /Source: \/tachigo\/architecture/)
  assert.deepEqual(
    artifacts.manifest.docs.map((item) => item.path),
    ['/tachigo/architecture', '/tachigo/dev-portal/source-index', '/tachigo/dev-portal/start-here'],
  )
  assert.deepEqual(artifacts.manifest.docs.at(-1)?.related_issues, [697])

  const outDir = await fs.mkdtemp(path.join(os.tmpdir(), 'tachigo-llms-build-'))
  await llmsTxtPlugin({siteDir: new URL('.', import.meta.url).pathname}, {docsRoot}).postBuild({outDir})

  assert.match(await fs.readFile(path.join(outDir, 'llms.txt'), 'utf8'), /# tachigo Dev Portal/)
  assert.match(await fs.readFile(path.join(outDir, 'llms-full.txt'), 'utf8'), /Source: \/tachigo\/architecture/)
  assert.equal(JSON.parse(await fs.readFile(path.join(outDir, 'manifest.json'), 'utf8')).docs.length, 3)
})
