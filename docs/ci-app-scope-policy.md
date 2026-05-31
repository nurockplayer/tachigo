# App-scoped CI policy

Tachigo is a monorepo. CI routing should follow the app boundary instead of a coarse frontend/backend split.

This document describes the intended policy boundary for `.github/workflows/ci.yml`. It is not a request to rename existing required checks.

## App scopes

| Scope | Paths | Expected validation |
| --- | --- | --- |
| Extension | `apps/extension/**`, legacy `tachimint/**` | Extension build, lint, test, and i18n checks. |
| Dashboard | `apps/dashboard/**`, legacy `dashboard/**` | Dashboard build, lint, and test checks. |
| Backend | `services/api/**`, legacy `backend/**` | Backend build, Go checks, migration checks, and integration checks. |
| Contracts | `contracts/**` | Contract build, test, format, and report checks. |
| Shared frontend workspace | `package.json`, `pnpm-lock.yaml`, `pnpm-workspace.yaml` | Extension and Dashboard checks. |
| API contract surfaces | API client, shared API types, backend router, OpenAPI docs | API contract drift checks. |
| CI routing files | CI workflow and workflow regression files | Full CI, because routing changes affect the router itself. |
| Docs and templates | `docs/**`, `plans/**`, root Markdown, PR/issue templates | No heavy product CI for metadata-only changes. |

## Routing rules

- Treat Extension and Dashboard as separate app scopes.
- Do not force Extension checks for Dashboard-only changes.
- Do not force Dashboard checks for Extension-only changes.
- Run both Extension and Dashboard checks for shared frontend workspace changes, because root package metadata can affect either app.
- Run API contract drift checks for backend router/OpenAPI changes, app API client/service changes, shared API packages, and root workspace package metadata.
- Run full CI for CI workflow or workflow regression changes, because the router itself is part of the change.
- Keep docs/template/metadata-only PRs out of heavy product CI unless their paths also affect workflow behavior.
- Keep the existing required-check names stable unless branch protection is updated in the same change.

## Required check names

The current `frontend` CI job name maps to the Extension app scope. New documentation should use `Extension` for that app while preserving the existing check name.

This naming distinction matters because branch protection and auto-ready logic may depend on check names. A policy or documentation PR may clarify that `frontend` means Extension, but it must not rename the required check without a separate branch-protection-aware workflow change.

## Reviewer checklist

- If a PR touches only `apps/extension/**`, expect Extension/frontend checks without Dashboard checks.
- If a PR touches only `apps/dashboard/**`, expect Dashboard checks without Extension/frontend checks.
- If a PR touches root frontend workspace files, expect both Extension and Dashboard checks.
- If a PR touches `.github/workflows/ci.yml` or `.github/workflows/ci.test.ts`, expect full CI and workflow regression coverage.
- If a PR changes only docs/templates/metadata, heavy product CI may be skipped by design.
