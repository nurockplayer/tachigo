# tachimint

Twitch Extension frontend for tachigo.
Built with React + TypeScript + Vite.

目前實作仍依賴 Twitch-hosted runtime / helper 與 `extension_jwt` 流程。
若未來要改成 Chrome Extension，需要另行定義 migration spec，而不是直接把現況文件整批改名。

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
