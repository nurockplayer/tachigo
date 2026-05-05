# Security Scanner Evaluation

**Status**: accepted for Milestone 4 planning
**Source of truth**: #210
**Last reviewed**: 2026-05-05

## Goal

Pick the first production-readiness security scanners for tachigo without
turning CI into a noisy wall of non-actionable warnings.

This document decides which scanners should be introduced first, whether each
scanner should block merges or only report findings, how often each scanner
should run, and how false positives should be handled. It intentionally does not
implement scanners or change branch protection.

## Repo Context

Current repo surfaces:

- Backend: Go service in `services/api`.
- Contracts: Foundry Solidity project in `contracts`.
- Frontend: pnpm React apps in `apps/extension` and `apps/dashboard`.
- Dependency automation: Dependabot already targets Go and both pnpm apps.
- CI: `.github/workflows/ci.yml` already runs path-aware backend, frontend,
  dashboard, contracts, and workflow regression jobs.

## Recommendation Summary

| Surface | Tool | First rollout | Later required check | Frequency |
|---|---|---|---|---|
| Backend Go static analysis | `staticcheck ./...` | Blocking once baseline is clean | Yes, after first implementation PR proves clean baseline | PR touching `services/api`, push to `develop`, release PR |
| Backend Go vulnerabilities | `govulncheck ./...` | Blocking for reachable known vulnerabilities | Yes, after false-positive process exists | PR touching `services/api`, weekly schedule, release PR |
| Contracts static analysis | Slither via `crytic/slither-action` | Report-only SARIF / artifact | Not in first rollout | PR touching `contracts`, weekly schedule, release PR |
| Contracts gas regression | `forge snapshot --check` | Report-only until `.gas-snapshot` is intentionally committed | Conditional; only after gas budget ownership is clear | PR touching `contracts`, release PR |
| Frontend dependency changes | GitHub Dependency Review Action | Blocking for new high/critical production dependency vulnerabilities | Yes, for lockfile changes | PR touching `apps/*/pnpm-lock.yaml` or root lockfiles |
| Existing dependency inventory | Dependabot alerts plus optional OSV Scanner | Report-only scheduled summary | No in first rollout | Weekly schedule |
| Build-time security warnings | Vite / pnpm build logs | Document policy first | No in first rollout | Same as frontend/dashboard build |

## Backend

### `staticcheck`

Use `staticcheck ./...` in `services/api`.

Decision:

- Roll out as blocking after the implementation PR confirms the current baseline
  is clean.
- Pin the Staticcheck version in CI rather than using an unbounded `@latest` in
  every run.
- Do not merge Staticcheck into `go vet`; keep it as a separate CI step so
  failures are attributable.

Rationale:

- Staticcheck is a mature Go linter that runs over normal Go package patterns.
- Findings are usually code-quality or correctness issues, not dynamic advisory
  data, so it is suitable as a stable PR gate once the repo baseline is clean.

### `govulncheck`

Use `govulncheck ./...` in `services/api`.

Decision:

- Roll out as blocking for reachable known vulnerabilities after the
  false-positive and waiver process below exists.
- Also run on a weekly schedule because vulnerability databases change even when
  code does not.
- Treat release PR failures as blockers.

Rationale:

- `govulncheck` uses the Go vulnerability database and analyzes whether project
  code transitively calls vulnerable functions, which should be lower noise than
  package-version matching alone.
- Because the vulnerability database changes over time, a scheduled run is
  necessary; relying only on PR path changes misses newly published advisories.

## Contracts

### Slither

Use the official Crytic Slither GitHub Action for Solidity static analysis.

Decision:

- First rollout should be report-only.
- Upload SARIF when available so findings live in GitHub code scanning instead
  of disappearing into CI logs.
- Do not make Slither required until the team has triaged the initial baseline
  and documented accepted findings.

Rationale:

- Slither is a strong Solidity static analyzer, but smart-contract findings need
  human triage. Blocking immediately can create review churn and train the team
  to ignore scanner output.
- The project currently has a small token contract surface; report-only gives
  useful signal while keeping the first CI hardening pass low risk.

### Gas Snapshot

Use Foundry `forge snapshot` and eventually `forge snapshot --check`.

Decision:

- First implementation should generate and review a baseline `.gas-snapshot`.
- Keep gas snapshot report-only until the team agrees that gas deltas are
  production-relevant for the contract surface.
- If enabled as blocking later, use a tolerance and require PR body notes for
  intentional gas changes.

Rationale:

- Gas snapshots are not security scanners, but #210 explicitly asks to evaluate
  gas snapshot as a contracts quality signal.
- `forge snapshot --check` exits non-zero when the snapshot differs, which makes
  it usable as a gate only after baseline ownership is clear.

## Frontend And Dashboard

### Dependency Review Action

Use GitHub Dependency Review Action for PRs that change frontend or root
lockfiles.

Decision:

- Make it blocking for newly introduced high/critical vulnerabilities in
  production dependencies.
- Keep development dependency findings report-only unless they affect build-time
  execution or packaged extension code.
- Run it in a dedicated workflow or job with minimal permissions.

Rationale:

- Dependabot already handles ongoing update PRs. Dependency Review complements
  it by preventing new vulnerable dependency versions from entering through
  normal PRs.
- This is better suited for PR gating than a broad `pnpm audit` check because it
  focuses on dependency changes and GitHub can enforce severity thresholds.

### OSV Scanner

Use OSV Scanner only as a scheduled report in the first phase.

Decision:

- Do not make OSV Scanner a required PR check in the first rollout.
- Consider it for weekly inventory scans over lockfiles and, later, container
  images.
- If adopted, keep reports separate from Dependency Review to avoid duplicate
  vulnerability noise.

Rationale:

- OSV Scanner can scan language artifacts and container images, but adding it as
  an immediate blocking PR gate would overlap with Dependabot and Dependency
  Review.

### Build-Time Security Warnings

Do not create a generic "fail on any warning" policy yet.

Decision:

- Keep frontend/dashboard builds as they are.
- Define warning categories before making them blocking:
  - dependency vulnerability warning
  - unsafe browser API usage
  - CSP / extension manifest warning
  - bundler deprecation warning
- Only security-relevant warnings should become blockers.

Rationale:

- A broad fail-on-warning policy often blocks routine toolchain deprecations and
  creates low-signal CI failures.

## False Positive And Waiver Policy

Every accepted scanner finding needs an explicit owner and expiration.

Suggested waiver format:

```markdown
## <scanner> / <finding id>

- Owner:
- Accepted on:
- Expires on:
- Affected surface:
- Why this is accepted:
- Recheck trigger:
```

Rules:

- Waivers live in version control, preferably `docs/security-scanner-waivers.md`
  once the first waiver is needed.
- Waivers must expire. A default 90-day expiration is appropriate for dependency
  and static-analysis findings.
- A waiver cannot be created by the same PR that introduced the finding unless
  the PR body explains why the risk is accepted.
- Security findings on auth, wallet signatures, token accounting, migrations, or
  deploy credentials require human review even when the scanner job is
  report-only.

## Rollout Order

1. Backend scanners: add `staticcheck` and `govulncheck` because the Go service
   is the largest production risk surface and already has native CI.
2. Frontend dependency gate: add GitHub Dependency Review for lockfile changes
   to prevent new high/critical production dependency vulnerabilities.
3. Contracts scanner report: add Slither as report-only plus SARIF/artifact
   output.
4. Contracts gas snapshot policy: decide whether `.gas-snapshot` belongs in
   version control and whether gas drift should block PRs.
5. Scheduled inventory scan: decide whether OSV Scanner adds signal beyond
   Dependabot alerts and Dependency Review.

## Follow-Up Implementation Issues

These issues split implementation into independently reviewable PRs:

| Issue | Purpose | Suggested first gate |
|---|---|---|
| #504 | Backend `staticcheck` and `govulncheck` CI | Blocking after clean baseline |
| #505 | Frontend/dashboard Dependency Review gate | Blocking for new high/critical production dependency vulnerabilities |
| #506 | Contracts Slither report workflow | Report-only SARIF/artifact |
| #507 | Contracts gas snapshot policy | Report-only until baseline ownership is agreed |
| #508 | Scheduled OSV/dependency inventory report | Report-only |

## References

- Staticcheck getting started: https://staticcheck.dev/docs/getting-started/
- Go vulnerability management and `govulncheck`: https://go.dev/doc/security/vuln/
- Crytic Slither GitHub Action: https://github.com/marketplace/actions/slither-action
- Foundry `forge snapshot`: https://getfoundry.sh/forge/reference/snapshot/
- GitHub Dependency Review: https://docs.github.com/en/code-security/concepts/supply-chain-security/about-dependency-review
- GitHub SARIF upload: https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/integrate-with-existing-tools/uploading-a-sarif-file-to-github
- OSV Scanner: https://github.com/google/osv-scanner
