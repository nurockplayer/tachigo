# Dashboard Deployment Runbook

## Target

Dashboard production hosting target：Cloudflare Pages static site.

| Environment | Branch | Cloudflare Pages project | Public URL | API origin |
| --- | --- | --- | --- | --- |
| Staging | `develop` | `tachigo-dashboard-staging` | `https://tachigo-dashboard-staging.pages.dev` | `https://api.tachigo.io` |
| Production | `main` | `tachigo-dashboard` | `https://admin.tachigo.io` | `https://api.tachigo.io` |

Owner：dashboard release owner for the deployment window.

## Artifact Contract

Build command:

```bash
cd apps/dashboard
pnpm package:production
```

Artifact path：`apps/dashboard/dist`.

Serving strategy:

- Serve `dist/index.html` as the SPA fallback for all dashboard routes.
- Serve `dist/assets/*` as immutable static assets.
- Keep `index.html` no-cache or short-cache so rollbacks and redeploys take effect quickly.

`pnpm package:production` injects `VITE_TACHIGO_API_URL=https://api.tachigo.io`, runs the Vite production build, and then runs `pnpm package:readback`. The readback requires:

- `dist/index.html` exists.
- `dist/assets/` includes at least one JavaScript asset.
- The inspected HTML/JS artifact embeds `https://api.tachigo.io`.
- The inspected HTML/JS artifact does not contain `localhost`, `127.0.0.1`, or `0.0.0.0`.

## Runtime Env

Required production build variable:

| Variable | Value |
| --- | --- |
| `VITE_TACHIGO_API_URL` | `https://api.tachigo.io` |

`VITE_API_URL` remains a local migration fallback for old developer env files. Production builds must use `VITE_TACHIGO_API_URL`; dashboard runtime now fails closed if a production bundle starts without an explicit API URL.

## Deploy Steps

1. Confirm the backend production API at `https://api.tachigo.io` is healthy and has dashboard CORS configured.
2. Run `pnpm package:production` from `apps/dashboard`.
3. Upload or deploy `apps/dashboard/dist` to the Cloudflare Pages project for the target environment.
4. Confirm the Cloudflare Pages deployment SHA matches the git commit being released.
5. Run the smoke checklist below.

## Smoke Checklist

- Login: open `/login`, submit valid dashboard credentials, and confirm the app stores only the in-memory access token.
- Authenticated routing: reload `/`, then navigate to `/streamers`, `/raffles`, `/transactions`, and `/settings`.
- Streamers: list streamers and open a streamer detail page.
- Raffles: list raffles, open a raffle detail page, and verify draw/result controls render.
- Transactions: load points transaction history for an authenticated user.
- Settings: open the settings route and confirm unsupported backend actions do not create failing API calls.
- Network readback: browser requests go to `https://api.tachigo.io`; no request goes to localhost.

## Rollback / Readback

- Rollback owner：dashboard release owner.
- Use Cloudflare Pages deployment history to promote the previous known-good deployment for the same project.
- After rollback, repeat login, authenticated routing, streamers, raffles, transactions, settings, and network readback smoke checks.
- Record the rolled-back-from SHA, restored SHA, reason, and smoke result in the release ticket.
