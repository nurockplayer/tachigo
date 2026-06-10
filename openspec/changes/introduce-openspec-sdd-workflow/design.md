# Design

OpenSpec becomes the source of truth for feature proposal, specs, design, and tasks. GitHub issues remain the public source issue, and spec-injector remains the autonomous local-only evidence gate.

## Integration Points

- `docs/ai/openspec-workflow.md` defines when to use OpenSpec and how artifacts map to PR evidence.
- `AGENTS.md` and `CLAUDE.md` tell agents to read OpenSpec artifacts before implementation.
- `.github/PULL_REQUEST_TEMPLATE.md` collects OpenSpec change path, artifact review, delta spec, task alignment, and archive status.
- `openspec/config.yaml` captures tachigo context and artifact rules without installing dependencies.

## Risks

- Agents may confuse proposed delta specs with accepted living specs; docs state `openspec/specs/**` only updates after acceptance / archive.
- Agents may try to install OpenSpec tooling; docs preserve supply-chain restrictions and allow manual artifact fallback.
