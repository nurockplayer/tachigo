# tachimint

Chrome Extension frontend for tachigo.
Built with React + TypeScript + Vite.

## Dev

Run from the repo root:

```bash
make dev          # starts Vite dev server at http://localhost:5173
```

`window.Twitch.ext` is automatically mocked in dev mode as a legacy compatibility layer.
No Twitch Developer Rig is required for current UI development.

## Build

```bash
docker compose run --no-deps --rm frontend npm run build
# output → dist/
```

Current product direction is Chrome Extension. If legacy Twitch-hosted testing notes still appear elsewhere, treat them as outdated documentation rather than the target delivery format.
