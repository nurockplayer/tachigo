# Backend Request Context Baseline

## Purpose

This document records the current request-context boundary for backend DB work
tracked by #645. It is a review aid for request timeout cancellation work; it
does not introduce new timeout policy, queue architecture, or worker behavior.

## Request-scoped DB rule

HTTP handlers should pass `c.Request.Context()` into service methods that accept
`context.Context`. Service-layer DB reads, writes, and transactions that belong
to that request should use `db.WithContext(ctx)` or a transaction that inherited
that context.

Legacy wrapper methods that do not accept a context may use
`context.Background()` only as a compatibility boundary for non-request callers
and tests. New request-serving paths should prefer the `*Context` form.

## Transaction context rule

Transactions started from `db.WithContext(ctx).Transaction(...)` carry the
request context through `tx.Statement.Context`. Helpers that need to re-enter
GORM with an explicit context should preserve that statement context rather than
falling back to a fresh background context.

Example: `loadTachiBalanceValue(ctx, db, userID)` accepts an explicit context,
and `finalizeClaim` calls it with `gormStatementContext(tx)` so balance readback
stays aligned with the transaction context.

## Intentionally detached DB paths

The following paths are intentionally not a mechanical `WithContext(requestCtx)`
replacement target. Some are true async jobs; others are synchronous
post-side-effect persistence paths where cancellation semantics must preserve
reconciliation evidence before they are changed.

| Path | Current boundary | Why detached |
| --- | --- | --- |
| `RaffleScheduler.Start` scheduled snapshots | The scheduler is launched with its own lifecycle context. Each run creates a five-minute child context and passes it through `RunScheduledSnapshots`, `snapshotOne`, and DB calls that use `s.db.WithContext(ctx)`. | This is not tied to an HTTP request. The scheduler should stop on process/service lifecycle cancellation, not on any viewer/admin request cancellation. |
| Raffle winner email / Discord notification goroutines | `DrawNext` / `DrawFromTier` start short-lived goroutines that create a fresh 30-second background timeout before calling notification helpers. Those helpers use that timeout for DB lookup and outbound delivery. | Notification delivery is intentionally best-effort after the draw is already committed. It should not inherit a canceled request context and undo or block the draw response path. |
| `ClaimService.Claim` post-mint / compensation transactions | The initial reservation transaction uses `s.db.WithContext(ctx)`. Rollback, broadcast recording, receipt-unknown recording, failure compensation, finalize, and finalize-failed marking currently use service DB transactions without the request context. | After a mint transaction is broadcast, a canceled HTTP request must not prevent local claim status, compensation, or reconciliation markers from being persisted. A future change needs dedicated tests for canceled requests after broadcast and receipt uncertainty. |
| `SpendService.Redeem` after burn / Tachiya handoff | The initial reservation transaction uses `s.db.WithContext(ctx)`. Burn uses a child timeout of the request context. Coupon redemption recording, post-burn status updates, and reservation rollback currently use service DB operations without the request context; Tachiya redemption uses its own background timeout. | After a burn is broadcast, local `coupon_redemptions` rows and compensation-needed state are the recovery trail. Existing tests assert the Tachiya call can outlive a canceled request after burn. |

## Change requirements

Any PR that converts one of the detached paths above to request-scoped DB work
must include regression coverage for the cancellation point it changes. At
minimum, tests should prove that a canceled request cannot leave one of these
states without a durable recovery marker:

- claim reservation consumed but no claim status or compensation marker;
- mint broadcast recorded by chain but missing local claim broadcast evidence;
- burn broadcast recorded by chain but missing local coupon redemption evidence;
- Tachiya voucher issued but local redemption status missing reconciliation data.

If the desired behavior is to keep post-chain persistence detached, keep that
decision documented here instead of hiding it in broad context-propagation
cleanup.
