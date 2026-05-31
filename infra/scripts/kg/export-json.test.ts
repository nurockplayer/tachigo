const assert = require('node:assert/strict')
const { execFileSync } = require('node:child_process')
const { existsSync, mkdirSync, readFileSync, writeFileSync } = require('node:fs')
const { mkdtemp, rm } = require('node:fs/promises')
const { tmpdir } = require('node:os')
const path = require('node:path')
const test = require('node:test')

const exporterPath = path.join(__dirname, 'export-json.ts')

async function withSeed(content, fn) {
  const root = await mkdtemp(path.join(tmpdir(), 'tachigo-kg-json-'))
  try {
    const seedPath = path.join(root, 'kg', 'seeds', 'tachigo.yaml')
    mkdirSync(path.dirname(seedPath), { recursive: true })
    writeFileSync(seedPath, content)
    await fn({ root, seedPath })
  } finally {
    await rm(root, { recursive: true, force: true })
  }
}

function runExporter(seedPath, outPath) {
  return execFileSync('node', [
    '--experimental-strip-types',
    '--no-warnings',
    exporterPath,
    '--seed',
    seedPath,
    '--out',
    outPath,
  ], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  })
}

const validSeed = `
nodes:
  - kind: Feature
    name: Watch Points
    path: apps/extension
    metadata:
      description: Viewers earn off-chain points.
  - kind: Service
    name: PointsService
    path: services/api
  - kind: DatabaseTable
    name: points_ledger

edges:
  - from:
      kind: Feature
      name: Watch Points
    relation: IMPLEMENTED_BY
    to:
      kind: Service
      name: PointsService
    source: docs/architecture.md
  - from:
      kind: Feature
      name: Watch Points
    relation: WRITES
    to:
      kind: DatabaseTable
      name: points_ledger
    source: docs/architecture.md
`

test('exports a normalized repository knowledge graph JSON file', async () => {
  await withSeed(validSeed, async ({ root, seedPath }) => {
    const outPath = path.join(root, 'kg', 'generated', 'tachigo.graph.json')
    const output = runExporter(seedPath, outPath)
    const graph = JSON.parse(readFileSync(outPath, 'utf8'))

    assert.match(output, /Knowledge graph JSON exported\./)
    assert.equal(graph.version, 1)
    assert.match(graph.generatedAt, /^\d{4}-\d{2}-\d{2}T/)
    assert.deepEqual(graph.nodes.find((node) => node.id === 'Feature:Watch Points'), {
      id: 'Feature:Watch Points',
      kind: 'Feature',
      name: 'Watch Points',
      path: 'apps/extension',
      metadata: {
        description: 'Viewers earn off-chain points.',
      },
    })
    assert.deepEqual(graph.nodes.find((node) => node.id === 'DatabaseTable:points_ledger'), {
      id: 'DatabaseTable:points_ledger',
      kind: 'DatabaseTable',
      name: 'points_ledger',
      path: null,
      metadata: {},
    })
    assert.deepEqual(graph.nodes.map((node) => node.id), [
      'DatabaseTable:points_ledger',
      'Feature:Watch Points',
      'Service:PointsService',
    ])
    assert.deepEqual(graph.edges.find((edge) => edge.id === 'Feature:Watch Points-IMPLEMENTED_BY-Service:PointsService'), {
      id: 'Feature:Watch Points-IMPLEMENTED_BY-Service:PointsService',
      from: 'Feature:Watch Points',
      to: 'Service:PointsService',
      relation: 'IMPLEMENTED_BY',
      source: 'docs/architecture.md',
      metadata: {},
    })
  })
})

test('rejects invalid seeds before writing JSON output', async () => {
  await withSeed(`
nodes:
  - kind: Feature
    name: Watch Points
edges:
  - from:
      kind: Feature
      name: Watch Points
    relation: WRITES
    to:
      kind: DatabaseTable
      name: missing_table
    source: docs/architecture.md
`, async ({ root, seedPath }) => {
    const outPath = path.join(root, 'kg', 'generated', 'tachigo.graph.json')
    assert.throws(
      () => runExporter(seedPath, outPath),
      (error) => {
        assert.notEqual(error.status, 0)
        assert.match(String(error.stderr ?? error.message ?? ''), /edge references missing to node/i)
        return true
      },
    )
    assert.equal(existsSync(outPath), false)
  })
})
