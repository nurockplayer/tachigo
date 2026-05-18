# Cross-Surface Smoke Baseline

## Automated Scope

The first launch smoke baseline is a static dashboard/extension-to-API contract check:

```bash
pnpm smoke:contracts
```

It verifies that key dashboard and extension API paths still line up with backend router registrations and, where already generated, `services/api/docs/swagger.json`.

Covered automated paths:

- Dashboard streamers list and streamer channels.
- Dashboard channel config.
- Dashboard raffle list and raffle draws.
- Dashboard transactions history.
- Extension login.
- Extension watch heartbeat.
- Extension raffle result.

The command runs in CI through the `API contract drift` job when relevant backend router, Swagger, dashboard API, extension API, or contract-smoke files change.

## Manual UAT

These flows depend on external platforms or browser extension review state and remain manual for the non-Web3 MVP:

- Twitch Extension identity / helper behavior in a real Twitch channel context.
- Chrome Web Store package submission and permission review.
- Tachiya commerce / coupon redemption calls outside the repo-local API contract.
- Real dashboard login credentials, role assignment, and session restore against staging.

Manual UAT should record the target URL, git SHA, account/role used, API origin, and pass/fail notes in the release ticket.
