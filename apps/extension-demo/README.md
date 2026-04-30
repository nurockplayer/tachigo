# Tachigo Demo Side Panel Extension

Final demo Chrome / Brave MV3 extension for the Tachigo login, loading, and HUD flow.

## What This Is

- Side panel-first demo extension
- Popup window demo mode is also available from the panel controls
- Demo-only local flow: `login -> loading -> hud`
- Local persistence for current screen, language, and HUD demo state

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
