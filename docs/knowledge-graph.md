# tachigo Knowledge Graph

`tachigo` uses a lightweight repository knowledge graph to connect features,
services, database tables, external systems, documents, issues, and PRs.

The graph is meant for architecture navigation, PR impact analysis, AI coding
assistant context, documentation sync checks, and future GraphRAG support. The
first source of truth is `kg/seeds/tachigo.yaml`.

Generated files under `kg/generated/` are local derived artifacts and should not
be edited by hand. Regenerate them from the seed when needed.

## First Phase

The first phase is intentionally deterministic and dependency-free:

- seed file: `kg/seeds/tachigo.yaml`
- validator: `infra/scripts/kg/validate-kg.ts`
- JSON exporter: `infra/scripts/kg/export-json.ts`
- local command: `make kg-validate`
- local JSON export command: `make kg-export-json`

This phase does not add Mermaid export, impact analysis, CI wiring, LLM
extraction, embeddings, Neo4j, pgvector, runtime APIs, or production DB
migrations. Those are future phases.

## Node Kinds

Allowed node kinds:

- `Feature`
- `Service`
- `APIEndpoint`
- `DatabaseTable`
- `Migration`
- `Document`
- `File`
- `Package`
- `Decision`
- `ExternalSystem`
- `Issue`
- `PR`

Node identity is `kind:name`. A seed cannot contain two nodes with the same
identity.

## Edge Relations

Allowed relations:

- `IMPLEMENTS`
- `IMPLEMENTED_BY`
- `DEPENDS_ON`
- `CALLS`
- `READS`
- `WRITES`
- `DOCUMENTED_BY`
- `DECIDED_BY`
- `MIGRATED_BY`
- `EXPOSES`
- `CONFIGURES`
- `SYNC_WITH`
- `AFFECTS`
- `RELATED_TO`

Edge identity is `from-relation-to`. A seed cannot contain duplicate edge
identities.

## Seed Format

`kg/seeds/tachigo.yaml` uses a narrow YAML subset so it can be validated without
adding package dependencies. The supported structure is:

```yaml
nodes:
  - kind: Feature
    name: Watch Points
    path: apps/extension
    metadata:
      description: Viewers earn off-chain points from watch heartbeat activity.

edges:
  - from:
      kind: Feature
      name: Watch Points
    relation: IMPLEMENTED_BY
    to:
      kind: Service
      name: ExtensionService
    source: docs/architecture.md
```

Each edge must include a non-empty `source` that points to the document or file
used as the evidence for that relationship.

## Validate

Run:

```bash
make kg-validate
```

The validator checks:

- node kind allowlist
- edge relation allowlist
- duplicate node identities
- duplicate edge identities
- missing edge `from` / `to` endpoints
- edge endpoints that do not match declared nodes
- missing edge `source`

Expected success output:

```txt
Knowledge graph validation passed.
Nodes: 22
Edges: 15
Kinds: APIEndpoint=1, DatabaseTable=6, Document=3, ExternalSystem=5, Feature=3, Service=4
Relations: DEPENDS_ON=4, DOCUMENTED_BY=1, EXPOSES=1, IMPLEMENTED_BY=2, READS=1, SYNC_WITH=2, WRITES=4
```

## Export JSON

Run:

```bash
make kg-export-json
```

This validates `kg/seeds/tachigo.yaml` before writing normalized JSON to:

```txt
kg/generated/tachigo.graph.json
```

The exported JSON includes stable `id` values for nodes and edges:

```json
{
  "version": 1,
  "generatedAt": "2026-05-31T00:00:00.000Z",
  "nodes": [
    {
      "id": "Feature:Watch Points",
      "kind": "Feature",
      "name": "Watch Points",
      "path": "apps/extension",
      "metadata": {}
    }
  ],
  "edges": [
    {
      "id": "Feature:Watch Points-IMPLEMENTED_BY-Service:ExtensionService",
      "from": "Feature:Watch Points",
      "to": "Service:ExtensionService",
      "relation": "IMPLEMENTED_BY",
      "source": "docs/architecture.md",
      "metadata": {}
    }
  ]
}
```

Generated JSON files under `kg/generated/` are ignored by git because
`generatedAt` changes on each export. Regenerate them locally from the seed
instead of editing or reviewing them by hand.

## Future Phases

Future work should stay issue-first and small:

1. Export Mermaid into `kg/generated/tachigo.mmd`.
2. Add deterministic `impact --files` analysis.
3. Add lightweight CI validation once the seed and validator stabilize.
4. Add deterministic code-aware extractors for Go routes, GORM models,
   migrations, frontend routes, and contract surfaces.
5. Only after the deterministic graph proves useful, evaluate embeddings or
   GraphRAG.

## Non-Goals

This graph is not a production runtime database and is not business data. It
must not change backend runtime behavior, database migrations, auth contracts,
frontend behavior, token flows, or deployment behavior by itself.
