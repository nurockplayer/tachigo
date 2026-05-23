# AI Workflow Metrics Baseline

Source issue: #881
Sample window: latest 30 pull requests returned by `gh pr list --state all --limit 30` at 2026-05-24 01:14 Asia/Taipei.

This note defines a lightweight baseline for AI-native repo workflow friction using only GitHub metadata that already exists on issues, pull requests, reviews, commits, and checks. It is intentionally a docs-only baseline: no telemetry pipeline, dashboard, PR template expansion, or Scope Police behavior change is part of this cut.

## Immediately Measurable

These metrics can be computed from current GitHub metadata without adding instrumentation.

| Metric | Baseline | Source field | Caveat |
| --- | ---: | --- | --- |
| PR body has a `Source of truth` issue | 30 / 30 | PR body | This confirms the PR contract is present, not that the issue scope was semantically correct. |
| Issue-to-first-PR elapsed time | median 224.21 hours, min 0.47, max 940.45 across 29 / 30 PRs | PR `createdAt` plus source issue `createdAt` | One source reference, #877, did not resolve through `gh issue view`, so it is excluded. Reused umbrella issues skew the metric upward. |
| Scope Police friction frequency | 9 / 30 PRs had at least one Scope-related failed check in `statusCheckRollup` | PR check rollup | This is a historical friction signal, not necessarily the current blocking state; stale failed attempts can remain in the rollup after a passing rerun. |
| Review coverage | 29 / 30 PRs had at least one review record | PR reviews | Automated and human review records should be separated before using this as a quality metric. |
| Change-request frequency | 4 / 30 PRs had a `CHANGES_REQUESTED` review | PR reviews | This does not distinguish resolved from still-active blocker state. |
| Rework proxy | 7 / 30 PRs had more than one commit | PR commits | Extra commits are only a proxy for rework; they may also represent intentional incremental commits. |

## Manual Sampling Needed

These signals are visible today but require human classification or a stricter convention before they should be used as automated gates.

- Source issue fit: sample whether the source issue acceptance criteria actually match the PR behavior, especially for umbrella issues reused by many small cuts.
- Scope Police cause: classify failures as missing metadata, stale source branch, large diff, generated-file mismatch, or true scope pollution.
- Review type: separate Codex, CodeRabbitAI, Cloudflare, GitHub Actions, and human maintainer signals before calculating review load.
- Rework cause: classify multi-commit PRs by CI fix, reviewer blocker, author polish, or merge-base refresh.
- AWP/spec-injector evidence quality: check whether PR bodies include bounded task context, dry-run source, worker/fallback log, and local validation evidence.

## Gate-Weight Audit Inputs

The later gate-weight audit should read these fields before changing any threshold:

- PR body: `Source of truth`, risk class, scope summary, local validation, and autonomous/delegation log.
- Issue metadata: issue creation time, labels, acceptance criteria, and whether it is an umbrella or single-cut issue.
- Review metadata: latest review state per actor and whether a change request was superseded by a new head SHA.
- Check metadata: latest check conclusion per head SHA, plus historical failed attempts for friction analysis.
- Commit metadata: commit count, head SHA transitions, and whether extra commits followed review or CI feedback.

Recommended split for the audit:

- Keep the current `Source of truth` requirement as a hard metadata gate.
- Treat Scope Police historical failures as an optimization metric, not a merge blocker once the latest head passes.
- Use manual sampling before weighting umbrella-issue elapsed time, because long-lived source issues make the elapsed metric noisy.
- Prefer small, source-issue-specific PRs when measuring cycle time; umbrella issue slices should be tracked separately.
