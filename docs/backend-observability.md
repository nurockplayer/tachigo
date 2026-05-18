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

- `tachigo_raffle_scheduler_runs_total{result}`: scheduled snapshot run count. `result` is `success`, `partial_failure`, or `failure`.
- `tachigo_raffle_scheduler_failures_total{result="failure"}`: failed scheduled snapshot batch count. Per-raffle tolerated errors use `result="partial_failure"` and do not increment this counter.
- `tachigo_raffle_scheduler_duration_seconds{result}`: scheduled snapshot duration histogram.

## Tracing baseline

The tracing baseline is OpenTelemetry with OTLP traces. It is vendor-neutral and disabled by default:

- `TRACING_ENABLED=false` is the safe default for local development and production deploys that have not provisioned a collector.
- `OTEL_SERVICE_NAME` defaults to `tachigo-api`; `OTEL_ENVIRONMENT` defaults to `APP_ENV`.
- `OTEL_TRACES_SAMPLE_RATIO` defaults to `0.05` and must stay between `0` and `1` when tracing is enabled.
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` is required when tracing is enabled.
- `OTEL_EXPORTER_OTLP_TRACES_INSECURE=true` is only for trusted local or private-network collectors.
- `OTEL_EXPORTER_OTLP_HEADERS` supports collector auth headers, but those values are secrets and must not be committed or logged.

Each HTTP span records only low-cardinality request metadata: `request_id`, HTTP method, matched Gin route, response status, and Gin error count. Spans intentionally do not record query strings, request bodies, `Authorization`, cookies, OAuth tokens, signing keys, wallet addresses, raw handler error text, or route parameter values.

Sentry remains a future error-reporting adapter candidate only. The current backend implementation does not require a Sentry DSN, vendor account, or SDK initialization. If Sentry is adopted later, it should consume the same request id correlation and redaction policy rather than becoming a second source of PII-bearing request capture.

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
9. Optional: run a local OTLP collector, set `TRACING_ENABLED=true`, `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to the collector traces endpoint, and confirm the span contains `request_id=local-readback-1` without query/body/header secrets.

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
10. Enable tracing only after the staging OTLP collector endpoint is known. Startup must fail fast if `TRACING_ENABLED=true` but `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` is empty or `OTEL_TRACES_SAMPLE_RATIO` is outside `0..1`.
11. Send a request with a known `X-Request-ID` and verify the trace span contains the same `request_id`, matched route, method, and status code.
12. Inspect the span attributes and confirm there is no query string, request body, `Authorization`, cookie, OAuth token, signing key, wallet address, or raw handler error text.

## Sampling, retention, and PII

- Start staging at `OTEL_TRACES_SAMPLE_RATIO=1.0` during short readback windows, then reduce to the production candidate before promotion.
- Start production at `0.05` or lower unless incident response explicitly needs a temporary increase.
- Keep trace retention aligned with backend log retention and incident-response needs; do not use tracing as a long-term customer data store.
- Treat OTLP headers and collector credentials as secrets. Rotate them if they are exposed in shell history, logs, or deployment output.
- Never add span attributes for user email, Twitch login, wallet address, JWT claims, OAuth state, OAuth tokens, request/response bodies, or route parameter values without a new privacy review.

## Production rollout and rollback

Owner: backend/on-call release owner for the deployment window.

Rollout:

1. Confirm staging readback passes with the exact collector endpoint, sampling ratio, and redaction checks.
2. Deploy production with `TRACING_ENABLED=false` first if the release contains unrelated backend changes.
3. Enable tracing by setting `TRACING_ENABLED=true`, `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, `OTEL_SERVICE_NAME=tachigo-api`, `OTEL_ENVIRONMENT=production`, and the approved sampling ratio.
4. Watch startup logs. A missing endpoint or invalid sampling ratio should fail startup before serving traffic.
5. Send one known `X-Request-ID` request and verify log/trace correlation.

Rollback:

1. Set `TRACING_ENABLED=false` and redeploy or roll the service to the last known-good environment config.
2. If exporter auth may be exposed, rotate OTLP headers or collector credentials.
3. Keep structured request logs enabled; they remain the fallback correlation source.

Escalation: if tracing causes startup failure, elevated latency, exporter backpressure, or suspected PII capture, page the backend/on-call release owner and disable tracing before investigating vendor-side configuration.

## Error reporting choice

OTel is the baseline because it keeps instrumentation vendor-neutral and works with any OTLP-compatible collector. Sentry is useful for grouped application errors and release health, but adopting it would introduce vendor-specific SDK behavior and DSN ownership. The next Sentry step should be a separate adapter issue that maps sanitized server errors to Sentry events using `request_id` and OTel trace IDs, with PII redaction tests before rollout.

## Alerting baseline

Staging thresholds:

- Request 5xx rate: warn if `sum(rate(tachigo_http_request_errors_total[5m])) / sum(rate(tachigo_http_requests_total[5m])) > 0.02` for 10 minutes.
- Request latency: warn if the 95th percentile from `tachigo_http_request_duration_seconds` exceeds 1 second for 10 minutes.
- Raffle scheduler failure: warn on any increase in `tachigo_raffle_scheduler_failures_total` over 30 minutes; track `result="partial_failure"` separately for noisy upstream/data issues.

Production thresholds:

- Request 5xx rate: page if the 5xx ratio exceeds 1% for 10 minutes, warn above 0.5% for 15 minutes.
- Request latency: page if the 95th percentile exceeds 2 seconds for 10 minutes, warn above 1 second for 15 minutes.
- Raffle scheduler failure: page on any production scheduler batch failure, because the scheduled snapshot window is daily and user-visible. Treat `result="partial_failure"` as an investigation signal, not a paging signal.

Dashboard readback ownership:

- Backend/on-call release owner owns the `/metrics` scrape readback during deployment.
- Production on-call owns alert acknowledgement and threshold tuning after launch.
- Dashboard panels should group HTTP metrics by `route` and `status_family`, and scheduler metrics by `result` only.

## Owner

Backend production owner: backend/on-call release owner for the deployment window.

Review cadence:

- During staging deploy: verify request logs and scheduler logs before promoting.
- During production launch: keep the log stream, metrics dashboard, and trace readback open for first request traffic and first scheduled job window.
- After launch: keep OTel tracing sampling, retention, and metrics thresholds under review, and handle Sentry/error-reporting as a separate adapter decision.

## Deferred follow-ups

- Future Sentry adapter: sanitized error grouping on top of the OTel/request-id baseline.
