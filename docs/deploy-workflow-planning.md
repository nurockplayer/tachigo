---
title: Deploy workflow planning
status: active
owner: engineering
last_reviewed: 2026-06-01
source_of_truth: true
code_areas:
  - docs
  - .github/workflows
  - services/api
  - apps/dashboard
  - apps/extension
related_repos:
  - tachigo
related_issues:
  - 212
  - 1020
---

# Deploy workflow planning

This document closes the planning part of #212. It records the deploy workflow boundaries that should guide the implementation issue #1020. It does not create or modify deploy automation.

## Current runbooks

| Surface | Existing source | Current status |
|---|---|---|
| Backend staging API | [`backend-staging-deploy.md`](backend-staging-deploy.md) | Has staging API target, GitHub Environment name, approval expectation, Atlas migration step, smoke script, and rollback limits. |
| Dashboard | [`dashboard-deployment.md`](dashboard-deployment.md) | Has Cloudflare Pages staging / production target, artifact contract, production API URL readback, smoke checklist, and rollback readback. |
| Dev Portal | [`dev-portal/deployment.md`](dev-portal/deployment.md) | Has Cloudflare Pages manual setup, `develop` production branch model, preview readback, and rollback checklist for docs hosting only. |
| Extension package | [`apps/extension/README.md`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/README.md) | Has Chrome Web Store / sideload package checklist and production package readback. |
| API schema changes | [`services/api/README.md`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/README.md), [`atlas-schema-reconciliation.md`](atlas-schema-reconciliation.md) | Atlas owns schema migrations; API startup must not regain schema DDL ownership. |

## Decisions

### Deploy entrypoints

The first repo-owned deploy workflow should be manual-first:

| Environment | Initial trigger | Gate | Notes |
|---|---|---|---|
| Backend staging | `workflow_dispatch` with a commit SHA | GitHub Environment `staging-api` approval | Use the backend staging runbook and smoke script. |
| Dashboard staging | `workflow_dispatch` with a commit SHA | Dashboard release owner approval | Build `apps/dashboard/dist` and deploy to the staging Pages project. |
| Production | manual promotion after staging readback | GitHub Environment production approval | Do not connect this to auto-ready or auto-merge. |

Release PR automation remains separate. A `develop -> main` release PR can identify the candidate commit, but deploy must still require explicit environment approval and smoke readback.

### GitHub Environments

Deploy implementation should use GitHub Environments for approval and secret scoping:

| Environment | Owner | Required approval | Secret / variable scope |
|---|---|---|---|
| `staging-api` | backend release owner | yes | staging database URL, staging smoke token, staging API base URL |
| `staging-dashboard` | dashboard release owner | yes | Cloudflare Pages deployment credentials and staging project name |
| `production-api` | backend release owner | yes | production database URL, production API target, production smoke token |
| `production-dashboard` | dashboard release owner | yes | production Pages credentials and production project name |

Secrets must be configured by maintainers in GitHub / Cloudflare. AI agents may document expected names, but must not invent, rotate, or print secret values.

### Migration strategy

Backend deploy must keep the existing Atlas ownership model:

1. Build the API image from the exact commit SHA selected for deployment.
2. Run `atlas migrate apply --dir file://migrations --url "$ATLAS_DATABASE_URL"` before starting the API process, as described by the backend staging runbook and Docker entrypoint.
3. Capture migration status as `applied`, `noop`, or `failed`.
4. If migration fails before startup, abort the deploy and keep the previous image serving traffic.
5. Destructive migrations require a separate maintainer-approved rollback plan and backup before deployment.

The deploy workflow must not hide schema changes in server startup or Cloudflare configuration. Schema changes belong in `services/api/migrations/`.

### Smoke and rollback

Every deploy path needs a readback before promotion:

| Surface | Minimum smoke readback | Rollback evidence |
|---|---|---|
| Backend API | deployment SHA, migration status, `GET /health`, `GET /readyz`, authenticated `GET /api/v1/users/me` | failed SHA, restored SHA, migration status, smoke output |
| Dashboard | production package readback, login, authenticated routes, streamers, raffles, transactions, settings, network origin | rolled-back-from SHA, restored SHA, reason, smoke result |
| Dev Portal | homepage, `llms.txt`, `manifest.json`, search, source index | deployment id / commit SHA, URL readback, checklist result |
| Extension | production package readback, Chrome load-unpacked smoke, Twitch side panel smoke, API origin readback | package SHA, submission state, rollback / resubmission note |

Smoke failures should stop promotion. Production rollback should prefer the previous known-good artifact or image and then rerun the same smoke checklist.

## Implementation issue

Follow-up implementation is tracked in #1020.

#1020 should be an R4 workflow / release PR because it will modify `.github/workflows/**` and deployment behavior. It should not use `auto-ready`, and it should include workflow regression coverage for deploy workflow routing.

## Explicit non-goals

- Do not add `.github/workflows/deploy.yml` in this planning PR.
- Do not configure GitHub secrets, GitHub Environments, Cloudflare projects, DNS, or database credentials in this PR.
- Do not combine contract deploy / verify, testnet, or mainnet with the backend / dashboard deploy workflow.
- Do not make release PR creation automatically deploy production.
- Do not change Atlas migrations or API startup behavior here.

## Acceptance readback for #212

- Staging / production deploy entrypoints are decided.
- GitHub Environments and approval gates are documented.
- Secrets, migration, rollback, and smoke readback responsibilities are documented without exposing secret values.
- Follow-up implementation issue #1020 exists for the actual deploy workflow.
