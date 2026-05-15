---
title: Deployment Tracker
status: active
owner: engineering
last_reviewed: 2026-05-14
source_of_truth: true
code_areas:
  - docs
  - apps/docs
related_repos:
  - tachigo
related_issues:
  - 699
---

# Deployment Tracker

這頁記錄 `tachigo` Dev Portal 部署到 Cloudflare Pages 的 repo-side 設定與 readback checklist。範圍只涵蓋靜態 docs hosting，不擴張成 backend runtime 或其他 Cloudflare 產品導入。

## Scope

- 手動在 Cloudflare Pages 後台連接 GitHub repo
- 記錄 docs build 所需的 install / build / output / root 設定
- 定義 `develop` production deploy 與 PR preview 的預期行為
- 保留公開 URL 驗證、rollback 與 readback checklist

## Manual setup values

| 欄位 | 值 |
|---|---|
| Provider | Cloudflare Pages |
| Account ownership | Cloudflare org / team account |
| Production branch | `develop` |
| Root directory | repo root (`/`) |
| Install command | `pnpm install --frozen-lockfile --ignore-scripts` |
| Build command | `pnpm build:docs` |
| Build output directory | `apps/docs/build` |
| Node / package manager | Node 24 via `.node-version` / `pnpm@10.33.0` / lockfile |
| Base path | `/tachigo/` via static `_redirects` rewrites |
| Environment variables | optional `DOCS_SITE_URL`; Cloudflare injects `CF_PAGES_URL` |
| Initial public URL | Cloudflare `*.pages.dev` URL |
| Custom domain | 延後決定；目前偏好 `wiki.tachigo.dev` |

Cloudflare 後台操作重點：

1. 選擇 `nurockplayer/tachigo` repo。
2. 使用 Cloudflare org / team account，不使用個人帳號作為長期 ownership。
3. Framework preset 可維持 generic static site，只手動覆蓋 `Production branch`、`Root directory`、`Install command`、`Build command` 與 `Build output directory`。
4. Install command 必須加上 `--ignore-scripts`，避免 hosted deploy 在 install phase 執行 dependency lifecycle script。
5. Root 必須是 repo root，不是 `apps/docs`，因為 build script 定義在 workspace root。
6. Production branch 必須是 `develop`；不要沿用 GitHub Pages 或預設 `main` production 設定。
7. 保留 `apps/docs/static/_redirects`，讓 Cloudflare Pages 在靜態 output root 下仍能服務 `/tachigo/*` URL。
8. 若 production `*.pages.dev` URL 已知，可把 `DOCS_SITE_URL` 設為該 URL；未設定時 Docusaurus 會使用 Cloudflare 注入的 `CF_PAGES_URL`，再 fallback 到 `https://tachigo-dev-portal.pages.dev`。
9. 第一次連線完成後，先確認 Cloudflare 讀到的 branch、Node version、environment variables 與 build config 都和這頁一致，再觸發初次 deploy。

## Preview readback

Cloudflare Pages 的 preview 行為不能只靠填一個欄位宣稱完成。連 repo 後必須從 Pages dashboard 與實際 URL 讀回：

| 讀回項目 | 預期 |
|---|---|
| Account ownership | Pages project 位於可交接的 org / team account |
| Production deployment | branch 是 `develop`，commit SHA 對得上 GitHub 的 `develop` head |
| PR preview | 新開 PR 後會產生 preview URL，URL 對應該 PR branch / commit |
| Build config | root、install、build、output 值與上表一致 |
| Base path rewrite | `/tachigo/`、`/tachigo/llms.txt`、`/tachigo/manifest.json` 不回 404 |
| Site URL config | `DOCS_SITE_URL` 或 Cloudflare 注入的 `CF_PAGES_URL` 對應目前 deployment URL |
| Public URL | 初期使用 Cloudflare `*.pages.dev` URL，不要求 custom domain |

若 production deploy 或 PR preview 沒有出現在 dashboard，先不要把 #699 關成完成；補 Cloudflare Pages 設定或另開 issue 追蹤。

## Branch deploy model

| 分支 / 情境 | 預期 |
|---|---|
| `develop` | production docs URL；對外穩定入口與最新整合狀態 |
| PR branch | 每個 PR 自動產生 preview URL，供 reviewer 驗證 docs、搜尋與靜態資產 |

實務上要把 `develop` 視為 Cloudflare Pages production branch；`main` 不參與本階段 Pages production 設定。

## Domain gate

預設先接受 Cloudflare 提供的 `*.pages.dev` URL，等下列條件滿足後再由人類決定是否綁 custom domain。目前偏好是 `wiki.tachigo.dev`，但這不是本階段上線 gate。

- Cloudflare 帳號歸屬已確認是 org / team account
- `develop` production / PR preview 的 deploy 與 rollback 流程已走通一次
- DNS owner、TLS、子網域命名是否要對外長期承諾，已有明確結論

在這個 gate 完成前，`pages.dev` 已足以支撐 `/tachigo/llms.txt`、`/tachigo/manifest.json`、搜尋與 docs 頁面的公開驗證。

## Public URL verification checklist

每次首次上線、重大 docs pipeline 調整，或切換 custom domain 前，至少 readback 這些項目：

- [ ] `develop` production URL 可開啟首頁與 [Start Here](/tachigo/dev-portal/start-here)
- [ ] 任一 PR preview URL 可正常載入，且不指向舊 build
- [ ] `/tachigo/llms.txt` 可公開存取
- [ ] `/tachigo/manifest.json` 回傳合法 JSON
- [ ] 搜尋框可開啟並命中至少一份核心 doc（例如 `watch points`）
- [ ] [Source Index](/tachigo/dev-portal/source-index) 與至少一份 root source-of-truth doc 可正常導覽
- [ ] 頁面資產、樣式與站內連結沒有 base path 錯位

若 production URL 尚未綁 custom domain，以上檢查在 `pages.dev` URL 完成即可；切 custom domain 後再完整重跑一次。

## Rollback and readback

Cloudflare Pages rollback 以後台 deployment history 為主，repo 端只記錄必要 readback：

1. 在 Pages deployments 清單選定上一個已知正常版本並 rollback。
2. 讀回 rollback 後的 deployment commit SHA、branch、建立時間與 URL。
3. 重新跑一次上面的 public URL checklist，至少驗證首頁、`llms.txt`、`manifest.json`、搜尋。
4. 若 rollback 只影響 `develop` 或單一 PR preview，也要記錄 production 是否未受影響。

建議每次 rollback 都補一段簡短 readback 紀錄，至少包含：

- rollback 目標 deployment id / commit SHA
- rollback 前後 URL
- 重新驗證結果
- 是否需要後續 issue 追蹤 build config、內容或 domain 問題

## Explicit non-goals

這張票與這份文件明確不做：

- Cloudflare Workers
- Cloudflare Functions
- Cloudflare Access / Zero Trust
- backend runtime、API、database 或其他 server-side deployment
- `.github/workflows/deploy-docs.yml`
- GitHub Pages 自動部署流程
- Cloudflare bindings
- Cloudflare Workers / Functions rewrite logic；`/tachigo/*` 只能使用 static `_redirects`

若未來真的要改成 GitHub Pages、補 deploy workflow，或引入其他 Cloudflare 能力，應拆成新 issue / PR。
