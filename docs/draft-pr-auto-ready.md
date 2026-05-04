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
4. The auto-ready job verifies the PR is still eligible and all live required
   checks for the base branch are successful.
5. The workflow marks the PR ready for review.
6. Existing review-label automation handles the `ready_for_review` event.

## Rollout state

- The implementation workflow was merged in PR #472.
- The repository label `auto-ready` has been created.
- Codex / Claude PR creation guidance now says Codex task PRs should be opened
  as draft and labeled `auto-ready` by default.
- PR #488 adds the same-PR CI completion hook after rollout validation showed
  the standalone workflow could run before GitHub Actions checks were complete.
- The remaining validation step is to observe PR #488: draft -> checks pass ->
  auto-ready marks ready -> `needs-codex-review` flow takes over.

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
- Reuses the same live branch-protection required-check gate before marking the
  draft ready.

## Eligibility rules

The workflow only marks a PR ready when all of these are true at execution time:

- The PR targets `develop` or `main`.
- The PR is still a draft.
- The PR has the `auto-ready` label.
- The author is not `dependabot[bot]`.
- The PR head is in the same repository.
- The PR has a live head SHA.
- The current required checks for the base branch are successful.

Human-created draft PRs remain opt-in. Long-running WIP drafts should stay draft
until a human marks them ready or intentionally adds the `auto-ready` label.

## Readiness gate

The implementation queries GitHub's GraphQL `branchProtectionRule` for the live
base branch and reads:

- `requiredStatusChecks`
- `requiredStatusCheckContexts`

Required checks with an associated GitHub App are matched by `context + app id`.
Legacy required status contexts are matched by context name. The workflow also
records the latest visible result for each context/app key so a successful rerun
can replace an earlier failed result.

Allowed completed check-run conclusions:

- `success`
- `neutral`
- `skipped`

Commit statuses must have state `success`.

The auto-ready workflow's own check run is excluded from the gate. CodeRabbit is
not part of the auto-ready readiness gate unless repository branch protection
later makes it required.

If branch-protection or check/status lookups fail for a PR, the workflow skips
that PR for that run instead of falling back to an unsafe or overbroad check.

## PR creation default

Codex task PRs should now be opened as draft and labeled `auto-ready`:

```bash
gh pr create --draft --label auto-ready --title "[type] ..." --base develop --body-file /tmp/pr_body.md
```

Non-Codex tasks, human WIP drafts, and PRs that should not enter the review queue
automatically should not use the `auto-ready` label.

## Maintenance notes

- Keep the workflow regression tests in `.github/workflows/ci.test.mjs` aligned
  with both `.github/workflows/auto-ready-pr.yml` and the
  `auto-ready-after-ci` job in `.github/workflows/ci.yml`.
- If required checks are renamed, split, moved to another app, or added to
  branch protection, update the workflow and tests in the same PR.
- If Renovate is introduced in this repository, add `renovate[bot]` to the
  dependency-bot deny list before relying on auto-ready for Renovate PRs.
- If GraphQL branch-protection access changes, update the readiness-gate design
  rather than silently broadening the fallback behavior.

## Non-goals

- Do not change the auto-merge policy.
- Do not approve PRs automatically.
- Do not resolve review comments automatically.
- Do not close issues automatically.
- Do not change the scope police rules.
