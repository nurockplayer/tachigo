# Tasks

- [x] Add OpenSpec workflow documentation under `docs/ai/`.
- [x] Add repo-level `openspec/` scaffold and config.
- [x] Update `AGENTS.md` and `CLAUDE.md` with OpenSpec SDD rules.
- [x] Update PR template with OpenSpec evidence fields.
- [x] Update docs indexes.
- [x] Add CI regression coverage for OpenSpec wiring.
- [x] Run `.github/workflows/ci.test.ts` and `git diff --check`.

## Verification Evidence

- OpenSpec workflow documentation: `docs/ai/openspec-workflow.md` exists and defines proposal/spec/tasks as the source of truth.
- OpenSpec scaffold and config: `openspec/config.yaml`, `openspec/README.md`, `openspec/changes/README.md`, and `openspec/specs/README.md` exist; `openspec/config.yaml` requires task validation evidence.
- Agent instructions: `AGENTS.md` and `CLAUDE.md` contain the OpenSpec SDD workflow rules and PR scope expectations.
- PR template: `.github/PULL_REQUEST_TEMPLATE.md` includes OpenSpec evidence fields for change ID, tasks, and spec impact.
- Docs indexes: `docs/README.md`, `docs/ai/README.md`, and `docs/dev-portal/source-index.md` link the OpenSpec workflow documentation.
- CI regression coverage: `.github/workflows/ci.test.ts` covers OpenSpec wiring and repository workflow expectations.
- Validation commands:
  - `node --experimental-strip-types --no-warnings --test .github/workflows/ci.test.ts` — 104/104 passed.
  - `git diff --check` — passed with no whitespace errors.
