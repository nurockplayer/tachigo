# Scheduled Dependency Inventory

**Source of truth**: #508
**Status**: report-only first rollout

## Decision

The first scheduled dependency inventory rollout uses OSV Scanner as a weekly
and manually runnable report. The workflow runs through `workflow_dispatch` and
on a weekly schedule, uploads separate SARIF reports, and sends results to
GitHub code scanning.

This is not a required check. Vulnerability findings should not block PRs in
this rollout because the repository already has narrower PR-time signals:

- Dependabot alerts and update PRs cover known dependency updates over time.
- Dependency Review blocks newly introduced high or critical production
  dependency vulnerabilities in frontend lockfile changes.
- `govulncheck` remains the lower-noise Go gate because it accounts for
  reachability instead of only package versions.

OSV Scanner still adds value as an inventory sweep because vulnerability data
changes even when repository files do not.

## Report surfaces

### Go module manifest

The Go report scans `services/api/go.mod`, which is the Go module manifest
supported by OSV Scanner for source scans.

Use this report to catch ecosystem advisories that may not yet be visible during
normal code review. If OSV reports a Go package vulnerability, first compare it
with `govulncheck` because reachable findings have higher priority than
package-version-only findings.

### pnpm lockfiles

The pnpm report scans the root lockfile plus the extension and dashboard
lockfiles:

- `pnpm-lock.yaml`
- `apps/extension/pnpm-lock.yaml`
- `apps/dashboard/pnpm-lock.yaml`

Use this report as an inventory view, not as a second PR gate. New vulnerable
runtime dependencies should still be handled by Dependency Review on the PR that
introduced the lockfile change.

### Container images

The container report builds and exports the backend, extension, and dashboard
images, then scans the image archives. This separates base image and OS package
findings from language lockfile findings.

Container findings are expected to be noisier than lockfile findings because dev
images include toolchains. Treat them as maintenance inventory until the project
has production-specific image targets and an agreed base image update cadence.

## Report owner

The production-readiness maintainer owns the weekly inventory run and is
responsible for routing findings to the relevant surface owner:

- Backend owner: Go module findings.
- Frontend owner: pnpm lockfile findings for extension and dashboard.
- Infra owner: container image findings and base image update policy.

## Triage SLA

- Review each weekly report within three business days.
- For critical or actively exploited findings, open or update a tracking issue
  by the next business day.
- For high severity findings with a clear upgrade path, target the next normal
  maintenance window.
- For medium, low, unknown, duplicate, or unreachable findings, record the
  decision during the weekly report review.

## Avoiding duplicate noise

Do not file a new issue only because OSV repeats an existing Dependabot alert,
Dependency Review finding, or `govulncheck` finding. Link the OSV report to the
existing issue or PR instead.

Create a separate issue only when OSV adds new information, such as a container
base image package that Dependabot does not cover, a lockfile finding without an
existing alert, or evidence that a previous waiver has expired.

## False positives and waivers

Accepted findings need an owner and an expiry date. Prefer fixing or upgrading
over waiving when a safe update exists.

Suggested waiver format:

```markdown
## OSV / <vulnerability id>

- Owner:
- Accepted on:
- Expires on:
- Surface: Go module manifest / pnpm lockfiles / Container images
- Affected package:
- Source report:
- Decision: false positive / accepted risk / duplicate / follow-up issue
- Why this is accepted:
- Recheck trigger:
```

## Promotion conditions for a blocking gate

Do not promote OSV inventory to a required check until all of the following are
true:

- The weekly report has run long enough to establish a clean or triaged baseline.
- Duplicate handling with Dependabot alerts and Dependency Review is stable.
- Container findings have a production image target or an explicit base image
  owner.
- Waivers have owners, expiry dates, and recheck triggers.
