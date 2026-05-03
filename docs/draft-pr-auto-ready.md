# Draft PR auto-ready proposal

**Status**: proposed
**Created**: 2026-05-02
**Issue**: https://github.com/nurockplayer/tachigo/issues/470

## Context

Codex task PRs are often opened while GitHub Actions and external review checks
are still running. Opening them as draft PRs would make the state clearer: the PR
is published for CI, but not yet asking for human review. Once the required
checks are green, the PR can become ready for review automatically.

This is adjacent to the existing auto-merge policy in
[`docs/auto-merge-policy.md`](./auto-merge-policy.md): auto-ready would happen
before review, while auto-merge happens after approval.

## Goal

Add an automated workflow that marks a draft PR as ready for review after its
required checks pass.

Expected flow:

1. A Codex-authored PR is opened as draft with the `auto-ready` label.
2. CI and required checks run normally.
3. The auto-ready workflow verifies the PR is still draft and all required
   checks are successful.
4. The workflow marks the PR ready for review.
5. Existing review-label automation handles the `ready_for_review` event.

## Requirements

- Only target PRs into `develop` or `main`.
- Only target PRs that explicitly opt in with the `auto-ready` label.
- Do not affect dependency bot PRs. Maintain a dependency-bot deny list instead
  of hardcoding one actor; the initial list should include at least
  `dependabot[bot]` and `renovate[bot]`.
- Do not mark a PR ready if any required check is pending, failing, cancelled, or
  skipped when it should be required.
- Do not wait on the auto-ready workflow itself.
- Treat GitHub branch-protection and active branch-ruleset required checks as
  the readiness gate. CodeRabbit is not a required auto-ready gate unless it is
  later added as a required status check by branch protection or a ruleset.
- The workflow that marks the PR ready must use the narrow permissions it needs:
  `pull-requests: write`, `checks: read`, and `statuses: read`.
- Prefer GitHub API / GraphQL over browser automation.
- The mark-ready action must verify the PR number, current head SHA, base branch,
  draft state, label state, and author deny-list state at execution time before
  mutating PR state.
- `workflow_run` events must resolve a single PR number before loading or
  mutating PR state. Ambiguous or missing PR associations must be treated as a
  no-op.
- Completion events must be treated as hints only. Never assume a
  `check_suite` or `workflow_run` completion still describes the PR's current
  head SHA.
- The workflow must only process same-repository PRs. Fork PRs are out of scope
  because completion-triggered workflows run with base repository permissions.
- The workflow must not check out the PR head ref. If repository contents are
  needed, use the triggering workflow SHA (`github.sha`) or avoid checkout
  entirely.
- The workflow must not read artifacts produced by fork-controlled workflow
  runs.

## Candidate implementation

Create `.github/workflows/auto-ready-pr.yml`.

Selected triggers:

- `pull_request` on `opened`, `synchronize`, `reopened`, `labeled`, and
  `edited`
- `workflow_run` completion for the required CI workflow
- required `schedule` fallback to catch required checks whose completion does
  not emit a monitored `workflow_run` event

The `labeled` trigger is required because adding `auto-ready` after CI has
already completed is a supported opt-in path. The `edited` trigger is required
to re-evaluate draft PRs after a base branch change into `develop` or `main`.
The scheduled fallback is required because required checks may be spread across
multiple workflows or external status contexts; the last required context to
finish may not be the single monitored CI workflow.

Do not use `check_suite` as the primary trigger. `workflow_run` can scope the
event to the CI workflow name, while `check_suite` is broader and increases the
chance of stale or irrelevant completion events. The implementation must still
re-query the live PR state because `workflow_run` can also be stale.

The job should:

1. Resolve the PR number safely:
   - For `pull_request` events, use `github.event.pull_request.number`.
   - For `workflow_run` events, first inspect
     `github.event.workflow_run.pull_requests`; if it contains exactly one item,
     use that PR number.
   - If `workflow_run.pull_requests` is empty or contains multiple items, query
     the GitHub REST API for open PRs by the workflow run's head branch, using
     the head owner and branch from the event payload. Use the result only if it
     resolves to exactly one same-repository PR.
   - If no unique same-repository PR can be resolved, exit without changing
     state.
2. Load the PR metadata from the GitHub API using the resolved PR number.
3. Exit if the PR is not from the same repository as the base branch.
4. Exit if the PR is not targeting `develop` or `main`.
5. Exit if the PR is not draft.
6. Exit if the PR does not have the `auto-ready` label.
7. Exit if the author is in the dependency-bot deny list.
8. For event triggers that include a source SHA, compare the live PR head SHA
   with the source SHA; exit if they differ. For `workflow_run`, use
   `github.event.workflow_run.head_sha` as the source SHA only after resolving a
   unique PR.
9. Resolve the required check contexts for the live base branch:
   - Query branch protection with
     `GET /repos/{owner}/{repo}/branches/{base}/protection` and read
     `required_status_checks.checks` / `required_status_checks.contexts`.
   - Query active branch rules with
     `GET /repos/{owner}/{repo}/rules/branches/{base}` and read any active
     `required_status_checks` rules that apply to the base branch.
   - Treat the readiness gate as the union of branch-protection and active
     ruleset required contexts.
   - If the workflow cannot read either required-check source, fail closed and
     exit without marking the PR ready.
10. Query check runs / status contexts for the live current head SHA.
11. Filter out the auto-ready workflow's own check run.
12. Confirm every required context is successful.
13. Mark the PR ready with either:
   - `gh pr ready <number> --repo <owner>/<repo>`
   - GraphQL `markPullRequestReadyForReview`

## Decisions

- CodeRabbit is not a required readiness gate for auto-ready. It remains part of
  review, not the pre-review readiness transition, unless repository branch
  protection later makes it a required check.
- Auto-ready is label-based opt-in via `auto-ready`; it must not apply to all
  draft PRs.
- Human-created draft PRs must opt in explicitly. Long-running WIP drafts should
  remain draft until a human marks them ready or adds the label intentionally.
- Trigger choice: use `pull_request`, `workflow_run` for the required CI
  workflow, and a required scheduled fallback. Do not leave the trigger choice
  to implementation-time preference.
- The monitored workflow name must be kept in sync with branch-protection and
  ruleset required checks. If required checks are renamed, split, or moved to an
  external status provider, update the workflow trigger and keep the scheduled
  fallback enabled so the automation does not silently stop re-evaluating PRs.

## Open questions

- Should auto-ready require no unresolved review threads, or is that irrelevant
  before first review?

## Implementation sequence

Recommended rollout:

1. Merge the documentation PR first so the proposal and trade-offs are preserved.
2. Merge the workflow implementation PR that adds `.github/workflows/auto-ready-pr.yml`.
3. Create the `auto-ready` label once in the repository.
4. Open a follow-up PR to update Codex / Claude PR creation guidance so Codex task
   PRs are opened as draft and labeled `auto-ready` by default.
5. Validate the full flow with the next real Codex-authored PR.

Do not update PR creation defaults before the workflow PR has merged. Until the
workflow exists on the target branch, opening all Codex PRs as draft would only
add manual ready-for-review work without the auto-ready path.

## Non-goals

- Do not change the auto-merge policy.
- Do not approve PRs automatically.
- Do not resolve review comments automatically.
- Do not close issues automatically.
- Do not change the scope police rules.
