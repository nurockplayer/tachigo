# Backend Observability Baseline

## Current baseline

The first production baseline includes vendor-neutral structured logging and
Prometheus-compatible text metrics:

- HTTP requests get an `X-Request-ID` response header. Incoming safe request IDs are preserved; otherwise the API generates one.
- Request logs emit `event=http_request`, `request_id`, `method`, matched `route`, sanitized `path`, `status`, `duration_ms`, `client_ip`, and Gin error count.
- Request logs intentionally do not include query strings, request bodies, `Authorization`, cookies, OAuth tokens, signer keys, vouchers, or receipt payloads.
- Raffle scheduled snapshot jobs emit start, per-raffle error, query-error, and completion events with `job=raffle_scheduled_snapshots`, counts, IDs, source, and redacted error text.
- When `ENABLE_METRICS=true`, the backend exposes `/metrics` in Prometheus text format.
- Production deployments with metrics enabled must set `METRICS_BEARER_TOKEN`; scrapers send `Authorization: Bearer <token>`.
- Metrics labels are intentionally limited to Gin route pattern plus `status_family`, or scheduler `result`. They must not include query strings, bodies, `Authorization`, cookies, tokens, user IDs, vouchers, or receipts.

## Metrics inventory

HTTP metrics:

- `tachigo_http_requests_total{route,status_family}`: request count.
- `tachigo_http_request_errors_total{route,status_family}`: 5xx response count.
- `tachigo_http_request_duration_seconds{route,status_family}`: request latency histogram.

Raffle scheduler metrics:

- `tachigo_raffle_scheduler_runs_total{result}`: scheduled snapshot run count.
- `tachigo_raffle_scheduler_failures_total{result="failure"}`: failed scheduled snapshot run count.
- `tachigo_raffle_scheduler_duration_seconds{result}`: scheduled snapshot duration histogram.

## Local readback

1. Start the API locally.
2. Send a request with a known request ID:

   ```bash
   curl -i -H 'X-Request-ID: local-readback-1' 'http://localhost:8080/health?access_token=should-not-log'
   ```

3. Confirm the response contains `X-Request-ID: local-readback-1`.
4. Confirm the API log contains `event=http_request request_id=local-readback-1`.
5. Confirm the same log line does not contain `access_token`, `Authorization`, `Bearer`, cookies, or request body content.
6. Enable metrics locally:

   ```bash
   ENABLE_METRICS=true METRICS_BEARER_TOKEN=local-metrics-token make run
   ```

7. Scrape metrics after a synthetic request:

   ```bash
   curl -s -H 'Authorization: Bearer local-metrics-token' http://localhost:8080/metrics
   ```

8. Confirm the output includes `tachigo_http_requests_total` and does not include query strings, request bodies, `Authorization`, cookies, bearer tokens, vouchers, receipts, or user IDs.

## Staging readback

1. Deploy the backend to staging with normal production-like logging enabled.
2. Run a health request with a known request ID and verify the ID appears in both response headers and centralized logs.
3. Exercise one authenticated route and verify logs show route/status/duration without tokens or cookies.
4. Trigger or simulate a raffle scheduled snapshot window in staging.
5. Verify scheduler logs include `event=raffle_scheduled_snapshots_start` and `event=raffle_scheduled_snapshots_complete`.
6. If a staging raffle snapshot fails, verify `event=raffle_scheduled_snapshot_error` includes `raffle_id`, `user_id`, and `source`, but does not include OAuth tokens.
7. Set `ENABLE_METRICS=true` and `METRICS_BEARER_TOKEN` in staging.
8. Scrape `/metrics` with the bearer token after synthetic requests and a scheduler exercise.
9. Confirm `tachigo_http_requests_total`, `tachigo_http_request_duration_seconds`, and `tachigo_raffle_scheduler_runs_total` advance.

## Alerting baseline

Staging thresholds:

- Request 5xx rate: warn if `sum(rate(tachigo_http_request_errors_total[5m])) / sum(rate(tachigo_http_requests_total[5m])) > 0.02` for 10 minutes.
- Request latency: warn if the 95th percentile from `tachigo_http_request_duration_seconds` exceeds 1 second for 10 minutes.
- Raffle scheduler failure: warn on any increase in `tachigo_raffle_scheduler_failures_total` over 30 minutes.

Production thresholds:

- Request 5xx rate: page if the 5xx ratio exceeds 1% for 10 minutes, warn above 0.5% for 15 minutes.
- Request latency: page if the 95th percentile exceeds 2 seconds for 10 minutes, warn above 1 second for 15 minutes.
- Raffle scheduler failure: page on any production scheduler failure, because the scheduled snapshot window is daily and user-visible.

Dashboard readback ownership:

- Backend/on-call release owner owns the `/metrics` scrape readback during deployment.
- Production on-call owns alert acknowledgement and threshold tuning after launch.
- Dashboard panels should group HTTP metrics by `route` and `status_family`, and scheduler metrics by `result` only.

## Owner

Backend production owner: backend/on-call release owner for the deployment window.

Review cadence:

- During staging deploy: verify request logs and scheduler logs before promoting.
- During production launch: keep the log stream and metrics dashboard open for first request traffic and first scheduled job window.
- After launch: move tracing/error reporting into dedicated follow-up work.

## Deferred follow-ups

- #793: backend tracing and error reporting integration.
