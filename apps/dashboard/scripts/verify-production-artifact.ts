import assert from 'node:assert/strict'
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const distUrl = new URL('../dist/', import.meta.url)
const productionApiOrigin = 'https://api.tachigo.io'
const localOnlyPattern = /localhost|127\.0\.0\.1|0\.0\.0\.0/

function assertFile(relativePath: string, label: string) {
  const fileUrl = new URL(relativePath, distUrl)
  assert.ok(existsSync(fileUrl), `Missing ${label}: ${relativePath}`)
  assert.ok(statSync(fileUrl).isFile(), `${label} is not a file: ${relativePath}`)
}

function readText(relativePath: string) {
  return readFileSync(new URL(relativePath, distUrl), 'utf8')
}

function listAssetFiles() {
  const assetsUrl = new URL('assets/', distUrl)
  assert.ok(existsSync(assetsUrl), `Missing assets directory: ${fileURLToPath(assetsUrl)}`)

  return readdirSync(assetsUrl).map((fileName) => `assets/${fileName}`)
}

function assertNoLocalOnlyURLs(relativePath: string, contents: string) {
  assert.equal(
    localOnlyPattern.test(contents),
    false,
    `${relativePath} must not include localhost / loopback URLs in a production artifact.`,
  )
}

assert.ok(existsSync(distUrl), `Missing dashboard dist directory: ${fileURLToPath(distUrl)}. Run pnpm build first.`)
assertFile('index.html', 'dashboard HTML entry')

const assetFiles = listAssetFiles()
const javascriptAssets = assetFiles.filter((fileName) => fileName.endsWith('.js'))
assert.ok(javascriptAssets.length > 0, 'Production artifact must include at least one JavaScript asset.')

const filesToInspect = ['index.html', ...javascriptAssets]
const inspectedContents = filesToInspect.map((fileName) => [fileName, readText(fileName)] as const)

for (const [fileName, contents] of inspectedContents) {
  assertNoLocalOnlyURLs(fileName, contents)
}

assert.ok(
  inspectedContents.some(([, contents]) => contents.includes(productionApiOrigin)),
  `Production artifact must embed ${productionApiOrigin}.`,
)

console.log('Dashboard production artifact readback passed.')
