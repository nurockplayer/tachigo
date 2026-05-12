# Tachimint Extension

`apps/extension` 是 tachigo viewer 端的 Chrome sidepanel / extension frontend，使用 React + TypeScript + Vite。它保留 Twitch Extension auth compatibility，同時以 Chrome Manifest V3 side panel 作為目前主要 runtime。

## 快速啟動

需求：

- Node.js：[`package.json`](package.json) 目前要求 `>=22.12.0`
- pnpm：以 `packageManager` 為準
- 後端 API：預設連到 `http://localhost:8080`

```bash
cd apps/extension
pnpm install
pnpm dev
```

Vite dev server 預設在：

```text
http://localhost:5173
```

開發模式會注入 `window.Twitch.ext` mock，因此一般 UI 開發不需要 Twitch Developer Rig。

常用指令：

```bash
pnpm dev
pnpm build
pnpm test
pnpm lint
pnpm check:i18n
pnpm preview
```

也可以從 repo root 啟動完整 stack：

```bash
make dev
```

## 連接後端

建立本機 env：

```bash
cp apps/extension/.env.example apps/extension/.env
```

| 變數 | 用途 |
| --- | --- |
| `VITE_TACHIGO_API_URL` | Tachigo API origin，例如 `http://localhost:8080` |

Chrome manifest 目前允許 `http://localhost:8080/*` host permission；若本機 API origin 改變，請同步檢查 [`public/manifest.json`](public/manifest.json)。

## Chrome Extension 載入方式

先產出 `dist/`：

```bash
cd apps/extension
pnpm build
```

在 Chrome 載入 unpacked extension：

1. 開啟 `chrome://extensions`
2. 開啟 Developer mode
3. 點選 **Load unpacked**
4. 選擇 `apps/extension/dist`
5. 點擊 Tachimint extension action，或從 Chrome side panel 開啟

目前 Manifest V3 設定在 [`public/manifest.json`](public/manifest.json)：

- `side_panel.default_path` 指向 `sidepanel.html`
- background service worker 由 `src/extension/background.ts` build 成 `assets/background.js`
- content script 由 `src/extension/content.ts` build 成 `assets/content.js`

## `sidepanel.html` 與 `index.html`

| Entry | 用途 |
| --- | --- |
| `sidepanel.html` | Chrome side panel entry；manifest 的 `default_path` 指向這個檔案 |
| `index.html` | Vite popup / legacy Twitch-hosted entry；會載入 Twitch Extension Helper，提供 `window.Twitch.ext` runtime |

兩個 entry 都載入 `src/main.tsx`，實際 UI flow 由 React app 決定。

## 主要模組

| 路徑 | 說明 |
| --- | --- |
| `src/app/` | Viewer UI panels、claim / coupon / raffle result 等互動流程 |
| `src/extension/` | Chrome runtime bridge、background、content、storage 與 manifest-facing types |
| `src/hooks/` | Watch heartbeat、click boost、T-Point、raffle result hooks |
| `src/services/` | Tachigo API client 與 auth recovery |
| `src/i18n/` | i18next setup 與語系資源 |
| `src/mock/` | Dev mode Twitch Extension mock |
| `src/styles/` | Fonts 與 global styles |

## 測試與 build

```bash
cd apps/extension
pnpm test
pnpm lint
pnpm check:i18n
pnpm build
```

修改 extension runtime、manifest 或 API origin 時，請同時確認相關 tests，例如 `src/extension/runtime-config.test.ts` 與 `src/services/api.test.ts`。
