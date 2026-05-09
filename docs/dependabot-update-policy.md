# Dependabot Update Policy

This repo uses Dependabot to keep Go and pnpm dependencies current without
turning routine tooling churn into review noise.

## Package Ecosystems

Dependabot uses `package-ecosystem: npm` for npm, yarn, and pnpm projects.
The dashboard and Twitch extension are pnpm projects because each directory has
its own `pnpm-lock.yaml`:

- `apps/dashboard/pnpm-lock.yaml`
- `apps/extension/pnpm-lock.yaml`

Do not change these entries to a non-existent `pnpm` ecosystem name.

## Scheduling

Version updates target `develop` and run on Monday morning in `Asia/Taipei`:

- `/services/api`: 09:00
- `/apps/dashboard`: 09:30
- `/apps/extension`: 10:00

Each ecosystem has `open-pull-requests-limit: 2` so Dependabot cannot flood the
queue when many packages release close together.

## Grouping

Dependency updates are grouped by review shape:

- Go modules in `/services/api` are grouped into `backend-go-deps`.
- Dashboard pnpm updates are split into production and development groups.
- Tachimint pnpm updates are split into production and development groups.

The production/development split keeps runtime dependency updates visible while
still batching development tooling changes such as TypeScript, ESLint, Vite, and
test tooling.

Routine pnpm version updates intentionally skip production patch releases. For
production dependencies, Dependabot opens routine version update PRs only for
minor and major releases; this avoids review noise for tiny runtime bumps.
Development dependencies still receive patch, minor, and major routine version
updates.

This `allow.update-types` boundary applies only to version updates. Dependabot
security update PRs remain enabled, including patch-level production updates
triggered by security alerts on the default branch. Production security update
PRs remain manual-review changes because dependency upgrades can introduce new
supply-chain, compatibility, or runtime risks even when they fix a known issue.

## Cooldown

High-frequency frontend tooling updates use Dependabot cooldown:

- patch updates: 21 days
- minor updates: 21 days
- major updates: 30 days

This applies to common development tooling packages such as `typescript`,
`eslint*`, `@types/*`, `@vitejs/*`, `vite`, and related dashboard test/build
tooling.

The goal is to let small toolchain releases settle before opening PRs, while
still allowing security alerts and runtime dependency updates to remain visible.

## Auto-Merge Boundary

The `dependabot-automerge.yml` workflow remains conservative:

- direct development security updates may auto-merge when checks pass
- selected direct development patch/minor updates may auto-merge
- updates explicitly labeled `safe-to-automerge` may auto-merge when checks pass;
  this is a manual override label, not a bot-only decision
- production dependency updates, including security patches, and excluded tooling
  packages do not auto-merge by default

If a Dependabot PR touches runtime behavior or a production dependency, review
it as a normal PR instead of relying on automation unless a maintainer has
explicitly applied `safe-to-automerge` after review.

## Dependency Review Gate

Dependabot opens routine version update PRs for the configured update levels and
security update PRs for alert-triggered fixes. Dependency Review is the
complementary PR gate for dependency changes opened by humans or automation: it
compares the dependency diff in the PR and blocks newly introduced high/critical
production dependency vulnerabilities before they enter `develop`.

The gate runs only when dependency manifests or lockfiles for the frontend
workspace change:

- `apps/dashboard/package.json`
- `apps/dashboard/pnpm-lock.yaml`
- `apps/extension/package.json`
- `apps/extension/pnpm-lock.yaml`
- `package.json`
- `pnpm-lock.yaml`
- `pnpm-workspace.yaml`

Policy:

- high/critical production dependency vulnerabilities are blocking
- development dependency findings are report-only in the first rollout
- license findings are not part of this gate
- broad `pnpm audit` remains out of scope because it scans the whole current
  tree instead of focusing on dependency changes in the PR

Development dependency findings can become blockers later only when the finding
affects packaged extension code or build-time execution in a way that creates a
real production risk.

## False Positives And Waivers

Do not add blanket waivers for Dependency Review. If a false positive or
temporarily accepted finding blocks a production dependency change, document the
exception in the PR body and create a follow-up issue before merging.

Waiver notes must include:

- Owner:
- Accepted on:
- Expires on:
- Affected package:
- Finding ID:
- Why this is accepted:
- Recheck trigger:

Waivers expire by default after 90 days. Security findings related to auth,
wallet signatures, token accounting, deploy credentials, or packaged extension
runtime behavior require human review even when the finding appears in a
development dependency.
