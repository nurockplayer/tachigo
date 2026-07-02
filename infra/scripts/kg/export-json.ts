#!/usr/bin/env node

const { mkdir, readFile, writeFile } = require('node:fs/promises')
const path = require('node:path')
const process = require('node:process')

const {
  defaultSeedPath,
  edgeIdentity,
  identity,
  parseSeedYaml,
  validateGraph,
} = require('./validate-kg.ts')

const defaultOutputPath = path.join(process.cwd(), 'kg', 'generated', 'tachigo.graph.json')

function parseArgs(argv) {
  const options = {
    out: defaultOutputPath,
    seed: defaultSeedPath,
  }

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--seed') {
      const value = argv[index + 1]
      if (!value) {
        throw new Error('--seed requires a path')
      }
      options.seed = value
      index += 1
    } else if (arg === '--out') {
      const value = argv[index + 1]
      if (!value) {
        throw new Error('--out requires a path')
      }
      options.out = value
      index += 1
    } else {
      throw new Error(`unknown argument: ${arg}`)
    }
  }

  return options
}

function normalizeMetadata(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value) ? value : {}
}

function isNonEmptyString(value) {
  return typeof value === 'string' && value.trim().length > 0
}

function normalizeGraph(graph, generatedAt = new Date().toISOString()) {
  const nodes = graph.nodes
    .map((node) => ({
      id: identity(node),
      kind: node.kind,
      name: node.name,
      path: isNonEmptyString(node.path) ? node.path : null,
      metadata: normalizeMetadata(node.metadata),
    }))
    .sort((left, right) => left.id.localeCompare(right.id))

  const edges = graph.edges
    .map((edge) => ({
      id: edgeIdentity(edge),
      from: identity(edge.from),
      to: identity(edge.to),
      relation: edge.relation,
      source: edge.source,
      metadata: normalizeMetadata(edge.metadata),
    }))
    .sort((left, right) => left.id.localeCompare(right.id))

  return {
    version: 1,
    generatedAt,
    nodes,
    edges,
  }
}

async function main() {
  const options = parseArgs(process.argv.slice(2))
  const seedPath = path.resolve(options.seed)
  const outPath = path.resolve(options.out)
  const graph = parseSeedYaml(await readFile(seedPath, 'utf8'))
  const problems = validateGraph(graph)

  if (problems.length > 0) {
    console.error('Knowledge graph validation failed:')
    for (const problem of problems) {
      console.error(`- ${problem}`)
    }
    process.exitCode = 1
    return
  }

  const normalizedGraph = normalizeGraph(graph)
  await mkdir(path.dirname(outPath), { recursive: true })
  await writeFile(outPath, `${JSON.stringify(normalizedGraph, null, 2)}\n`)
  console.log('Knowledge graph JSON exported.')
  console.log(`Output: ${outPath}`)
  console.log(`Nodes: ${normalizedGraph.nodes.length}`)
  console.log(`Edges: ${normalizedGraph.edges.length}`)
}

main().catch((error) => {
  console.error(error.message)
  process.exitCode = 1
})
