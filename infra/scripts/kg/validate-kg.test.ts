const assert = require('node:assert/strict')
const { execFileSync } = require('node:child_process')
const { mkdirSync, writeFileSync } = require('node:fs')
const { mkdtemp, rm } = require('node:fs/promises')
const { tmpdir } = require('node:os')
const path = require('node:path')
const test = require('node:test')

const validatorPath = path.join(__dirname, 'validate-kg.ts')

async function withSeed(content, fn) {
  const root = await mkdtemp(path.join(tmpdir(), 'tachigo-kg-'))
  try {
    const seedPath = path.join(root, 'kg', 'seeds', 'tachigo.yaml')
    mkdirSync(path.dirname(seedPath), { recursive: true })
    writeFileSync(seedPath, content)
    await fn(seedPath)
  } finally {
    await rm(root, { recursive: true, force: true })
  }
}

function runValidator(seedPath) {
  return execFileSync('node', ['--experimental-strip-types', '--no-warnings', validatorPath, '--seed', seedPath], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  })
}

function runValidatorFailure(seedPath) {
  assert.throws(
    () => runValidator(seedPath),
    (error) => {
      assert.notEqual(error.status, 0)
      return true
    },
  )
}

const validSeed = `
nodes:
  - kind: Feature
    name: Watch Points
    path: apps/extension
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

test('validates a repository knowledge graph seed', async () => {
  await withSeed(validSeed, async (seedPath) => {
    const output = runValidator(seedPath)

    assert.match(output, /Knowledge graph validation passed\./)
    assert.match(output, /Nodes: 3/)
    assert.match(output, /Edges: 2/)
    assert.match(output, /Feature=1/)
    assert.match(output, /IMPLEMENTED_BY=1/)
  })
})

test('rejects duplicate node identities', async () => {
  await withSeed(`${validSeed}
nodes:
  - kind: Feature
    name: Watch Points
  - kind: Feature
    name: Watch Points
`, async (seedPath) => {
    runValidatorFailure(seedPath)
  })
})

test('rejects edges that reference missing nodes', async () => {
  await withSeed(`
nodes:
  - kind: Feature
    name: Watch Points
edges:
  - from:
      kind: Feature
      name: Watch Points
    relation: IMPLEMENTED_BY
    to:
      kind: Service
      name: MissingService
    source: docs/architecture.md
`, async (seedPath) => {
    runValidatorFailure(seedPath)
  })
})

test('rejects edges without a source document', async () => {
  await withSeed(`
nodes:
  - kind: Feature
    name: Watch Points
  - kind: Service
    name: PointsService
edges:
  - from:
      kind: Feature
      name: Watch Points
    relation: IMPLEMENTED_BY
    to:
      kind: Service
      name: PointsService
`, async (seedPath) => {
    runValidatorFailure(seedPath)
  })
})
