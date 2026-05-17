import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

interface ExtensionManifest {
  background?: {
    service_worker?: string
  }
  content_scripts?: Array<{
    matches?: string[]
    js?: string[]
  }>
  host_permissions?: string[]
  side_panel?: {
    default_path?: string
  }
}

const distUrl = new URL('../dist/', import.meta.url)
const distPath = fileURLToPath(distUrl)
const manifestUrl = new URL('manifest.json', distUrl)
const productionApiUrl = 'https://api.tachigo.io'
const productionApiPermission = `${productionApiUrl}/*`
const productionTwitchMatch = 'https://www.twitch.tv/*'
const localOnlyPattern = /localhost|127\.0\.0\.1|0\.0\.0\.0/
const localApiPattern = /https?:\/\/(?:localhost|127\.0\.0\.1|0\.0\.0\.0):8080\b/
const scannedBundleExtensions = new Set(['.css', '.html', '.js'])

function readManifest(): ExtensionManifest {
  assert.ok(
    existsSync(manifestUrl),
    `Missing production manifest at ${fileURLToPath(manifestUrl)}. Run pnpm build first.`,
  )

  return JSON.parse(readFileSync(manifestUrl, 'utf8')) as ExtensionManifest
}

function assertDistFile(relativePath: string, label: string) {
  const fileUrl = new URL(relativePath, distUrl)

  assert.ok(existsSync(fileUrl), `Missing ${label}: ${relativePath}`)
}

function collectManifestUrls(manifest: ExtensionManifest) {
  return [
    ...(manifest.host_permissions ?? []),
    ...(manifest.content_scripts ?? []).flatMap((script) => script.matches ?? []),
  ]
}

function collectBundleFiles(directoryPath = distPath): Array<{ relativePath: string; contents: string }> {
  return readdirSync(directoryPath, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directoryPath, entry.name)

    if (entry.isDirectory()) {
      return collectBundleFiles(entryPath)
    }

    if (!entry.isFile() || !scannedBundleExtensions.has(path.extname(entry.name))) {
      return []
    }

    return [
      {
        relativePath: path.relative(distPath, entryPath),
        contents: readFileSync(entryPath, 'utf8'),
      },
    ]
  })
}

const manifest = readManifest()

assert.ok(
  manifest.host_permissions?.includes(productionApiPermission),
  `Production manifest must include ${productionApiPermission}`,
)
assert.ok(
  manifest.content_scripts?.some((script) => script.matches?.includes(productionTwitchMatch)),
  `Production manifest must target ${productionTwitchMatch}`,
)

const localOnlyUrls = collectManifestUrls(manifest).filter((url) => localOnlyPattern.test(url))
assert.deepEqual(localOnlyUrls, [], `Production manifest must not include local-only URLs: ${localOnlyUrls.join(', ')}`)

assert.equal(manifest.background?.service_worker, 'assets/background.js')
assert.equal(manifest.side_panel?.default_path, 'sidepanel.html')
assert.ok(
  manifest.content_scripts?.some((script) => script.js?.includes('assets/content.js')),
  'Production manifest must include assets/content.js as a content script.',
)

assertDistFile('assets/background.js', 'background service worker')
assertDistFile('assets/content.js', 'content script')
assertDistFile('sidepanel.html', 'side panel entry')

const bundleFiles = collectBundleFiles()
const localApiBundleFiles = bundleFiles
  .filter(({ contents }) => localApiPattern.test(contents))
  .map(({ relativePath }) => relativePath)

assert.deepEqual(
  localApiBundleFiles,
  [],
  `Production bundle must not include local API URLs: ${localApiBundleFiles.join(', ')}`,
)
assert.ok(
  bundleFiles.some(({ contents }) => contents.includes(productionApiUrl)),
  `Production bundle must include ${productionApiUrl}`,
)

console.log('Production extension package readback passed.')
