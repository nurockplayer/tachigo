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

- security updates may auto-merge when checks pass
- updates explicitly labeled `safe-to-automerge` may auto-merge when checks pass
- selected direct development patch/minor updates may auto-merge
- production dependency updates, major updates, and excluded tooling packages do
  not auto-merge by default

If a Dependabot PR touches runtime behavior or a production dependency, review it
as a normal PR instead of relying on automation.

## Dependency Review Gate

Dependabot opens routine version and security update PRs. Dependency Review is
the complementary PR gate for dependency changes opened by humans or automation:
it compares the dependency diff in the PR and blocks newly introduced
high/critical production dependency vulnerabilities before they enter
`develop`.

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
