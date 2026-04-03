# UUID 版本策略

> **最後更新：** 2026-04-01

---

## 決策

所有 model 的 Primary Key 使用 **UUID v7**，不使用 UUID v4。

---

## 背景

UUID v4（`gen_random_uuid()`）完全隨機，沒有時間順序。用作 Primary Key 時，每筆新資料插入的位置在 B-tree index 裡隨機分散，導致：

- **Page split** — index 頁面頻繁分裂
- **Index fragmentation** — 讀取效能下降
- **Write amplification** — 寫入越多越慢

UUID v7 前 48 bits 為毫秒時間戳，後接隨機值。新資料永遠插在 index 末端，行為接近自增 INT，同時保留 UUID 的全域唯一性與無中央協調的優勢。

---

## 實作方式

Go 層在 `BeforeCreate` hook 使用 `uuid.New7()`（`github.com/google/uuid` v1.6+）：

```go
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.ID == uuid.Nil {
        u.ID = uuid.New7()
    }
    return nil
}
```

PostgreSQL 的 `default:gen_random_uuid()` GORM tag 維持不變作為 fallback（Go 層永遠先生成，DB default 不會觸發）。PostgreSQL 17 才有原生 `uuidv7()`，不在此版本做 DB 層改動。

---

## 範圍

所有 model 的 `BeforeCreate` hook，以及 service 層直接賦值 `ID: uuid.New()` 的地方。測試中的 `uuid.New()` 不需更換（測試 ID 無須時序性）。
