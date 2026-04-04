# [Discussion] Tachigo 互動挖礦與放置手遊化規劃 (CN/EN)

> 建議分類 / Suggested category: `Ideas`
>  
> 建議標籤 / Suggested labels: `feature`, `gameplay`, `frontend`, `backend`, `twitch-extension`

## 1) 討論目標 / Discussion Goal

我們希望把 Tachigo 目前的觀看點數系統，升級為「可互動、可協作、可成長」的放置手遊體驗：

- 觀眾可透過點擊幫助挖礦角色加速，獲得額外點數
- 全看台可共同完成「安琪拉之門」史詩任務，解鎖頻道專屬 NFT
- 角色具備時裝與裝備加成，主播與經紀公司可施放全體 `PUFF`（觀看點數加成）
- 前端採用類放置手遊 UI，並拆分獨立「裝備」「任務」「商城」介面（參考楓之谷式分頁）

EN:
We want to evolve Tachigo from passive watch-point accrual into an interactive idle-RPG loop with click boosts, raid-like community goals, progression systems, and game-like frontend navigation.

## 2) Epic 拆分總表 / Epic Breakdown Table

| Epic | 功能主題 (中文) | Feature Theme (English) | 優先級 | 相依性 |
|---|---|---|---|---|
| E1 | 挖礦角色點擊增益 | Mining Character Click Boost | P0 | 現有 watch points heartbeat |
| E2 | 安琪拉之門史詩任務 | Angela Gate Community Epic Mission | P0 | E1 點數與事件累計模型 |
| E3 | 角色時裝/裝備/PUFF 增益 | Costumes, Equipment, and Global PUFF Buff | P1 | E1、使用者/頻道設定 |
| E4 | 放置手遊化前端介面 | Idle-Game Style Frontend + Independent Panels | P0 | E1~E3 API 與狀態來源 |
| E5 | 頻道專屬 NFT 解鎖流程 | Channel-exclusive NFT Unlock Flow | P1 | E2 任務完成事件、web3 mint 流程 |

## 3) 需要團隊先對齊的決策 / Decisions Needed

1. 點擊增益上限與防濫用策略（每人每分鐘上限、冷卻、機器人檢測）
2. `PUFF` 加成是否可疊加，以及與裝備加成的計算順序
3. 「安琪拉之門」是固定賽季制還是每週循環制
4. NFT 解鎖條件是全頻道一次性，或每季重置
5. 裝備與時裝是否上鏈（MVP 建議先 off-chain）

## 4) 建議開發順序 / Recommended Delivery Sequence

1. E1 挖礦角色點擊增益（可先上線驗證互動留存）
2. E4 前端分頁化與放置 UI 骨架
3. E2 安琪拉之門社群任務
4. E3 時裝/裝備/PUFF 增益系統
5. E5 任務完成後的頻道 NFT 解鎖

## 5) 對應 Feature Issues / Linked Feature Issues

建議直接用下列 issue 標題建立（完整模板見 `docs/issues-gamification-bilingual.md`）：

- [Feature] 挖礦角色點擊增益 / Mining Character Click Boost
- [Feature] 安琪拉之門社群史詩任務 / Angela Gate Community Epic Mission
- [Feature] 時裝與裝備加成系統 / Costume and Equipment Buff System
- [Feature] 主播/經紀公司全體 PUFF 技能 / Streamer & Agency Global PUFF Skill
- [Feature] 放置手遊化前端與獨立介面 / Idle-Game Frontend with Independent Panels
- [Feature] 頻道專屬 NFT 解鎖流程 / Channel-exclusive NFT Unlock Flow

## 6) 成功指標草案 / Success Metrics (Draft)

- 互動率：每場直播「點擊行為參與觀眾比例」
- 協作率：參與安琪拉之門任務的唯一觀眾數
- 留存：導入後 7 日回訪率、平均觀看時長
- 轉化：裝備購買率、任務完成率、NFT 解鎖率
