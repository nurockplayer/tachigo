---
title: Dev Portal 視覺重塑與資料整理
status: proposed
owner: engineering
last_reviewed: 2026-05-15
source_of_truth: true
code_areas:
  - apps/docs
  - docs/dev-portal
  - docs
related_repos:
  - tachigo
related_issues:
  - 699
  - 728
---

# Dev Portal 視覺重塑與資料整理

## Context

Phase 1 Link Layer（#694–#698）已將 Dev Portal 內容層建置完成，#699 也將 Cloudflare Pages 部署 repo-side 準備就緒。但目前 Dev Portal 的呈現有兩個顯著問題：

1. **內頁完全沒有自訂樣式**。首頁有 573 行 `custom.css`（拓撲圖、卡片網格），但 `start-here`、`domain-maps`、`flows` 等內頁直接使用 Docusaurus 預設樣式，table 是無底色邊框、heading 沒有層次、整體質感顯著低於首頁。使用者實測後反映「沒有質感」。
2. **資料殘片散落**。`domain-maps.md` 有 5 個「Coming soon」字串、`flows.md` 有 1 個未完成流程，且 sidebar 結構扁平（5 個分類），P0 onboarding 路徑與 reference / archive 條目混雜。

本次工作將以 **engineering console × atlas workbook** 為視覺方向，全面重塑 Dev Portal 的 CSS 系統、補上 3 個 MDX 元件、並重新組織 sidebar 與資料殘片。目標是讓 Dev Portal 在 `*.pages.dev` 公開部署前達到「人類與 AI agent 都覺得有質感且可掃讀」的狀態。

## 視覺方向

採用 **engineering console / atlas workbook**：

- 安靜的文件工作台：灰底 board、白色 card、薄 border、低陰影
- 以藍/綠/橘/紅 token 做狀態與拓撲提示，但避免整頁變成高彩度看板
- 首頁像專案導覽 console；內頁像可掃讀的 engineering runbook
- Callout 保留左側色條語彙，PR2/PR3 若導入 MDX markup 再補更精準的狀態型別

## 設計系統

### 色彩 token

```css
:root {
  --dp-primary: #0052cc;
  --dp-success: #00875a;
  --dp-warning: #ff8b00;
  --dp-neutral: #6b778c;
  --dp-danger: #de350b;

  --dp-bg-board: #f4f5f7;
  --dp-bg-card: #ffffff;
  --dp-bg-subtle: #fafaf9;

  --dp-text-primary: #172b4d;
  --dp-text-secondary: #5e6c84;
  --dp-text-muted: #9b9a97;

  --dp-border: #dfe1e6;
  --dp-border-strong: #c1c7d0;

  --dp-radius-sm: 3px;
  --dp-radius-md: 6px;
  --dp-radius-lg: 8px;

  --dp-shadow-card: 0 1px 0 rgba(9, 30, 66, 0.25);
  --dp-shadow-card-hover: 0 4px 12px rgba(9, 30, 66, 0.15);
}

[data-theme='dark'] {
  --dp-bg-board: #0f1117;
  --dp-bg-card: #1a1d24;
  --dp-text-primary: #f9fafb;
  --dp-text-secondary: #9ca3af;
  --dp-border: #1f2937;
}
```

### 字型

- 內文：`Inter`（既有，移除 Avenir Next / Optima）
- 中文 fallback：`Noto Sans TC`（保留）
- 代碼：`JetBrains Mono`（取代現有 SFMono）

## CSS 元件（自動套用）

| Selector | 樣式 |
|---|---|
| `article table` | 白底、表頭 `--dp-bg-board` 灰底、表格邊框 `1px solid var(--dp-border)`、`border-radius: var(--dp-radius-lg)`、`overflow: hidden`、tbody row `:hover` 加 `background: var(--dp-bg-subtle)` |
| `article h2` | 左側 `4px` 漸層色條（`linear-gradient(180deg, var(--dp-primary), var(--dp-success))`）、`padding-left: 12px`、margin `32px 0 12px` |
| `article h3` | uppercase label 風格（`font-size: 13px; letter-spacing: .08em; color: var(--dp-text-secondary)`） |
| `article :not(pre) > code` | `background: var(--dp-bg-board); color: var(--dp-danger); padding: 1px 5px; border-radius: var(--dp-radius-sm)` |
| `article pre` | 深底 `#172b4d` + 白字、`JetBrains Mono`、`border-radius: var(--dp-radius-lg)` |
| `article blockquote` | 左側 4px 色條（依首字 emoji 變色：💡 黃、⚠️ 橘、🚨 紅、📝 藍）、淡底色 |
| `article a` | `color: var(--dp-primary)` + 點線下底（`text-decoration: underline dotted`），hover 變實線 |

## MDX 元件

### `<StatusBadge status="..." />`

```tsx
type StatusBadgeProps = {
  status: 'complete' | 'draft' | 'planned' | 'blocker' | 'in-progress';
  children?: React.ReactNode;
};
```

對應色彩：complete = `--dp-success`、draft = `--dp-warning`、planned = `--dp-neutral`、blocker = `--dp-danger`、in-progress = `--dp-primary`。

位置：`apps/docs/src/components/StatusBadge/`

### `<DomainCard title="..." status="..." sources={[]} issues={[]} />`

替換 `domain-maps.md` 的 P0/P1 markdown table，以卡片堆疊呈現。每張卡顯示：

- 標題（h3 級別）
- 狀態 badge（用 `<StatusBadge>`）
- Source 連結清單（GitHub blob URLs）
- 相關 issues（GitHub issue links）

位置：`apps/docs/src/components/DomainCard/`

### `<RoadmapStub title="..." status="planned" eta="..." owner="..." />`

取代 `domain-maps.md` 與 `flows.md` 的 5 個 + 1 個「Coming soon」字串，明確顯示狀態、ETA、owner，並提供 placeholder 連結（可連到 issue 或 source entry point）。

位置：`apps/docs/src/components/RoadmapStub/`

## 資料整理

### Sidebar 重組

`apps/docs/sidebars.ts` 改為 4 群組：

```text
🚀 Getting Started (expanded)
  ├─ start-here
  ├─ domain-maps
  ├─ daily-dev-guide
  └─ deployment-tracker (來自 #699，標 in-progress)

🗺️ Architecture & Flows
  ├─ architecture
  ├─ sequence-diagram
  ├─ flows
  ├─ source-index
  ├─ graph-explorer
  ├─ watch-to-points-design
  ├─ auth-architecture
  ├─ tokenomics
  └─ backend-permissions

⚙️ Daily Work
  ├─ AI Workflow（README、autonomous-pr-gates、cheatsheet…）
  └─ Policies（auto-merge、dependabot、scope policy…）

📦 Reference & Archive
  ├─ extension-ui-prompts
  ├─ feature-discussion
  ├─ loyalty-claim-boundary
  ├─ Plans and Proposals
  └─ archive/（搬入 reference-notes 中的 6 個 2026-04-xx ~ 2026-05-xx history）
```

### 「Coming soon」處理

`domain-maps.md` 的 5 個 stub 用 `<RoadmapStub>` 取代：

| Domain | status | 補充說明 |
|---|---|---|
| Raffle / airdrop | `planned` | 標示 Phase 2 |
| Claim / spend / coupon redemption | `draft` | 連結到 cross-repo flow |
| Dashboard | `draft` | 連結到 `apps/dashboard` README |
| Tachiya commerce | `draft` | 連結到 tachiya repo README |
| AI workflow / PR scope policy | `complete` | 連結到 sidebar 既有條目 |

`flows.md` 的 streamer / agency 流程：保留為 `<RoadmapStub status="draft">`，補上 `source-index.md` 已記錄的 entry points 連結。

### History 條目搬移

`docs/reference-notes/` 內 6 個 `2026-04-XX-*.md` 與 `2026-05-XX-*.md` 搬到 `docs/reference-notes/archive/`，避免 sidebar 雜訊。

## 實作分階段

### PR 1：CSS 設計系統（~400-500 行）

- 重寫 `apps/docs/src/css/custom.css`
- 保留 `.tachigo-*` 結構，將顏色與樣式映射到 `--dp-*`；PR 1 只允許 token remap，不移除首頁 DOM / class，以避免首頁樣式回歸
- 加入 `--dp-*` token、字型載入（self-hosted `@fontsource/inter` + `@fontsource/jetbrains-mono`）
- 套用全域 selector（table、h2、h3、code、blockquote、a）

**驗證**：本機 `pnpm start` 跑起來，每頁逐一檢視 light/dark mode，截首頁 + Start Here + Domain Maps 三張圖。

### PR 2：MDX 元件 + high-value 頁面套用（~300-400 行）

- 建立 `apps/docs/src/components/{StatusBadge,DomainCard,RoadmapStub}/`
- 套用到 `docs/index.md`（首頁 status row）
- 套用到 `docs/dev-portal/domain-maps.md`（P0/P1 卡片）
- 套用到 `docs/dev-portal/flows.md`（streamer/agency stub）

**驗證**：本機 build + render，確認 3 個元件在 light/dark mode 都正常。

### PR 3：Sidebar 重組 + 資料整理（~200-300 行）

- 重寫 `apps/docs/sidebars.ts`
- `git mv` history 條目到 `docs/reference-notes/archive/`
- 補完 5 個 Coming soon 的 `<RoadmapStub>` 內容
- `flows.md` streamer/agency stub 補 source entry points

**驗證**：本機 build pass（`onBrokenLinks: 'throw'`），每個 sidebar 群組展開檢查、所有 RoadmapStub 連結有效。

## 驗證流程

1. **本機渲染**：`cd apps/docs && pnpm start`，逐頁檢視
2. **Light / Dark**：每頁切換 theme，確認對比可讀
3. **斷點**：DevTools 切 375px / 768px / 1024px，確認 sidebar collapse、卡片堆疊
4. **Build**：`pnpm build` 通過（內含 broken link check）
5. **Screenshot**：PR description 附首頁 + Start Here + Domain Maps before/after
6. **AI agent fetch**：確認 `/tachigo/llms.txt`、`/tachigo/manifest.json` 內容未被視覺改動破壞

## 不在本次 scope

- 不部署到 Cloudflare Pages（#699 account-side / human gate）
- 不引入 i18n（單一 zh-Hant locale 維持）
- 不重寫拓撲圖 SVG（保留現有 grid layout，僅換配色）
- 不改 markdown content（除 5 個 Coming soon + flows streamer stub）
- 不引入新的搜尋引擎（保留 EasyOps local search）

## 風險

- **首頁 markdown 內嵌大量 `.tachigo-*` class JSX**：PR 1 必須保留既有 class / DOM 結構，只做 token remap；若要移除 class 或重組 markup，需另拆 PR 2/PR 3 範圍
- **字型載入影響 Cloudflare Pages 部署**：外部字型服務改用 self-hosted 或保留 system font，避免外部依賴延遲。建議 PR 1 用 `@fontsource/inter` + `@fontsource/jetbrains-mono`
- **Dark mode 對比測試成本**：建議 PR 1 PR description 附 dark mode 截圖

## 相關 issue

- `#699` — Dev Portal 部署追蹤（本 spec 為其 prerequisite，但不互相 block）
- 待開新 issue：`[frontend] Dev Portal 視覺重塑與資料整理` 作為 epic，三個 PR 各自 closes 子 issue
