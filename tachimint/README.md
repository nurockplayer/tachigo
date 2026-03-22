# tachimint

Twitch Extension frontend for tachigo.
Built with React + TypeScript + Vite.

## Dev

Run from the repo root:

```bash
make dev          # starts Vite dev server at http://localhost:5173
```

`window.Twitch.ext` is automatically mocked in dev mode.
No Twitch Developer Rig needed for UI development.

## Build

```bash
docker compose run --no-deps --rm frontend npm run build
# output → dist/
```

Upload `dist/` to the Twitch Developer Console to test as a hosted extension.
