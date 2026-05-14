import test from 'node:test'
import assert from 'node:assert/strict'
import {fileURLToPath} from 'node:url'

import {
  collectRelatedLinks,
  default as relatedLinksPlugin,
} from './index.ts'
import {validateFrontmatterDocs} from '../frontmatter-validator/index.ts'

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
