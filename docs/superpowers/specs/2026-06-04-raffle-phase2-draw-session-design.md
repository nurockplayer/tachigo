# Raffle Phase 2 — 逐輪抽獎模式 設計文件

**日期**：2026-06-04
**關聯 issue**：原始討論 [#234](https://github.com/nurockplayer/tachigo/issues/234)（已收斂為本設計，Campaign 層不另行實作）；實作追蹤 [#1043](https://github.com/nurockplayer/tachigo/issues/1043)
**狀態**：待實作

---

## 背景

Phase 1 已完整實作抽獎系統核心：`Raffle` → `RafflePrizeTier` → `RaffleDraw`。其中 `DrawFromTierContext` 已實作 cross-tier 排除（`raffle_service.go:1264`），不同獎項層之間得獎者不重複。

Issue #234 提出 Campaign 多場抽獎層設計，但確認需求後，Phase 2 的核心場景為：

> 同一場直播中，主播依序從多個獎項層抽獎，每輪抽完後暫停展示得獎者，確認後再手動推進到下一輪。

由於後端資料層已完整支援此場景，Phase 2 為**純前端 UX 工作**，不新增資料表、不新增 API 端點。

---

## 不做

- Campaign 資料表與 API
- 每輪不同報名資格 / 不同名單（留待 Phase 3 視需求決定）
- 自動進輪（每輪結束後系統自動觸發下一輪）
- 主播自由選輪次順序（固定依 `position` 排序）

---

## 設計

### 輪次順序

`RafflePrizeTier` 依 `position` 欄位升序排列，由小到大依序抽。主播在活動建立時設定 position，開播後固定順序。

### 狀態機

```
[idle]
  ↓ Raffle 狀態為 active，PrizeTiers 存在
[round_ready]  ── 顯示當前 tier 名稱、獎項說明、本輪名額
  ↓ 主播按「抽這一輪」→ 呼叫 drawFromTier API
[drawing]  ── loading 狀態
  ↓ API 回傳成功
[round_result]  ── 展示得獎者名稱；若本輪 winner_count > 1，可繼續抽同輪
  ↓ 本輪 drawn_count === winner_count，主播按「繼續下一輪」
[round_ready]  ── 移至下一個 tier
  ↓ 所有 tier 的 drawn_count === winner_count
[session_complete]  ── 顯示完整得獎名單
```

### 前端元件

| 元件 | 職責 |
|---|---|
| `RaffleDrawSession` | 逐輪模式容器；持有 `currentTierIndex` state；協調狀態機轉換 |
| `TierRoundCard` | 顯示當前 tier 資訊（名稱、獎項說明、已抽 N / 共 M 名）與「抽這一輪」按鈕 |
| `DrawResultOverlay` | 得獎者展示層（名字 + 既有 ocean 主題動畫）；「繼續下一輪」按鈕 |
| `FinalWinnerList` | session_complete 狀態的完整得獎名單，依輪次分組 |

### 與現有元件的關係

- `RaffleDrawSession` 嵌入現有 `RaffleDetailPage`，取代現有的單次抽獎按鈕區塊（當 PrizeTiers 存在時顯示）
- `drawFromTier` service function（`apps/dashboard/src/services/raffles.ts`）直接複用，不新增
- Ocean 主題動畫元件複用現有實作（`RaffleDetailPage.tsx` 中已有 keyframes）

### API 使用

不新增端點，全部使用現有：

```
POST /api/v1/dashboard/raffles/:id/tiers/:tier_id/draw   ← drawFromTier
GET  /api/v1/dashboard/raffles/:id/tiers                 ← listPrizeTiers（取得 tiers + drawn_count）
GET  /api/v1/dashboard/raffles/:id/draws                 ← listDraws（session_complete 時取完整名單）
```

---

## 完成條件

- [ ] 所有 PrizeTier 依 position 順序逐輪顯示
- [ ] 每輪按鈕觸發 `drawFromTier`，顯示得獎者名稱
- [ ] 同一輪 winner_count > 1 時，可在同一輪繼續抽足名額再進下一輪
- [ ] 所有輪次抽完後顯示完整得獎名單
- [ ] 重複得獎不可能發生（由後端 cross-tier 排除保證，前端無需額外處理）
- [ ] 既有非 PrizeTier 的單次抽獎流程不受影響

---

## 驗證方式

1. 建立含 3 個 PrizeTier 的 Raffle（一等獎 × 1、二等獎 × 2、三等獎 × 3）
2. 走完完整抽獎流程，確認共 6 名不重複得獎者
3. 確認各輪得獎者姓名正確顯示於 `FinalWinnerList`
