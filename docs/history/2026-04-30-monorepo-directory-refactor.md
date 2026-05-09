# Monorepo 目錄重整紀錄（2026-04）

> 狀態：已完成，保留作為歷史遷移紀錄
> 最後校正：2026-05-03

## 背景

原始 repo 的頂層目錄隨著專案成長變得混亂：
- 前端散落在 `tachimint/`、`dashboard/` 兩個分離目錄
- 後端直接放在 `backend/`，與其他資料夾平行但缺乏語意分層

這次重整的目標是讓目錄結構能反映「這個 repo 裡有哪些服務」，方便未來新增服務時有明確的放置位置。

## 遷移前後對照

```
遷移前                          遷移後
──────────────────────────────  ──────────────────────────────
tachigo/                        tachigo/
├── backend/                    ├── services/
│   └── (Go API)                │   └── api/        ← 後端 Go 服務
├── tachimint/                  ├── apps/
│   └── (Chrome extension)      │   ├── extension/  ← Chrome extension 前端
├── dashboard/                  │   └── dashboard/  ← 後台管理前端
│   └── (React 後台)            ├── contracts/
├── contracts/                  ├── docs/
└── docs/                       └── ...
```

## 執行的 PR

### PR #417 — 前端移入 apps workspace

- `tachimint/` → `apps/extension/`
- `dashboard/` → `apps/dashboard/`
- 建立 pnpm workspace（`pnpm-workspace.yaml`）
- 更新 Docker Compose、Dependabot、LFS 路徑、相關 docs/scripts
- Issue：#396 | Merged：2026-04-30

### PR #435 — 後端移入 services/api

- `backend/` → `services/api/`
- Go module path 不變（`github.com/nurockplayer/tachigo`）
- 更新 Docker Compose、Makefile、CI、setup scripts
- 屬於純路徑遷移，不改 API contract、migration、runtime behavior
- Issue：#397 | Merged：2026-04-30

### PR #438 — CI 優化：移除 artifact roundtrip

- `backend-build` job 移除 artifact export/upload（只保留 Docker build 驗 cache）
- `backend` job 改用 native `go test / go vet`（`actions/setup-go@v6`）
- 兩個 job 改為平行執行，不再有 artifact 依賴
- 實測節省約 2.5 分鐘（gzip 66s + upload 19s + download 16s + docker load 50s）
- Issue：#437 | Merged：2026-04-30

## 後續項目狀態

| Issue | 內容 | 狀態 |
|---|---|---|
| #399 | 整合 monorepo 遷移後的 infra assets 與 AI instructions | 已關閉（2026-05-02） |
| #400 | 統一前端 API base URL env 命名 | 已關閉（2026-05-02） |
| #385 | check-backend-ci-cache.sh artifact pipeline 補強 | 2026-05-03 快照：仍 open；artifact roundtrip 已由 PR #438 移除 |

## 不變的事項

- Go module path：`github.com/nurockplayer/tachigo`
- API contract 與 Swagger annotations
- Database migration 內容
- Runtime behavior
- `contracts/`、`docs/`、`deployments/` 位置
