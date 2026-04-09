# Tachigo Demo Side Panel Extension

Chrome / Brave MV3 side panel demo for the Tachigo login and loading flow.

## What This Is

- Side panel-first demo extension
- Demo-only local flow: `login -> loading`
- Local persistence for current screen and language

## Local Development

```bash
pnpm install
pnpm dev
```

## Build For Demo

```bash
pnpm build
```

This outputs the unpacked extension bundle to `dist/`.

## Load In Chrome / Brave

1. Open `chrome://extensions` or `brave://extensions`
2. Enable Developer mode
3. Click `Load unpacked`
4. Select the generated `dist/` folder
5. Click the extension toolbar icon to open the side panel

## Verification

```bash
pnpm test:run
pnpm build
```
