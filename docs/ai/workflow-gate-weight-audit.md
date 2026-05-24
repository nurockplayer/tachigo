# AI Workflow Gate Weight Audit

Source issue: #884
Baseline input: #881 / [workflow-metrics-baseline.md](workflow-metrics-baseline.md)

This audit classifies the current AI / PR governance surface by observed risk
reduction and workflow cost. It is intentionally docs-only: no PR template,
Scope Police, CI, auto-ready, label, or branch protection behavior changes are
made here.

## Baseline Signals

The #881 baseline sampled the latest 30 pull requests at 2026-05-24 01:14
Asia/Taipei and found:

| Signal | Baseline | Gate-weight implication |
| --- | ---: | --- |
| PR body has `Source of truth` | 30 / 30 | Keep as a hard metadata gate. The field is now cheap and consistently present. |
| Issue-to-first-PR elapsed time | median 224.21 hours across 29 / 30 PRs | Use only for analysis. Umbrella issues make this noisy and should not drive blocking checks. |
| Historical Scope Police friction | 9 / 30 PRs had at least one Scope-related failed check | Treat as optimization input. Historical failures can remain after latest-head pass. |
| Review coverage | 29 / 30 PRs had at least one review record | Keep latest-head review readback for merge decisions, but separate human and automated review signals. |
| `CHANGES_REQUESTED` frequency | 4 / 30 PRs | Keep blocker handling, but key it by latest head SHA to avoid stale-review drag. |
| Rework proxy | 7 / 30 PRs had more than one commit | Use as a sampling metric, not a gate. Multi-commit PRs are not automatically bad. |

The immediate conclusion is that fields with strong review alignment should stay
hard gates, while historical or noisy counters should move to readback / audit
use instead of becoming more template weight.

## Gate Classification

### Keep

These checks have clear risk-reduction value and should remain hard or near-hard
gates for normal PRs.

| Surface | Current role | Why keep |
| --- | --- | --- |
| `Source of truth` | PR body required field | 30 / 30 sampled PRs already comply, and it anchors scope review to an issue, PR, or doc instead of reviewer memory. |
| `Depends on PR` | PR body required field | Prevents frontend / docs / workflow PRs from silently depending on unmerged contracts or policy baselines. |
| `本 PR 明確不做` | PR body required field | High signal against scope creep; cheap for both human and AI-authored PRs. |
| One selected `PR Risk Class` | PR body required field | Keeps R4 visibility explicit and blocks R4 + `auto-ready`. |
| File count and hard diff limits | Scope Police hard gate | Still useful as an early reviewability guard, especially before CI spends heavier resources. |
| Product surface separation | Scope Police hard gate | Directly prevents backend/frontend/contract mixed PRs from bypassing review focus. |
| Latest-head check readback | Merge/review practice | Needed because status rollups retain stale failures and stale approvals. |
| Latest-head review state | Merge/review practice | Required to avoid treating old `CHANGES_REQUESTED` or approval as current. |

### Merge / Consolidate

These are useful, but the repo should avoid making reviewers read the same
evidence twice.

| Surface | Current overlap | Recommended consolidation |
| --- | --- | --- |
| `Delegation Execution Log` and `spec workflow-check evidence` | Both prove autonomous routing / bounded context | Keep both slots, but PR bodies should store short status + evidence ref only. Full start / commit / merge details belong in local evidence, PR comments, or issue comments. |
| `Review conversation closeout` and final merge gate fields | Both can describe review readiness | Keep review disposition in closeout; keep final gate to latest head SHA, checks, and reviewer action. Avoid repeating the full review matrix. |
| #664 threshold ledger and PR body workflow friction notes | Both describe routing friction | Keep #664 as the durable ledger. PR body should only include `#664` comment URL or `n/a`. |
| `docs/ai/autonomous-pr-gates.md` and `docs/ai/codex-autonomous-workflow.md` | Both describe AWP evidence discipline | Keep `autonomous-bootstrap.md` as the entrypoint, and gradually move duplicated policy details behind links instead of adding more text to the PR template. |

### Downgrade By Risk Class

These checks are valuable for R3/R4 or explicitly autonomous work, but too heavy
when applied with the same weight to R0/R1 docs, tooling, and metadata PRs.

| Surface | Current cost | Recommended lower-weight behavior |
| --- | --- | --- |
| Backend contract checkbox | Useful for product PRs; noise for docs/template/metadata | Keep current docs/template skip. Do not require it for pure R0 docs or template PRs. |
| Full autonomous evidence sections | Necessary for autonomous implementation PRs; bulky for tiny docs-only cuts | For R0/R1 controller-direct cuts, allow a concise trivial/self-only exception plus `spec plan` / manual checklist evidence. |
| Review triage refs | Valuable after actual findings; awkward before any review exists | Keep `pending with reason` at PR creation. Require concrete refs only after findings exist or at final ready-to-merge closeout. |
| Scope Police historical failure count | Useful friction metric; stale failures remain in rollup | Never block on historical failures after the latest matching check for the current head passes. |
| Issue-to-first-PR elapsed time | Good process metric; umbrella issues skew it | Use for monthly/track audit only, not PR-level gating. |

### Remove / Defer

These should not become new mandatory fields or checks without more evidence.

| Candidate | Reason |
| --- | --- |
| Adding `spawn_count`, CI rerun count, or review-thread count to the PR template | #664 already stores threshold data. Duplicating it in every PR adds weight without improving merge safety. |
| Automated gates based on elapsed time from issue creation | The #881 sample is noisy because umbrella issues stay open across many small cuts. |
| Parsing full review triage schema inside tachigo Scope Police | The source of truth is spec-injector. tachigo should keep thin evidence refs instead of duplicating parser logic. |
| `blocked-by-dependency` label side-effect | `gh label list --search blocked-by-dependency` and `gh pr list --label blocked-by-dependency` both return no active label/PR signal. Keep the dependency failure and sticky comment, but remove the unused label management. |
| Making CodeRabbit / connector timing a hard initial-review blocker for all R0/R1 PRs | Useful as review input, but rate limits and skipped contexts should not freeze low-risk docs/tooling work when human review is available. |
| New product-AI planning fields in the PR template | Track B product docs are planning references, not implementation contracts. Product AI features still need their own issues and specs. |

## Track A Exit Criteria

Track A is the AI-native repo / workflow foundation. It should end when the repo
has enough evidence and command surface to make autonomous work predictable
without continuing to add meta-work.

Track A can exit when all of these are true:

1. `make ai-*` command surface exists and covers common readback / validation
   paths for docs, backend, extension, dashboard, and PR metadata.
2. `docs/ai/workflow-metrics-baseline.md` exists and has been used by at least
   one gate-weight audit.
3. Scope Police and PR template policy have a documented lightweight path for
   R0/R1 docs, template, metadata, and tooling PRs.
4. R4 boundaries remain explicit for auth, permissions, schema, migration,
   wallet, payments, ledger, workflow runtime, and release changes.
5. Autonomous PR bodies store evidence refs instead of expanding full routing,
   threshold, and review matrices inline.
6. #664 threshold ledger has enough recent data to make a keep / adjust / stop
   decision, and new metrics are no longer being added to the PR template.
7. Open follow-up work is expressed as small implementation issues, not broad
   "continue optimizing AI workflow" meta-issues.

After these criteria are met, new AI workflow work should require one of:

- a concrete bug in the command surface, Scope Police, CI lane, or review label
  automation;
- a measured regression from the baseline;
- an implementation issue that changes one small policy/runtime surface.

## Follow-Up Implementation Cuts

These are the implementation-sized follow-ups implied by the audit. They should
be opened or implemented separately; this PR only records the split.

| Cut | Purpose | Suggested scope | Validation |
| --- | --- | --- | --- |
| R0/R1 template slimming | Reduce repeated evidence text for low-risk PRs | PR template wording only; no Scope Police runtime change | `git diff --check`, PR metadata fixture if template parser is touched |
| Scope Police latest-head summary | Make sticky comment distinguish historical failures from latest-head blockers | `.github/workflows/pr-scope-police.yml` plus workflow regression tests | targeted workflow tests and `actionlint` |
| Autonomous evidence ref guidance | Move duplicated AWP evidence prose behind docs links | `docs/ai/autonomous-pr-gates.md` and `docs/ai/codex-autonomous-workflow.md` only | docs review and `git diff --check` |
| #664 stop/freeze policy | Define when threshold ledger stops being updated | #664 issue comment or one docs note; no PR template fields | public readback plus docs diff check |
| R4 floor review | Confirm whether workflow/runtime paths should auto-upgrade to R4 | policy docs first, then separate Scope Police implementation if approved | manual sample plus workflow regression |

## Current Recommendation

Do not weaken high-risk protection. Instead:

- keep hard scope anchors that are already cheap and consistently used;
- downgrade noisy historical counters to audit-only metrics;
- consolidate autonomous evidence into short refs;
- stop adding template fields unless the #881 baseline shows a concrete missed
  risk that existing fields cannot catch.
