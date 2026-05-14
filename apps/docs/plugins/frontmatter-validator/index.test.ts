import test from 'node:test'
import assert from 'node:assert/strict'

import {
  default as frontmatterValidatorPlugin,
  validateFrontmatterDocs,
} from './index.ts'

test('scans docs/index.md so validator cannot no-op', async () => {
  const result = await validateFrontmatterDocs({
    docsRoot: new URL('./__fixtures__/valid-docs/', import.meta.url),
  })

  assert.ok(
    result.files.some((file) => file.relativePath === 'index.md'),
    'expected docs/index.md to be scanned',
  )
})

test('fails when related_issues contains a string', async () => {
  await assert.rejects(
    () =>
      validateFrontmatterDocs({
        docsRoot: new URL('./__fixtures__/invalid-related-issues/', import.meta.url),
      }),
    (error) => {
      assert.match(error.message, /index\.md/)
      assert.match(error.message, /related_issues/)
      return true
    },
  )
})

test('accepts unquoted YAML date scalars after normalization', async () => {
  const result = await validateFrontmatterDocs({
    docsRoot: new URL('./__fixtures__/unquoted-date/', import.meta.url),
  })

  const doc = result.files.find((file) => file.relativePath === 'index.md')
  assert.equal(doc.frontmatter.last_reviewed, '2026-05-13')
})

test('accepts reverse_index_scope values nested under code_areas', async () => {
  const result = await validateFrontmatterDocs({
    docsRoot: new URL('./__fixtures__/valid-reverse-scope/', import.meta.url),
  })

  const doc = result.files.find((file) => file.relativePath === 'index.md')
  assert.deepEqual(doc.frontmatter.reverse_index_scope, ['services/api/internal/watchtime'])
})

test('fails when reverse_index_scope falls outside code_areas', async () => {
  await assert.rejects(
    () =>
      validateFrontmatterDocs({
        docsRoot: new URL('./__fixtures__/invalid-reverse-scope/', import.meta.url),
      }),
    (error) => {
      assert.match(error.message, /index\.md/)
      assert.match(error.message, /reverse_index_scope/)
      return true
    },
  )
})

test('plugin lifecycle loads content and publishes global data', async () => {
  const plugin = frontmatterValidatorPlugin(
    {siteDir: new URL('.', import.meta.url).pathname},
    {docsRoot: new URL('./__fixtures__/valid-docs/', import.meta.url)},
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

  assert.ok(content.files.some((file) => file.relativePath === 'index.md'))
  assert.equal(globalData, content)
})

test('plugin lifecycle respects include option', async () => {
  const plugin = frontmatterValidatorPlugin(
    {siteDir: new URL('.', import.meta.url).pathname},
    {
      docsRoot: new URL('./__fixtures__/include-option/', import.meta.url),
      include: ['included.md'],
    },
  )
  const content = await plugin.loadContent()

  assert.deepEqual(
    content.files.map((file) => file.relativePath),
    ['included.md'],
  )
})
