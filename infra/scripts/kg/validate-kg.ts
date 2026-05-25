#!/usr/bin/env node

const { readFile } = require('node:fs/promises')
const path = require('node:path')
const process = require('node:process')

const defaultSeedPath = path.join(process.cwd(), 'kg', 'seeds', 'tachigo.yaml')

const allowedKinds = new Set([
  'Feature',
  'Service',
  'APIEndpoint',
  'DatabaseTable',
  'Migration',
  'Document',
  'File',
  'Package',
  'Decision',
  'ExternalSystem',
  'Issue',
  'PR',
])

const allowedRelations = new Set([
  'IMPLEMENTS',
  'IMPLEMENTED_BY',
  'DEPENDS_ON',
  'CALLS',
  'READS',
  'WRITES',
  'DOCUMENTED_BY',
  'DECIDED_BY',
  'MIGRATED_BY',
  'EXPOSES',
  'CONFIGURES',
  'SYNC_WITH',
  'AFFECTS',
  'RELATED_TO',
])

function parseArgs(argv) {
  const options = { seed: defaultSeedPath }
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--seed') {
      const value = argv[index + 1]
      if (!value) {
        throw new Error('--seed requires a path')
      }
      options.seed = value
      index += 1
    } else {
      throw new Error(`unknown argument: ${arg}`)
    }
  }
  return options
}

function parseScalar(rawValue) {
  const value = rawValue.trim()
  if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
    return value.slice(1, -1)
  }
  if (value === '{}') {
    return {}
  }
  return value
}

function assignKey(target, content, lineNumber) {
  const match = content.match(/^([^:]+):(?:\s*(.*))?$/)
  if (!match) {
    throw new Error(`line ${lineNumber}: expected key/value pair`)
  }

  const key = match[1].trim()
  const rawValue = match[2] ?? ''
  if (!key) {
    throw new Error(`line ${lineNumber}: key cannot be empty`)
  }

  if (rawValue.trim() === '') {
    target[key] = {}
    return key
  }

  target[key] = parseScalar(rawValue)
  return null
}

function parseSeedYaml(content) {
  const graph = { nodes: [], edges: [] }
  let section = null
  let current = null
  let nestedKey = null

  const lines = content.split(/\r?\n/)
  for (const [lineIndex, rawLine] of lines.entries()) {
    const lineNumber = lineIndex + 1
    const withoutComment = rawLine.replace(/\s+#.*$/, '')
    if (!withoutComment.trim()) {
      continue
    }

    const indent = withoutComment.match(/^ */)[0].length
    const contentLine = withoutComment.trimEnd()

    if (indent === 0) {
      const match = contentLine.match(/^(nodes|edges):$/)
      if (!match) {
        throw new Error(`line ${lineNumber}: expected top-level nodes: or edges:`)
      }
      section = match[1]
      current = null
      nestedKey = null
      continue
    }

    if (!section) {
      throw new Error(`line ${lineNumber}: entry appears before a section`)
    }

    if (indent === 2 && contentLine.trimStart().startsWith('- ')) {
      current = {}
      graph[section].push(current)
      nestedKey = null

      const rest = contentLine.trimStart().slice(2).trim()
      if (rest) {
        nestedKey = assignKey(current, rest, lineNumber)
      }
      continue
    }

    if (!current) {
      throw new Error(`line ${lineNumber}: key appears before a list item`)
    }

    if (indent === 4) {
      nestedKey = assignKey(current, contentLine.trim(), lineNumber)
      continue
    }

    if (indent === 6 && nestedKey) {
      assignKey(current[nestedKey], contentLine.trim(), lineNumber)
      continue
    }

    throw new Error(`line ${lineNumber}: unsupported indentation or structure`)
  }

  return graph
}

function identity(node) {
  return `${node.kind}:${node.name}`
}

function edgeIdentity(edge) {
  return `${identity(edge.from)}-${edge.relation}-${identity(edge.to)}`
}

function countBy(values) {
  const counts = new Map()
  for (const value of values) {
    counts.set(value, (counts.get(value) || 0) + 1)
  }
  return [...counts.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${key}=${value}`)
    .join(', ')
}

function validateGraph(graph) {
  const problems = []
  const nodeIds = new Set()
  const edgeIds = new Set()

  if (graph.nodes.length === 0) {
    problems.push('nodes must not be empty')
  }

  for (const node of graph.nodes) {
    if (!allowedKinds.has(node.kind)) {
      problems.push(`unsupported node kind: ${node.kind || '(missing)'}`)
    }
    if (!node.name) {
      problems.push(`node ${node.kind || '(missing kind)'} is missing name`)
      continue
    }

    const id = identity(node)
    if (nodeIds.has(id)) {
      problems.push(`duplicate node identity: ${id}`)
    }
    nodeIds.add(id)
  }

  for (const edge of graph.edges) {
    if (!edge.from?.kind || !edge.from?.name) {
      problems.push('edge is missing from.kind/from.name')
    }
    if (!edge.to?.kind || !edge.to?.name) {
      problems.push('edge is missing to.kind/to.name')
    }
    if (!allowedRelations.has(edge.relation)) {
      problems.push(`unsupported edge relation: ${edge.relation || '(missing)'}`)
    }
    if (!edge.source) {
      problems.push(`edge ${edge.relation || '(missing relation)'} is missing source`)
    }

    if (edge.from?.kind && edge.from?.name && !nodeIds.has(identity(edge.from))) {
      problems.push(`edge references missing from node: ${identity(edge.from)}`)
    }
    if (edge.to?.kind && edge.to?.name && !nodeIds.has(identity(edge.to))) {
      problems.push(`edge references missing to node: ${identity(edge.to)}`)
    }

    if (edge.from?.kind && edge.from?.name && edge.to?.kind && edge.to?.name && edge.relation) {
      const id = edgeIdentity(edge)
      if (edgeIds.has(id)) {
        problems.push(`duplicate edge identity: ${id}`)
      }
      edgeIds.add(id)
    }
  }

  return problems
}

async function main() {
  const options = parseArgs(process.argv.slice(2))
  const seedPath = path.resolve(options.seed)
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

  console.log('Knowledge graph validation passed.')
  console.log(`Nodes: ${graph.nodes.length}`)
  console.log(`Edges: ${graph.edges.length}`)
  console.log(`Kinds: ${countBy(graph.nodes.map((node) => node.kind))}`)
  console.log(`Relations: ${countBy(graph.edges.map((edge) => edge.relation))}`)
}

main().catch((error) => {
  console.error(error.message)
  process.exitCode = 1
})
