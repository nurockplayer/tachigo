# UUID 版本策略

> **最後更新：** 2026-05-13

---

## 目標決策

正式環境 model 的 Primary Key 應優先使用 **UUID v7**，避免新資料以 UUID v4 隨機散落在 B-tree index。

目前實作狀態仍是部分完成；剩餘遷移清單維護在 [`plans/uuid-v7-migration.md`](../plans/uuid-v7-migration.md)。本文件只描述技術策略，不追蹤逐檔進度。

---

## 背景

UUID v4（`gen_random_uuid()`）完全隨機，沒有時間順序。用作 Primary Key 時，每筆新資料插入的位置在 B-tree index 裡隨機分散，導致：

- **Page split** — index 頁面頻繁分裂
- **Index fragmentation** — 讀取效能下降
- **Write amplification** — 寫入越多越慢

UUID v7 前 48 bits 為毫秒時間戳，後接隨機值。新資料永遠插在 index 末端，行為接近自增 INT，同時保留 UUID 的全域唯一性與無中央協調的優勢。

---

## 實作方式

Go 層在 `BeforeCreate` hook 使用 `uuid.NewV7()`（`github.com/google/uuid` v1.6+）：

```go
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.ID == uuid.Nil {
        id, err := uuid.NewV7()
        if err != nil {
            id = uuid.New()
        }
        u.ID = id
    }
    return nil
}
```

PostgreSQL 的 `default:gen_random_uuid()` GORM tag 維持不變作為資料庫層 fallback；正常路徑由 Go 層先生成 ID，因此 DB default 不會觸發。PostgreSQL 17 才有原生 `uuidv7()`，不在此版本做 DB 層改動。

---

## 範圍

正式環境 model 的 `BeforeCreate` hook，以及 service 層直接賦值正式資料列 ID 的地方。測試中的 `uuid.New()` 不需更換（測試 ID 無須時序性）。
