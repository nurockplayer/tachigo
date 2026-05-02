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

1. A Codex-authored PR is opened as draft.
2. CI and required checks run normally.
3. The auto-ready workflow verifies the PR is still draft and all required
   checks are successful.
4. The workflow marks the PR ready for review.
5. Existing review-label automation handles the `ready_for_review` event.

## Requirements

- Only target PRs into `develop` or `main`.
- Do not affect Dependabot PRs.
- Do not mark a PR ready if any required check is pending, failing, cancelled, or
  skipped when it should be required.
- Do not wait on the auto-ready workflow itself.
- Treat external checks such as CodeRabbit explicitly if they are expected to
  gate review readiness.
- Use `pull-requests: write` permission only in the workflow that marks the PR
  ready.
- Prefer GitHub API / GraphQL over browser automation.

## Candidate implementation

Create `.github/workflows/auto-ready-pr.yml`.

Possible triggers:

- `pull_request` on `opened`, `synchronize`, and `reopened`
- `check_suite` or `workflow_run` completion
- optional scheduled fallback to catch missed events

The job should:

1. Load the PR metadata.
2. Exit if the PR is not draft.
3. Exit if the author is `dependabot[bot]`.
4. Query check runs / status contexts for the current head SHA.
5. Filter out the auto-ready workflow's own check run.
6. Confirm every required context is successful.
7. Mark the PR ready with either:
   - `gh pr ready <number> --repo <owner>/<repo>`
   - GraphQL `markPullRequestReadyForReview`

## Open questions

- Should CodeRabbit be a required readiness gate, or should auto-ready only wait
  for GitHub Actions?
- Should the workflow apply to all draft PRs, or only PRs with a specific label
  such as `auto-ready`?
- Should human-created draft PRs opt in explicitly, so drafts can still be used
  for long-running work-in-progress?
- Should auto-ready require no unresolved review threads, or is that irrelevant
  before first review?

## Non-goals

- Do not change the auto-merge policy.
- Do not approve PRs automatically.
- Do not resolve review comments automatically.
- Do not close issues automatically.
- Do not change the scope police rules.
