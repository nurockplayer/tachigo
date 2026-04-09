# tachimint

Frontend for tachigo. Product positioning is Chrome Extension, while the current repository still contains Twitch-specific runtime and helper integration.
Built with React + TypeScript + Vite.

## Dev

Run from the repo root:

```bash
make dev          # starts Vite dev server at http://localhost:5173
```

`window.Twitch.ext` is automatically mocked in dev mode for local UI development.
The current implementation still depends on Twitch-specific helper/runtime behavior in several places.

## Build

```bash
docker compose run --no-deps --rm frontend npm run build
# output → dist/
```

The product direction is Chrome Extension. Existing Twitch-hosted runtime/testing notes in the repo should be treated as current implementation details that still need follow-up cleanup, not as the intended long-term product framing.
