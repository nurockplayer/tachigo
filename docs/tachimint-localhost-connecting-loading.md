# tachimint 本機轉圈問題（`Connecting…` 一直不消失）

## 問題現象

在瀏覽器直接打開 `http://localhost:5173/` 時，畫面會一直顯示 spinner 與 `Connecting…`，遲遲看不到（viewer 的）按鈕/商品列表。

## 影響範圍

- 前端：`tachimint`（React + TypeScript + Vite）
- 僅在「本機直接開頁」的開發情境更常遇到（`window` 不在真正 Twitch iframe 內）

## 我觀察到的前端行為（程式依據）

`tachimint/src/App.tsx` 只有在取得到 `context` 之後才會離開「轉圈畫面」：

 - `context === null` → 顯示 `Connecting…`
 - `context !== null` → 進入一般面板畫面（含按鈕）

而 `context` 是在 `useTwitch()` 裡、收到 Twitch Extension 的 `ext.onContext(...)` 回呼後才會被設定。

因此「一直轉圈」幾乎必然代表：`ext.onContext(...)` 的回呼在你當前執行環境沒有被觸發。

## 根因

專案有一套「開發模式 mock」：在 `DEV` 時會注入假 `window.Twitch.ext`，讓本機也能看到面板/按鈕。

但 mock 的注入邏輯原本有個 early-return 條件：

- 只要偵測到 `window.Twitch?.ext` 已存在，就直接不注入 mock

在本機直接開 `localhost:5173` 時，`tachimint/index.html` 會載入 Twitch helper script，導致 `window.Twitch.ext` 可能「看起來存在」，但實際上不在真正 Twitch iframe 情境，因此 `onContext`/`onAuthorized` 回呼不會如預期被觸發。

結果就是：

- mock 因為 `window.Twitch.ext` 已存在而被跳過
- `onContext` 沒被觸發
- `context` 永遠是 `null`
- 轉圈一直存在

## 我做了什麼修正（實際改動）

修改檔案：`tachimint/src/mock/twitch-ext.ts`

把 mock 的 early-return 條件改成：

- **只有在「真的在 iframe」且 `window.Twitch.ext` 已存在**時才跳過注入
- 在「本機 top-level window」時，即使 `window.Twitch.ext` 存在，也會強制注入 mock，避免卡在 `Connecting…`

關鍵邏輯（修正後）：

```ts
export function injectTwitchExtMock() {
  const isInIFrame = window.self !== window.top
  if (window.Twitch?.ext && isInIFrame) return

  // ...其餘 mock 注入邏輯
}
```

## 驗證方式

1. 重新整理 `http://localhost:5173/`
2. 打開瀏覽器 Console，確認是否出現 mock 注入訊息：
   - `"[Twitch.ext mock] injected"`
3. 確認畫面不再卡 `Connecting…`，並能看到 viewer 商品/按鈕面板
4. 若要測 bits transaction 流程，可等待 mock 1s 的交易模擬（由 mock 觸發）

