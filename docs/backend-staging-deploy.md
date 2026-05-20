# Backend Staging Deploy Runbook

## Target

Backend staging target：containerized API service behind `https://api-staging.tachigo.io`.

| Boundary | Value |
| --- | --- |
| GitHub Environment | `staging-api` |
| Required approval | Backend / release owner approval before deploy |
| Image | tachigo API image built from the exact git SHA being deployed |
| Database | Staging PostgreSQL with `ATLAS_DATABASE_URL` configured in the environment |
| Smoke script | `infra/scripts/backend-staging-smoke.sh` |

Owner：backend release owner for the deployment window.

## Required Environment Values

| Variable / secret | Scope | Purpose |
| --- | --- | --- |
| `ATLAS_DATABASE_URL` | GitHub Environment secret / runtime secret | Atlas migration target used by the API entrypoint before startup. |
| `STAGING_API_BASE_URL` | GitHub Environment variable | Expected public base URL, e.g. `https://api-staging.tachigo.io`. |
| `STAGING_AUTH_BEARER_TOKEN` | GitHub Environment secret | Short-lived staging token for an authenticated smoke call. |
| `DEPLOYMENT_SHA` | Deploy job env | Git SHA being deployed. |
| `MIGRATION_STATUS` | Deploy job env | Migration step status captured as `applied`, `noop`, or `failed`. |

## Deploy Workflow

1. Build the API image from the commit SHA selected for staging.
2. Request approval on GitHub Environment `staging-api`.
3. Deploy the image to the staging container target with `ATLAS_DATABASE_URL` present.
4. Let the existing API Docker entrypoint run `atlas migrate apply --dir file://migrations --url "$ATLAS_DATABASE_URL"` before `tachigo` starts.
5. Capture the migration result as `MIGRATION_STATUS`.
6. Run smoke readback:

```bash
STAGING_API_BASE_URL=https://api-staging.tachigo.io \
STAGING_AUTH_BEARER_TOKEN=<short-lived-token> \
DEPLOYMENT_SHA=<git-sha> \
MIGRATION_STATUS=<applied|noop> \
infra/scripts/backend-staging-smoke.sh
```

## Smoke Readback

The smoke script records:

- Deployment SHA.
- Migration status.
- `GET /health` liveness result.
- `GET /readyz` readiness result.
- `GET /api/v1/users/me` authenticated API result.

The authenticated token must belong to a staging user that is safe to read. Do not use production user tokens for staging smoke.

## Rollback

Application image rollback:

- If migration fails before API startup, abort deployment and keep the previous staging image serving traffic.
- If smoke fails after startup, roll the staging container target back to the previous known-good image and rerun smoke.
- Record failed SHA, restored SHA, `MIGRATION_STATUS`, and smoke output in the release ticket.

Database rollback limits:

- Atlas migration rollback is not automatic in this repo.
- Additive migrations can normally stay in place while the previous image is restored.
- Destructive migrations require a separate maintainer-approved rollback plan and fresh staging backup before deployment.
- Do not include contract deploy, mainnet, or testnet deployment in this staging API workflow.
