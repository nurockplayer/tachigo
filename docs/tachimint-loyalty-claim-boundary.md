# Tachimint 與 Claim Flow 邊界說明

> 用途：快速對齊產品方向，避免把 `tachimint` 的忠誠點數累積流程，和平台幣兌換流程混在一起。
> 定位：討論用短文件，不是正式 architecture source of truth。

---

## 1. 一句話版本

`tachimint` 負責讓觀眾在 Twitch Extension 內累積忠誠點數 `T-Point`；另一個獨立按鈕或入口，負責把 `T-Point` 兌換成平台幣 `$TACHI`。

---

## 2. 產品邊界

### `tachimint` 要負責的事

- 顯示目前 `T-Point` 餘額
- 透過 heartbeat 累積觀看點數
- 透過互動按鈕提供額外點數回饋
- 顯示 Bits 商品與互動狀態
- 維持小尺寸、快速瞥視的 extension panel 體驗

### `tachimint` 不應該優先承擔的事

- 完整 wallet 連接流程
- 鏈上交易細節
- 複雜的 claim / redeem 多步驟流程
- 平台幣資產管理頁
- 一整套 web3 dApp 體驗

---

## 3. Claim 按鈕應該代表什麼

這個按鈕的意義是：

- 把已累積的 `T-Point` 轉換成平台幣 `$TACHI`
- 它是「兌換入口」，不是「繼續累積點數的互動」

因此，Claim Flow 應被視為獨立能力：

- 可以從 extension 內放一個入口按鈕進去
- 也可以導向另一個更適合完成兌換的畫面
- 但它的產品責任，和 `tachimint` 的點數累積責任要分開看

---

## 4. 最推薦的拆法

### A. Extension 主畫面

重點放在：

- 你現在有多少 `T-Point`
- 你現在能不能繼續挖 / 累積
- 你現在能不能買 Bits 商品

### B. Claim 入口

提供一個清楚但不喧賓奪主的按鈕，例如：

- `Claim $TACHI`
- `兌換平台幣`
- `Convert Points`

### C. Claim 畫面 / 流程

重點放在：

- 你目前可兌換多少 `T-Point`
- 兌換後會得到多少 `$TACHI`
- 這次 claim 是否只是 DB 記帳，或已經觸發鏈上 mint
- 成功 / 失敗 / wallet 狀態

---

## 5. Phase 1 / Phase 2 切法

### Phase 1

先做：

- `tachimint` 持續累積 `T-Point`
- Claim 按鈕呼叫後端 claim API
- 後端把 `T-Point` 轉進 `tachi_balances`
- 先不碰鏈上 mint

這一階段的使用者理解可以是：

- `T-Point` 是忠誠點數
- `$TACHI` 是平台內記帳的可兌換平台幣

### Phase 2

再升級：

- claim 與 wallet 綁定
- 接 Sepolia / 合約 mint
- 將 `$TACHI` 從 DB-only 餘額升級成鏈上平台幣流程

這一階段才需要處理：

- wallet UX
- 合約部署
- 簽章 / relayer / gas
- on-chain 成功與失敗狀態

---

## 6. UI 設計上的意思

如果照這個方向，UI 上應該這樣理解：

- `tachimint` 是 loyalty points panel
- Claim 是 secondary action，不是主互動
- 主視覺仍應放在點數累積，而不是放在 web3 錢包操作
- 若 claim 流程開始變複雜，就應考慮抽到獨立頁面，而不是一直塞進 extension panel

---

## 7. 對實作者最重要的結論

- 不要把 `tachimint` 直接做成錢包 dApp
- 先把 `T-Point` 累積體驗做清楚
- 把 claim 視為「另一個能力入口」
- Phase 1 先跑通 DB-only claim
- Phase 2 再接 Sepolia

---

## 8. 可直接拿去討論的版本

```text
Tachigo 的產品拆分應該是：

1. `tachimint` 專注在 Twitch Extension 內的忠誠點數累積體驗
2. Claim 按鈕是把 `T-Point` 兌換成 `$TACHI` 的入口
3. Phase 1 先做 DB-only claim，不讓 extension 承擔完整 web3 流程
4. Phase 2 再把 claim 升級成 Sepolia / on-chain mint

這樣可以保持 extension 輕量，也能讓平台幣流程獨立演進。
```
