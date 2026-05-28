# App-scoped CI policy

Tachigo is a monorepo. CI routing should follow the app boundary instead of a coarse frontend/backend split.

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

## Rules

- Treat Extension and Dashboard as separate app scopes.
- Do not force extension checks for dashboard-only changes.
- Do not force dashboard checks for extension-only changes.
- Run both Extension and Dashboard checks for shared frontend workspace changes.
- Keep the existing required-check names stable unless branch protection is updated in the same change.

The current `frontend` CI job name maps to the Extension app scope. New documentation should use `Extension` for that app while preserving the existing check name.
