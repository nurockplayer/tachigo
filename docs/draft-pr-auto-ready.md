# Draft PR auto-ready

**Status**: implemented; rollout validation in progress
**Created**: 2026-05-02
**Issue**: https://github.com/nurockplayer/tachigo/issues/470
**Implementation PRs**: https://github.com/nurockplayer/tachigo/pull/472, https://github.com/nurockplayer/tachigo/pull/488

## Context

Codex task PRs are often opened while GitHub Actions and external review checks
are still running. Opening them as draft PRs makes the state clearer: the PR is
published for CI, but not yet asking for human review. Once the required checks
are green, the PR can become ready for review automatically.

This is adjacent to the existing auto-merge policy in
[`docs/auto-merge-policy.md`](./auto-merge-policy.md): auto-ready happens before
review, while auto-merge happens after approval.

## Current flow

1. A Codex-authored PR is opened as draft with the `auto-ready` label.
2. CI and required checks run normally.
3. `.github/workflows/ci.yml` runs `auto-ready-after-ci` after the protected CI
   gate jobs finish.
4. The auto-ready job verifies the PR is still eligible and all required checks
   in the maintained snapshot for the base branch are successful.
5. The workflow marks the PR ready for review.
6. The same workflow adds `needs-codex-review` and removes `changes-requested`
   because events emitted by `GITHUB_TOKEN` do not trigger the separate
   `ready_for_review` label workflow.

## Rollout state

- The implementation workflow was merged in PR #472.
- The repository label `auto-ready` has been created.
- Codex / Claude PR creation guidance now says Codex task PRs should be opened
  as draft and labeled `auto-ready` by default.
- PR #488 adds the same-PR CI completion hook after rollout validation showed
  the standalone workflow could run before GitHub Actions checks were complete.
- PR #488 also switches the readiness gate to a tested required-check snapshot
  after rollout validation showed the Actions `GITHUB_TOKEN` cannot read
  GraphQL `branchProtectionRule`.
- PR #488 validation showed `markPullRequestReadyForReview` can transition the
  PR from draft to ready after the `contents: write` fix.
- PR #488 also adds direct review-label handling after rollout validation showed
  a `ready_for_review` event emitted by `GITHUB_TOKEN` does not wake the
  separate review-label workflow.

## Implemented workflow

Workflow file:

- `.github/workflows/auto-ready-pr.yml`
- `.github/workflows/ci.yml`, job `auto-ready-after-ci`

Standalone workflow triggers:

- `pull_request` on `opened`, `synchronize`, `reopened`, and `labeled`
- `check_suite` on `completed`
- `schedule` every 10 minutes
- `workflow_dispatch`

The `labeled` trigger supports adding `auto-ready` after CI has already
completed. The scheduled fallback catches required checks whose completion does
not emit a useful event for this workflow after the workflow exists on the
default branch.

CI completion hook:

- Runs on `pull_request` after `scope-gate`, `backend-ci`, `frontend`,
  `dashboard`, and `contracts` finish.
- Allows required CI jobs with result `success` or `skipped`.
- Refreshes live PR state before mutating the PR.
- Reuses the same required-check snapshot gate before marking the draft ready.
- Uses `contents: write`, `pull-requests: write`, and `issues: write` only for
  the auto-ready mutation/label path; the main CI workflow's top-level
  permissions remain read-only.

## Eligibility rules

The workflow only marks a PR ready when all of these are true at execution time:

- The PR targets `develop` or `main`.
- The PR is still a draft.
- The PR has the `auto-ready` label.
- The author is not `dependabot[bot]`.
- The PR head is in the same repository.
- The PR has a live head SHA.
- The required checks in the base branch snapshot are successful.

Human-created draft PRs remain opt-in. Long-running WIP drafts should stay draft
until a human marks them ready or intentionally adds the `auto-ready` label.

## Readiness gate

Rollout validation showed GitHub Actions `GITHUB_TOKEN` cannot read GraphQL
`branchProtectionRule` for this repository; the API returns `Resource not
accessible by integration`. To avoid introducing a higher-privilege secret, the
workflow keeps a small required-check snapshot in:

- `.github/workflows/auto-ready-pr.yml`
- `.github/workflows/ci.yml`, job `auto-ready-after-ci`

Snapshot as of 2026-05-04:

| Branch | Required check contexts |
|---|---|
| `develop` | `Scope gate` |
| `develop` | `Backend CI (gate)` |
| `develop` | `Frontend build` |
| `develop` | `Dashboard build` |
| `develop` | `Contracts build` |
| `main` | `Scope police` |

Required checks with an associated GitHub App are matched by `context + app id`.
The current GitHub Actions app id in the snapshot is `15368`. Legacy required
status contexts can still be matched by context name if they are added to the
snapshot with a `null` app id. The workflow also records the latest visible
result for each context/app key so a successful rerun can replace an earlier
failed result.

Allowed completed check-run conclusions:

- `success`
- `neutral`
- `skipped`

Commit statuses must have state `success`.

The auto-ready workflow's own check run is excluded from the gate. CodeRabbit is
not part of the auto-ready readiness gate unless repository branch protection
later makes it required and the snapshot is updated in the same PR.

If check/status lookups fail for a PR, the workflow skips that PR for that run
instead of falling back to an unsafe or overbroad check. If a base branch has no
snapshot entry, the workflow also skips rather than using visible checks as a
fallback.

Rollout validation also showed `markPullRequestReadyForReview` fails for this
repository when the Actions token only has `contents: read`, even with
`pull-requests: write`. The auto-ready paths therefore grant `contents: write`
only where the ready-for-review mutation can run.

The ready-for-review mutation is executed with `GITHUB_TOKEN`, so the resulting
`ready_for_review` event does not trigger `.github/workflows/codex-review-flag.yml`.
The auto-ready paths therefore also use `issues: write` to add
`needs-codex-review` and remove stale `changes-requested` in the same guarded
mutation path.

## PR creation default

Codex task PRs should now be opened as draft and labeled `auto-ready`:

```bash
make pr-open TITLE="[type] ..." BODY_FILE=/tmp/pr_body.md AUTO_READY=1
```

If opening a PR directly with `gh pr create`, use `--draft --label auto-ready`.

Non-Codex tasks, human WIP drafts, and PRs that should not enter the review queue
automatically should not use the `auto-ready` label.

## Maintenance notes

- Keep the workflow regression tests in `.github/workflows/ci.test.mjs` aligned
  with both `.github/workflows/auto-ready-pr.yml` and the
  `auto-ready-after-ci` job in `.github/workflows/ci.yml`.
- If required checks are renamed, split, moved to another app, or added to
  branch protection, update the workflow snapshot, this document, and tests in
  the same PR.
- If Renovate is introduced in this repository, add `renovate[bot]` to the
  dependency-bot deny list before relying on auto-ready for Renovate PRs.
- If GraphQL branch-protection access becomes available to the Actions token,
  the readiness-gate design can move back to live branch protection lookup, but
  only with regression tests covering the new behavior.

## Non-goals

- Do not change the auto-merge policy.
- Do not approve PRs automatically.
- Do not resolve review comments automatically.
- Do not close issues automatically.
- Do not change the scope police rules.
