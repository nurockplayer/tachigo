# Backend Observability Baseline

## Current baseline

The first production baseline is vendor-neutral structured logging:

- HTTP requests get an `X-Request-ID` response header. Incoming safe request IDs are preserved; otherwise the API generates one.
- Request logs emit `event=http_request`, `request_id`, `method`, matched `route`, sanitized `path`, `status`, `duration_ms`, `client_ip`, and Gin error count.
- Request logs intentionally do not include query strings, request bodies, `Authorization`, cookies, OAuth tokens, signer keys, vouchers, or receipt payloads.
- Raffle scheduled snapshot jobs emit start, per-raffle error, query-error, and completion events with `job=raffle_scheduled_snapshots`, counts, IDs, source, and redacted error text.

## Local readback

1. Start the API locally.
2. Send a request with a known request ID:

   ```bash
   curl -i -H 'X-Request-ID: local-readback-1' 'http://localhost:8080/health?access_token=should-not-log'
   ```

3. Confirm the response contains `X-Request-ID: local-readback-1`.
4. Confirm the API log contains `event=http_request request_id=local-readback-1`.
5. Confirm the same log line does not contain `access_token`, `Authorization`, `Bearer`, cookies, or request body content.

## Staging readback

1. Deploy the backend to staging with normal production-like logging enabled.
2. Run a health request with a known request ID and verify the ID appears in both response headers and centralized logs.
3. Exercise one authenticated route and verify logs show route/status/duration without tokens or cookies.
4. Trigger or simulate a raffle scheduled snapshot window in staging.
5. Verify scheduler logs include `event=raffle_scheduled_snapshots_start` and `event=raffle_scheduled_snapshots_complete`.
6. If a staging raffle snapshot fails, verify `event=raffle_scheduled_snapshot_error` includes `raffle_id`, `user_id`, and `source`, but does not include OAuth tokens.

## Owner

Backend production owner: backend/on-call release owner for the deployment window.

Review cadence:

- During staging deploy: verify request logs and scheduler logs before promoting.
- During production launch: keep the log stream open for first request traffic and first scheduled job window.
- After launch: move request metrics and tracing/error reporting into dedicated follow-up work.

## Deferred follow-ups

- #792: backend metrics and alerting baseline.
- #793: backend tracing and error reporting integration.

