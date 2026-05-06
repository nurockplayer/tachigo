# Contracts Gas Snapshot Policy

**Source of truth**: #507
**Status**: report-only first rollout

## Decision

The first production-readiness rollout keeps gas snapshots report-only. CI runs
`forge snapshot --snap gas-snapshot.report` for contracts changes and uploads
the `contracts-gas-snapshot-report` artifact for review.

`.gas-snapshot` is not committed yet because the project has not assigned a
baseline owner or agreed on a drift tolerance. This avoids turning normal
contract iteration into a low-signal blocking gate before the team has reviewed
the first baseline.

## Tolerance

- No tolerance is active while the job is report-only.
- When the baseline is intentionally committed, start with `0%` tolerance unless
  the baseline review records a specific noisy measurement.
- Any non-zero tolerance must be documented in the PR that enables
  `forge snapshot --check`.

## Intentional gas changes checklist

- Link the contract or test change that caused the gas delta.
- Paste the relevant lines from the `contracts-gas-snapshot-report` artifact.
- Explain whether the delta is expected, acceptable, and user-visible.
- Reviewer accepted the gas impact before merge.

## Promotion conditions for a blocking drift gate

- Commit `.gas-snapshot` with an explicit owner.
- Update CI to run `forge snapshot --check` against the committed baseline.
- Document the tolerance value and how intentional deltas update the baseline.
