# Workflow Scope Threshold Sample

Source issue: #898
Upstream audit: #884 / #877
Baseline input: [workflow-metrics-baseline.md](workflow-metrics-baseline.md)
Sample window: PRs updated on or after 2026-04-24 with current
`scope-violation` or `scope-exception` labels.

This note samples the recent PRs behind the #884 observation that
`scope-exception` usage is about as frequent as `scope-violation`. It is
intentionally docs-only: no Scope Police threshold, label, workflow, template, or
auto-ready behavior changes are made here.

## Method

Read-only commands:

```bash
gh pr list --repo nurockplayer/tachigo --state all --limit 120 \
  --search 'updated:>=2026-04-24 label:scope-violation,label:scope-exception' \
  --json number,title,state,author,labels,createdAt,updatedAt,closedAt,mergedAt,url,additions,deletions,changedFiles

gh pr list --repo nurockplayer/tachigo --state all --limit 100 \
  --search 'updated:>=2026-04-24 label:scope-exception' \
  --json number,title,state,author,labels,createdAt,updatedAt,closedAt,mergedAt,url,additions,deletions,changedFiles

for n in 862 844 827 825 730 729 556 554 476 457 434 372 369 342 \
  863 845 819 733 679 622 488 459 435 417 389; do
  gh pr view "$n" --repo nurockplayer/tachigo \
    --json number,title,state,additions,deletions,changedFiles,labels,files,mergedAt,closedAt,url
done
```

Important limitation: GitHub search exposes the labels currently attached to
PRs, not a full historical label-event stream. The sample therefore counts
current label occurrences. It found 14 `scope-violation` occurrences plus 15
`scope-exception` occurrences, with 4 overlapping PRs, for 25 unique PRs.

## Summary

| Classification | Unique PRs | Pattern |
| --- | ---: | --- |
| Legit-large / mechanical | 13 | Monorepo moves, generated Swagger, migration baseline, framework dependency rollout, large deletion, docs portal import, or one-shot governance extraction. |
| Splittable / maybe splittable | 8 | Mixed backend/frontend/schema work, broad frontend redesigns, config plus dependency cleanup, or dirty retry attempts. |
| False positive / artifact-heavy friction | 4 | Long docs, planning artifacts, source-small dependency churn, or binary-asset-heavy PRs where hard line limits caught reviewability but not product risk. |

The result does not support raising the global `1000` hard or `600` soft line
thresholds. Those limits caught several genuinely splittable PRs. The main
calibration problem is that `scope-exception` is too undifferentiated: it covers
legitimate generated/mechanical work, broad feature bundles, and docs-only
planning artifacts with the same bypass semantics.

## Sample

| PR | Labels | Size | State | Classification | Notes |
| --- | --- | ---: | --- | --- | --- |
| #342 | `scope-violation` | +298/-40, 8 files | closed | Splittable | Backend and dashboard changes in one PR; the label correctly caught mixed product surfaces. |
| #369 | `scope-violation`, `scope-exception` | +551/-814, 10 files | merged | Legit-large / mechanical | One governance extraction touching agent rules, hooks, CI, and metadata. Exception was plausible, but it should carry an explicit governance-extraction reason. |
| #372 | `scope-violation`, `scope-exception` | +1691/-13, 5 files | closed | Legit-large / generated | Swagger-heavy backend rename slice. Generated-file handling needs a clearer category than generic exception. |
| #389 | `scope-exception` | +1714/-5, 5 files | merged | Legit-large / generated | Swagger regeneration dominated the diff; exception was plausible with generated-doc evidence. |
| #417 | `scope-exception` | +3262/-117, 151 files | merged | Legit-large / mechanical | Frontend monorepo move. This is the canonical mechanical-move exception class. |
| #434 | `scope-violation` | +1011/-14, 49 files | closed | Legit-large / mechanical | Extension demo move; violation is expected without an explicit mechanical-move exception. |
| #435 | `scope-exception` | +65/-36, 156 files | merged | Legit-large / mechanical | Backend service move; file-count exception was reasonable, but should be documented as path-only movement. |
| #457 | `scope-violation` | +1346/-531, 19 files | closed | Splittable | Dashboard Refine migration included pages, providers, tests, docs, and lockfile. Better split provider foundation from page migration. |
| #459 | `scope-exception` | +1101/-23, 8 files | merged | Legit-large / framework dependency | Dashboard provider foundation plus lockfiles. Exception was plausible, but needs dependency/artifact evidence. |
| #476 | `scope-violation`, `scope-exception` | +2061/-21, 10 files | closed | Legit-large / migration | Atlas migration/tooling baseline. A migration-specific exception category would be clearer than general bypass. |
| #488 | `scope-exception` | +786/-365, 9 files | merged | Legit-large / governance rollout | Auto-ready workflow rollout mixed CI, policy docs, agent docs, and plans. It was below the hard line limit but broad enough to justify stricter exception rationale. |
| #554 | `scope-violation` | +59/-1583, 6 files | closed | Maybe splittable | Backend config plus dependency cleanup. The line count was dependency-heavy; config change and module cleanup could have split. |
| #556 | `scope-violation` | +59/-1583, 6 files | closed | Maybe splittable | Same shape as #554. This looks like duplicate/retry friction rather than a threshold problem. |
| #622 | `scope-exception` | +7/-4581, 49 files | merged | Legit-large / mechanical | Legacy demo removal. Large deletion is a valid exception class when the PR is deletion-only or near deletion-only. |
| #679 | `scope-exception` | +15118/-2141, 23 files | merged | Legit-large / docs import | Dev Portal guide import is very large but docs-oriented. It should require an explicit docs-import reason and probably stay outside routine auto-ready. |
| #729 | `scope-violation` | +881/-174, 7 files | closed | Maybe splittable | Dev Portal CSS, cache, docker compose, spec, and lockfile together. Split visual CSS from environment/lockfile updates. |
| #730 | `scope-violation` | +879/-173, 4 files | closed | Maybe splittable | Similar CSS/system slice. Below hard limit but above soft warning; likely reviewability problem rather than false positive. |
| #733 | `scope-exception` | +1047/-0, 3 files | merged | False positive / docs-only | Planning/spec docs exceeded the hard line limit but did not alter product runtime. Needs docs-only calibration, not a global threshold raise. |
| #819 | `scope-exception` | +1157/-235, 5 files | merged | False positive / binary asset-heavy | Frontend redesign line count was inflated by a binary asset plus visual docs. The feature was coherent, but exception evidence should call out asset weight. |
| #825 | `scope-violation` | +1097/-2, 4 files | closed | False positive / metadata-only | DB schema review docs plus workflow/test metadata. The hard line limit caught long analysis, not runtime risk. |
| #827 | `scope-violation` | +1079/-0, 2 files | closed | False positive / docs-only | Discussion/planning docs only. Better handled by docs-specific exception guidance or issue comment when no repo artifact is required. |
| #844 | `scope-violation` | +2823/-31, 14 files | closed | Splittable | Backend, frontend, schema, migration, tests, and docs in one PR. This is exactly what hard gates should block. |
| #845 | `scope-exception` | +980/-10, 11 files | merged | Legit-large / migration | Backend-only raffle tiers API/model/migration with Atlas normalization. Under hard limit but still needs migration exception evidence. |
| #862 | `scope-violation`, `scope-exception` | +0/-1704, 12 files | closed | Legit-large / deletion | Extension JWT backend removal was deletion-heavy and backend-only, but generated Swagger/test fallout should be listed in the exception reason. |
| #863 | `scope-exception` | +0/-1739, 15 files | closed | Legit-large / deletion | Same deletion family as #862 with shared-types/contract checks included. Valid exception shape, but still needs explicit replacement/closure reason when not merged. |

## Recommendation

Keep the current global thresholds for normal product PRs:

- `600` diff lines should remain a warning.
- `1000` diff lines should remain a hard stop.
- `35` changed files should remain a hard stop.

Tighten `scope-exception` instead of raising the limits:

1. Require a short exception reason in the PR body or sticky comment category
   before the label is considered policy-compliant.
2. Recognize a small set of exception categories: `mechanical-move`,
   `generated-output`, `large-deletion`, `migration-baseline`, `framework-deps`,
   `binary-asset`, `docs-import`, and `governance-policy`.
3. Keep mixed product surfaces blocked unless the exception reason says why the
   surfaces are inseparable and names the follow-up split plan.
4. Treat docs-only or planning-only PRs above the hard line limit as a separate
   calibration problem. They may deserve docs-specific guidance, but not a
   broader product-code threshold raise.
5. Keep `auto-ready` away from exception PRs unless a human reviewer has already
   confirmed the exception reason and the risk class is not `R4`.

## Follow-Up Cuts

This PR does not implement these changes. Possible follow-ups:

| Follow-up | Scope | Validation |
| --- | --- | --- |
| Exception reason taxonomy | Update `docs/pr-scope-policy.md` and PR template wording only. | `git diff --check`, PR metadata regression if template parsing changes. |
| Scope Police exception reason check | Add a lightweight check that warns or fails when `scope-exception` has no reason. | Workflow regression tests in `.github/workflows/ci.test.ts`. |
| Docs-only large-analysis guidance | Define when long docs belong in repo docs vs issue/discussion comments. | Docs diff check only. |
| Generated-output accounting | Teach sticky comments to identify generated Swagger / lockfile / migration output separately from source edits. | Workflow regression tests plus fixture PR metadata. |
