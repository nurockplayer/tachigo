# Raffle Prize Tiers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在現有 Raffle 上新增 Prize Tier 層，讓 Streamer 可在同一場抽獎裡辦多個獎項，已中獎者自動排除於後續獎項之外。

**Architecture:** 新增 `raffle_prize_tiers` 資料表掛在現有 `raffles` 下，`raffle_draws` 加 `prize_tier_id` nullable FK。新增 5 個 service 方法（CRUD + DrawFromTier）、5 個 handler、對應路由，以及 Dashboard UI 的獎項區塊。現有 `DrawNext` 邏輯與所有現有端點不動。

**Tech Stack:** Go 1.23 / Gin / GORM / PostgreSQL（prod） / SQLite（test）、Atlas migrations、React + TypeScript / Refine + TanStack Query

---

## 異動檔案總覽

| 檔案 | 動作 |
|---|---|
| `services/api/migrations/021_raffle_prize_tiers.sql` | 新增 |
| `services/api/internal/models/raffle.go` | 修改（加 struct 與欄位） |
| `services/api/internal/services/raffle_service.go` | 修改（加方法、改 sendDiscordNotification） |
| `services/api/internal/services/raffle_service_test.go` | 修改（加測試） |
| `services/api/internal/handlers/raffle_handler.go` | 修改（加 handler） |
| `services/api/internal/handlers/raffle_handler_test.go` | 修改（加 handler 測試） |
| `services/api/internal/handlers/testutil_test.go` | 修改（加 SQLite CREATE TABLE） |
| `services/api/internal/router/router.go` | 修改（加路由） |
| `apps/dashboard/src/services/raffles.ts` | 修改（加型別與 API 函式） |
| `apps/dashboard/src/pages/RaffleDetailPage.tsx` | 修改（加獎項區塊） |

---

## Task 1: DB Migration + Model

**Files:**
- Create: `services/api/migrations/021_raffle_prize_tiers.sql`
- Modify: `services/api/internal/models/raffle.go`
- Modify: `services/api/internal/handlers/testutil_test.go`

- [ ] **Step 1: 建立 Atlas migration 檔案**

建立 `services/api/migrations/021_raffle_prize_tiers.sql`：

```sql
-- raffle_prize_tiers: sub-prize layers within a single raffle.
-- prize_tier_id on raffle_draws is nullable for backward compatibility.

CREATE TABLE raffle_prize_tiers (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    raffle_id        UUID         NOT NULL REFERENCES raffles(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    prize_description TEXT        NOT NULL DEFAULT '',
    winner_count     INT          NOT NULL CHECK (winner_count > 0),
    drawn_count      INT          NOT NULL DEFAULT 0,
    position         INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_raffle_prize_tiers_raffle_id
    ON raffle_prize_tiers (raffle_id, position);

ALTER TABLE raffle_draws
    ADD COLUMN prize_tier_id UUID REFERENCES raffle_prize_tiers(id) ON DELETE SET NULL;
```

- [ ] **Step 2: 更新 Atlas checksum**

```bash
cd services/api
atlas migrate hash --env dev
```

若本機沒有 Atlas dev DB，直接手動在 `atlas.sum` 末尾加入新 migration 的 hash（CI 會驗證）：

```bash
atlas migrate hash --dir "file://migrations"
```

- [ ] **Step 3: 在 `models/raffle.go` 加入 `RafflePrizeTier` struct**

在現有最後一個 struct 後面新增：

```go
// RafflePrizeTier represents one prize layer within a Raffle.
type RafflePrizeTier struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RaffleID         uuid.UUID `gorm:"type:uuid;not null;index"                       json:"raffle_id"`
	Name             string    `gorm:"type:varchar(255);not null"                     json:"name"`
	PrizeDescription string    `gorm:"type:text;not null;default:''"                  json:"prize_description"`
	WinnerCount      int       `gorm:"not null"                                       json:"winner_count"`
	DrawnCount       int       `gorm:"not null;default:0"                             json:"drawn_count"`
	Position         int       `gorm:"not null;default:0"                             json:"position"`
	CreatedAt        time.Time `                                                       json:"created_at"`
	UpdatedAt        time.Time `                                                       json:"updated_at"`
}

func (RafflePrizeTier) TableName() string { return "raffle_prize_tiers" }
```

- [ ] **Step 4: 在 `RaffleDraw` struct 加入 `PrizeTierID` 和 `PrizeTier`**

找到 `RaffleDraw` struct，在 `DrawnAt` 欄位之後加入：

```go
PrizeTierID *uuid.UUID       `gorm:"type:uuid"                              json:"prize_tier_id,omitempty"`
PrizeTier   *RafflePrizeTier `gorm:"foreignKey:PrizeTierID"                 json:"prize_tier,omitempty"`
```

- [ ] **Step 5: 在 `testutil_test.go` 的 `migrateTestDB` 加入新 SQLite 表**

找到 `raffle_draws` 的 `CREATE TABLE` 語句（約第 280 行），在其 `FOREIGN KEY` 行之前加入 `prize_tier_id` 欄位：

```go
// 找到這段，在 drawn_at 之後加一行：
`CREATE TABLE IF NOT EXISTS raffle_prize_tiers (
    id TEXT PRIMARY KEY,
    raffle_id TEXT NOT NULL REFERENCES raffles(id),
    name TEXT NOT NULL,
    prize_description TEXT NOT NULL DEFAULT '',
    winner_count INTEGER NOT NULL,
    drawn_count INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
`CREATE INDEX IF NOT EXISTS idx_raffle_prize_tiers_raffle_id ON raffle_prize_tiers (raffle_id, position)`,
```

並在現有 `raffle_draws` CREATE TABLE 的 `drawn_at DATETIME NOT NULL,` 之後加入：

```sql
prize_tier_id TEXT REFERENCES raffle_prize_tiers(id),
```

**注意：** `raffle_prize_tiers` 的 CREATE TABLE 必須放在 `raffle_draws` 的 CREATE TABLE **之前**，否則 FK 會失敗。

- [ ] **Step 6: 驗證 migration 語法正確**

```bash
cd services/api
docker compose run --no-deps --rm app go build ./...
```

預期：無錯誤輸出。

- [ ] **Step 7: Commit**

```bash
git add services/api/migrations/021_raffle_prize_tiers.sql \
        services/api/migrations/atlas.sum \
        services/api/internal/models/raffle.go \
        services/api/internal/handlers/testutil_test.go
git commit -m "feat: add raffle_prize_tiers schema and model

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Service — Prize Tier CRUD

**Files:**
- Modify: `services/api/internal/services/raffle_service.go`
- Modify: `services/api/internal/services/raffle_service_test.go`

- [ ] **Step 1: 寫 4 個失敗測試**

在 `raffle_service_test.go` 找到最後一個測試函式，在其後新增：

```go
func TestCreatePrizeTier_Success(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)

	raffle, err := svc.Create(streamer, "Test Raffle")
	if err != nil {
		t.Fatalf("create raffle: %v", err)
	}
	if err := db.Model(raffle).Update("status", "active").Error; err != nil {
		t.Fatalf("activate raffle: %v", err)
	}

	tier, err := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{
		Name:             "一等獎",
		PrizeDescription: "Switch 主機",
		WinnerCount:      1,
	})
	if err != nil {
		t.Fatalf("CreatePrizeTier: %v", err)
	}
	if tier.Name != "一等獎" || tier.WinnerCount != 1 || tier.DrawnCount != 0 {
		t.Fatalf("unexpected tier: %+v", tier)
	}
}

func TestCreatePrizeTier_InvalidWinnerCount(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")

	_, err := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{
		Name:        "無效獎",
		WinnerCount: 0,
	})
	if !errors.Is(err, services.ErrPrizeTierInvalidCount) {
		t.Fatalf("expected ErrPrizeTierInvalidCount, got %v", err)
	}
}

func TestListPrizeTiers_OrderedByPosition(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")

	svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "二等獎", WinnerCount: 3})
	svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	tiers, err := svc.ListPrizeTiers(raffle.ID, streamer)
	if err != nil {
		t.Fatalf("ListPrizeTiers: %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("want 2 tiers, got %d", len(tiers))
	}
	// position should be 1, 2 in order of creation
	if tiers[0].Position != 1 || tiers[1].Position != 2 {
		t.Fatalf("wrong order: %+v", tiers)
	}
}

func TestDeletePrizeTier_FailsIfDrawn(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")
	tier, _ := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	// Simulate drawn_count > 0
	db.Model(tier).UpdateColumn("drawn_count", 1)

	err := svc.DeletePrizeTier(raffle.ID, tier.ID, streamer)
	if !errors.Is(err, services.ErrPrizeTierHasDraws) {
		t.Fatalf("expected ErrPrizeTierHasDraws, got %v", err)
	}
}
```

- [ ] **Step 2: 執行測試，確認失敗**

```bash
cd services/api
docker compose run --no-deps --rm app go test ./internal/services/... -run "TestCreatePrizeTier|TestListPrizeTiers|TestDeletePrizeTier" -v
```

預期：`FAIL` — `CreatePrizeTier undefined`

- [ ] **Step 3: 在 `raffle_service.go` 加入 error 常數與 input types**

在現有 error 常數區塊（約第 27-42 行）末尾加入：

```go
ErrPrizeTierNotFound    = errors.New("prize tier not found")
ErrPrizeTierHasDraws    = errors.New("prize tier already has draws, cannot delete")
ErrPrizeTierExhausted   = errors.New("prize tier winner count reached")
ErrPrizeTierInvalidCount = errors.New("winner_count must be at least 1")
```

在 input types 區塊新增：

```go
type CreatePrizeTierInput struct {
	Name             string `json:"name"`
	PrizeDescription string `json:"prize_description"`
	WinnerCount      int    `json:"winner_count"`
}

type UpdatePrizeTierInput struct {
	Name             *string `json:"name,omitempty"`
	PrizeDescription *string `json:"prize_description,omitempty"`
	WinnerCount      *int    `json:"winner_count,omitempty"`
}
```

- [ ] **Step 4: 實作 4 個 service 方法**

在 `raffle_service.go` 末尾加入：

```go
// CreatePrizeTier adds a new prize tier to a raffle. Position is auto-assigned
// as max(position)+1 for the raffle.
func (s *RaffleService) CreatePrizeTier(raffleID, userID uuid.UUID, input CreatePrizeTierInput) (*models.RafflePrizeTier, error) {
	if input.WinnerCount < 1 {
		return nil, ErrPrizeTierInvalidCount
	}
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return nil, err
	}

	var maxPos int
	s.db.Model(&models.RafflePrizeTier{}).
		Where("raffle_id = ?", raffleID).
		Select("COALESCE(MAX(position), 0)").
		Scan(&maxPos)

	tier := &models.RafflePrizeTier{
		RaffleID:         raffleID,
		Name:             input.Name,
		PrizeDescription: input.PrizeDescription,
		WinnerCount:      input.WinnerCount,
		Position:         maxPos + 1,
	}
	if err := s.db.Create(tier).Error; err != nil {
		return nil, err
	}
	return tier, nil
}

// ListPrizeTiers returns all prize tiers for a raffle ordered by position.
func (s *RaffleService) ListPrizeTiers(raffleID, userID uuid.UUID) ([]models.RafflePrizeTier, error) {
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return nil, err
	}
	var tiers []models.RafflePrizeTier
	if err := s.db.
		Where("raffle_id = ?", raffleID).
		Order("position ASC").
		Find(&tiers).Error; err != nil {
		return nil, err
	}
	return tiers, nil
}

// UpdatePrizeTier updates mutable fields of a prize tier.
// winner_count cannot be set below drawn_count.
func (s *RaffleService) UpdatePrizeTier(raffleID, tierID, userID uuid.UUID, input UpdatePrizeTierInput) (*models.RafflePrizeTier, error) {
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return nil, err
	}
	var tier models.RafflePrizeTier
	if err := s.db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPrizeTierNotFound
		}
		return nil, err
	}
	if input.WinnerCount != nil {
		if *input.WinnerCount < 1 {
			return nil, ErrPrizeTierInvalidCount
		}
		if *input.WinnerCount < tier.DrawnCount {
			return nil, ErrPrizeTierInvalidCount
		}
		tier.WinnerCount = *input.WinnerCount
	}
	if input.Name != nil {
		tier.Name = *input.Name
	}
	if input.PrizeDescription != nil {
		tier.PrizeDescription = *input.PrizeDescription
	}
	if err := s.db.Save(&tier).Error; err != nil {
		return nil, err
	}
	return &tier, nil
}

// DeletePrizeTier removes a prize tier. Fails if any draws exist for this tier.
func (s *RaffleService) DeletePrizeTier(raffleID, tierID, userID uuid.UUID) error {
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return err
	}
	var tier models.RafflePrizeTier
	if err := s.db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPrizeTierNotFound
		}
		return err
	}
	if tier.DrawnCount > 0 {
		return ErrPrizeTierHasDraws
	}
	return s.db.Delete(&tier).Error
}
```

- [ ] **Step 5: 執行測試，確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run "TestCreatePrizeTier|TestListPrizeTiers|TestDeletePrizeTier" -v
```

預期：`PASS`

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/services/raffle_service.go \
        services/api/internal/services/raffle_service_test.go
git commit -m "feat: add prize tier CRUD service methods

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Service — DrawFromTier

**Files:**
- Modify: `services/api/internal/services/raffle_service.go`
- Modify: `services/api/internal/services/raffle_service_test.go`

- [ ] **Step 1: 寫失敗測試**

```go
func TestDrawFromTier_ExcludesWinnersFromOtherTiers(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)

	raffle, _ := svc.Create(streamer, "Multi-tier Raffle")
	// Seed 2 entries
	seedRaffleEntry(t, db, raffle.ID, "user_a")
	seedRaffleEntry(t, db, raffle.ID, "user_b")
	db.Model(raffle).Update("status", "active")

	tier1, _ := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})
	tier2, _ := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "二等獎", WinnerCount: 1})

	draw1, err := svc.DrawFromTier(raffle.ID, tier1.ID, streamer)
	if err != nil {
		t.Fatalf("DrawFromTier tier1: %v", err)
	}

	draw2, err := svc.DrawFromTier(raffle.ID, tier2.ID, streamer)
	if err != nil {
		t.Fatalf("DrawFromTier tier2: %v", err)
	}

	// Must be different winners
	if draw1.EntryID == draw2.EntryID {
		t.Fatal("tier2 must not pick the same entry as tier1")
	}

	// drawn_count should be updated
	var updated models.RafflePrizeTier
	db.First(&updated, tier1.ID)
	if updated.DrawnCount != 1 {
		t.Fatalf("expected drawn_count=1, got %d", updated.DrawnCount)
	}
}

func TestDrawFromTier_ExhaustedWhenTierFull(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)

	raffle, _ := svc.Create(streamer, "Single winner raffle")
	seedRaffleEntry(t, db, raffle.ID, "only_one")
	db.Model(raffle).Update("status", "active")
	tier, _ := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	svc.DrawFromTier(raffle.ID, tier.ID, streamer)

	_, err := svc.DrawFromTier(raffle.ID, tier.ID, streamer)
	if !errors.Is(err, services.ErrPrizeTierExhausted) {
		t.Fatalf("expected ErrPrizeTierExhausted, got %v", err)
	}
}

func TestDrawFromTier_PrizeTierIDSetOnDraw(t *testing.T) {
	db := newTestDB(t)
	svc := services.NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)

	raffle, _ := svc.Create(streamer, "Tier ID Test")
	seedRaffleEntry(t, db, raffle.ID, "winner")
	db.Model(raffle).Update("status", "active")
	tier, _ := svc.CreatePrizeTier(raffle.ID, streamer, services.CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	draw, _ := svc.DrawFromTier(raffle.ID, tier.ID, streamer)
	if draw.PrizeTierID == nil || *draw.PrizeTierID != tier.ID {
		t.Fatalf("expected PrizeTierID=%s, got %v", tier.ID, draw.PrizeTierID)
	}
}
```

`seedRaffleEntry` 是測試輔助函式，在測試檔加入：

```go
func seedRaffleEntry(t *testing.T, db *gorm.DB, raffleID uuid.UUID, twitchLogin string) {
	t.Helper()
	entry := models.RaffleEntry{
		RaffleID:    raffleID,
		TwitchLogin: twitchLogin,
		DisplayName: twitchLogin,
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seedRaffleEntry: %v", err)
	}
}
```

- [ ] **Step 2: 執行測試，確認失敗**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run "TestDrawFromTier" -v
```

預期：`FAIL` — `DrawFromTier undefined`

- [ ] **Step 3: 實作 `DrawFromTier`**

在 `raffle_service.go` 末尾加入：

```go
// DrawFromTier picks a random un-drawn entry for the given prize tier.
// Excludes all entries that have already won any tier within the same raffle.
func (s *RaffleService) DrawFromTier(raffleID, tierID, userID uuid.UUID) (*models.RaffleDraw, error) {
	raffle, err := s.GetByID(raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status != models.RaffleStatusActive {
		return nil, ErrRaffleNotActive
	}

	var tier models.RafflePrizeTier
	if err := s.db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPrizeTierNotFound
		}
		return nil, err
	}
	if tier.DrawnCount >= tier.WinnerCount {
		return nil, ErrPrizeTierExhausted
	}

	const drawMaxRetries = 5
	var result *models.RaffleDraw

	err = func() error {
		for attempt := 0; attempt < drawMaxRetries; attempt++ {
			var draw *models.RaffleDraw
			txErr := s.db.Transaction(func(tx *gorm.DB) error {
				// Exclude winners from ALL tiers of this raffle.
				var wonIDs []uuid.UUID
				tx.Model(&models.RaffleDraw{}).
					Where("raffle_id = ?", raffleID).
					Pluck("entry_id", &wonIDs)

				var entry models.RaffleEntry
				q := tx.Where("raffle_id = ?", raffleID)
				if len(wonIDs) > 0 {
					q = q.Where("id NOT IN ?", wonIDs)
				}
				if err := q.Order("RANDOM()").First(&entry).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return ErrRaffleExhausted
					}
					return err
				}

				rawToken, err := uuid.NewV7()
				if err != nil {
					return err
				}
				tierIDCopy := tierID
				d := &models.RaffleDraw{
					RaffleID:       raffleID,
					EntryID:        entry.ID,
					ClaimToken:     hashClaimToken(rawToken.String()),
					ClaimExpiresAt: time.Now().Add(claimTokenTTL),
					DrawnAt:        time.Now(),
					PrizeTierID:    &tierIDCopy,
				}
				if err := tx.Create(d).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.RafflePrizeTier{}).
					Where("id = ?", tierID).
					UpdateColumn("drawn_count", gorm.Expr("drawn_count + 1")).Error; err != nil {
					return err
				}
				d.ClaimTokenRaw = rawToken.String()
				d.Entry = entry
				d.PrizeTier = &tier
				draw = d
				return nil
			})
			if txErr == nil {
				result = draw
				return nil
			}
			if errors.Is(txErr, ErrRaffleExhausted) {
				return txErr
			}
			if errors.Is(txErr, gorm.ErrDuplicatedKey) {
				continue
			}
			return txErr
		}
		return ErrRaffleExhausted
	}()

	if err != nil {
		return nil, err
	}

	if s.mailer != nil {
		go func(d *models.RaffleDraw) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("raffle sendWinnerEmail panic (draw %s): %v", d.ID, r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendWinnerEmail(ctx, d)
		}(result)
	}
	if raffle.DiscordWebhookURL != nil {
		go func(d *models.RaffleDraw, webhookURL string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("raffle sendDiscordNotification panic (draw %s): %v", d.ID, r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendDiscordNotification(ctx, d, webhookURL)
		}(result, *raffle.DiscordWebhookURL)
	}
	return result, nil
}
```

**注意：** `ErrRaffleNotActive` 尚未定義，需在 error 常數區塊加入：

```go
ErrRaffleNotActive = errors.New("raffle is not active")
```

- [ ] **Step 4: 修改 `sendDiscordNotification` 加入獎項資訊**

找到現有的 `sendDiscordNotification` 函式，將 `content` 的 payload 改為：

```go
twitchLogin := draw.Entry.TwitchLogin
expiry := draw.ClaimExpiresAt.Format("2006-01-02 15:04 MST")

var content string
if draw.PrizeTier != nil {
    content = fmt.Sprintf(
        "🎉 [%s] 抽獎結果揭曉！中獎者：**%s**\n獎品：%s\n\n請中獎者留意 Tachigo 系統通知，或聯繫實況主確認領獎方式。\n領獎資格保留至：%s。",
        draw.PrizeTier.Name, twitchLogin, draw.PrizeTier.PrizeDescription, expiry,
    )
} else {
    content = fmt.Sprintf(
        "🎉 抽獎結果揭曉！中獎者：**%s**\n\n請中獎者留意 Tachigo 系統通知，或聯繫實況主確認領獎方式。\n領獎資格保留至：%s。",
        twitchLogin, expiry,
    )
}
payload := map[string]interface{}{"content": content}
```

刪除原本兩行重複設定 `payload["content"]` 的程式碼（第二行是多餘的）。

- [ ] **Step 5: 執行測試，確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run "TestDrawFromTier" -v
```

預期：`PASS`

- [ ] **Step 6: 執行全部 service 測試確認無回歸**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -v
```

預期：全部 `PASS`

- [ ] **Step 7: Commit**

```bash
git add services/api/internal/services/raffle_service.go \
        services/api/internal/services/raffle_service_test.go
git commit -m "feat: add DrawFromTier service method with cross-tier exclusion

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Handler + Router

**Files:**
- Modify: `services/api/internal/handlers/raffle_handler.go`
- Modify: `services/api/internal/handlers/raffle_handler_test.go`
- Modify: `services/api/internal/router/router.go`

- [ ] **Step 1: 寫失敗 handler 測試**

在 `raffle_handler_test.go` 末尾加入：

```go
func TestPrizeTierHandler_CreateAndList(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "host", "host@test.com", "pass1234")
	raffleID := env.createRaffle(t, token, "Tier Test Raffle")

	// Create tier
	body, _ := json.Marshal(map[string]interface{}{
		"name":              "一等獎",
		"prize_description": "Switch 主機",
		"winner_count":      1,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/prize-tiers",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create tier: want 201, got %d — %s", w.Code, w.Body.String())
	}

	// List tiers
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet,
		"/api/v1/dashboard/raffles/"+raffleID+"/prize-tiers", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	env.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list tiers: want 200, got %d", w2.Code)
	}
	var resp struct {
		Data struct {
			Tiers []map[string]interface{} `json:"tiers"`
		} `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&resp)
	if len(resp.Data.Tiers) != 1 {
		t.Fatalf("want 1 tier, got %d", len(resp.Data.Tiers))
	}
}

func TestPrizeTierHandler_DrawFromTier_ExcludesWinners(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "host2", "host2@test.com", "pass1234")
	raffleID := env.createRaffle(t, token, "Draw Tier Test")

	// Seed 2 entries via CSV
	env.uploadCSV(t, token, raffleID, "user_a,UserA\nuser_b,UserB\n")
	env.activateRaffle(t, token, raffleID)

	// Create 2 tiers
	tier1ID := env.createPrizeTier(t, token, raffleID, "一等獎", "Switch", 1)
	tier2ID := env.createPrizeTier(t, token, raffleID, "二等獎", "貼圖包", 1)

	// Draw tier 1
	w1 := env.drawFromTier(t, token, raffleID, tier1ID)
	if w1.Code != http.StatusCreated {
		t.Fatalf("draw tier1: want 201, got %d — %s", w1.Code, w1.Body.String())
	}

	// Draw tier 2 — should succeed with different winner
	w2 := env.drawFromTier(t, token, raffleID, tier2ID)
	if w2.Code != http.StatusCreated {
		t.Fatalf("draw tier2: want 201, got %d — %s", w2.Code, w2.Body.String())
	}

	var r1, r2 struct{ Data struct{ Draw struct{ EntryID string `json:"entry_id"` } `json:"draw"` } `json:"data"` }
	json.NewDecoder(w1.Body).Decode(&r1)
	json.NewDecoder(w2.Body).Decode(&r2)
	if r1.Data.Draw.EntryID == r2.Data.Draw.EntryID {
		t.Fatal("tier2 must not draw the same winner as tier1")
	}
}
```

在 `raffleTestEnv` 加入輔助方法（同檔案）：

```go
func (e *raffleTestEnv) activateRaffle(t *testing.T, token, raffleID string) {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles/"+raffleID+"/activate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	e.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("activate raffle: want 200, got %d", w.Code)
	}
}

func (e *raffleTestEnv) createPrizeTier(t *testing.T, token, raffleID, name, prize string, count int) string {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"name": name, "prize_description": prize, "winner_count": count,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles/"+raffleID+"/prize-tiers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	e.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createPrizeTier: want 201, got %d", w.Code)
	}
	var resp struct{ Data struct{ Tier struct{ ID string `json:"id"` } `json:"tier"` } `json:"data"` }
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Data.Tier.ID
}

func (e *raffleTestEnv) drawFromTier(t *testing.T, token, raffleID, tierID string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/prize-tiers/"+tierID+"/draws", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	e.router.ServeHTTP(w, req)
	return w
}
```

- [ ] **Step 2: 在 `newRaffleTestEnv` 的路由設定中加入新路由（測試用）**

找到 `newRaffleTestEnv` 函式中設定路由的部分，加入（現有 raffle 路由之後）：

```go
dashboard.POST("/raffles/:id/prize-tiers", middleware.RequireRole(models.RoleStreamer), raffleH.CreatePrizeTier)
dashboard.GET("/raffles/:id/prize-tiers", middleware.RequireRole(models.RoleStreamer), raffleH.ListPrizeTiers)
dashboard.PATCH("/raffles/:id/prize-tiers/:tier_id", middleware.RequireRole(models.RoleStreamer), raffleH.UpdatePrizeTier)
dashboard.DELETE("/raffles/:id/prize-tiers/:tier_id", middleware.RequireRole(models.RoleStreamer), raffleH.DeletePrizeTier)
dashboard.POST("/raffles/:id/prize-tiers/:tier_id/draws", middleware.RequireRole(models.RoleStreamer), raffleH.DrawFromTier)
```

- [ ] **Step 3: 執行測試，確認失敗**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run "TestPrizeTierHandler" -v
```

預期：`FAIL` — `raffleH.CreatePrizeTier undefined`

- [ ] **Step 4: 實作 5 個 handler**

在 `raffle_handler.go` 末尾加入：

```go
// CreatePrizeTier godoc
// @Summary      Create a prize tier for a raffle
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Raffle ID"
// @Param        body body      object  true  "Prize tier"
// @Success      201  {object}  Response
// @Router       /dashboard/raffles/{id}/prize-tiers [post]
func (h *RaffleHandler) CreatePrizeTier(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}
	var body services.CreatePrizeTierInput
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	tier, err := h.raffleSvc.CreatePrizeTier(raffleID, userID, body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound):
			notFound(c, "raffle not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierInvalidCount):
			badRequest(c, "winner_count must be at least 1")
		default:
			log.Printf("CreatePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	created(c, gin.H{"tier": tier})
}

// ListPrizeTiers godoc
// @Summary      List prize tiers for a raffle
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Raffle ID"
// @Success      200  {object}  Response
// @Router       /dashboard/raffles/{id}/prize-tiers [get]
func (h *RaffleHandler) ListPrizeTiers(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}
	tiers, err := h.raffleSvc.ListPrizeTiers(raffleID, userID)
	if err != nil {
		if errors.Is(err, services.ErrRaffleNotFound) {
			notFound(c, "raffle not found")
			return
		}
		log.Printf("ListPrizeTiers: %v", err)
		internal(c)
		return
	}
	ok(c, gin.H{"tiers": tiers})
}

// UpdatePrizeTier godoc
// @Summary      Update a prize tier
// @Tags         raffles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path  string  true  "Raffle ID"
// @Param        tier_id path  string  true  "Tier ID"
// @Success      200  {object}  Response
// @Router       /dashboard/raffles/{id}/prize-tiers/{tier_id} [patch]
func (h *RaffleHandler) UpdatePrizeTier(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	var body services.UpdatePrizeTierInput
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}
	tier, err := h.raffleSvc.UpdatePrizeTier(raffleID, tierID, userID, body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierInvalidCount):
			badRequest(c, "invalid winner_count")
		default:
			log.Printf("UpdatePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{"tier": tier})
}

// DeletePrizeTier godoc
// @Summary      Delete a prize tier (only if no draws exist)
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id      path  string  true  "Raffle ID"
// @Param        tier_id path  string  true  "Tier ID"
// @Success      200  {object}  Response
// @Router       /dashboard/raffles/{id}/prize-tiers/{tier_id} [delete]
func (h *RaffleHandler) DeletePrizeTier(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	if err := h.raffleSvc.DeletePrizeTier(raffleID, tierID, userID); err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrPrizeTierHasDraws):
			conflict(c, "cannot delete a tier with existing draws")
		default:
			log.Printf("DeletePrizeTier: %v", err)
			internal(c)
		}
		return
	}
	ok(c, gin.H{})
}

// DrawFromTier godoc
// @Summary      Draw one winner from a prize tier
// @Tags         raffles
// @Security     BearerAuth
// @Produce      json
// @Param        id      path  string  true  "Raffle ID"
// @Param        tier_id path  string  true  "Tier ID"
// @Success      201  {object}  Response
// @Router       /dashboard/raffles/{id}/prize-tiers/{tier_id}/draws [post]
func (h *RaffleHandler) DrawFromTier(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}
	raffleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid raffle id")
		return
	}
	tierID, err := uuid.Parse(c.Param("tier_id"))
	if err != nil {
		badRequest(c, "invalid tier id")
		return
	}
	draw, err := h.raffleSvc.DrawFromTier(raffleID, tierID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRaffleNotFound), errors.Is(err, services.ErrPrizeTierNotFound):
			notFound(c, "not found")
		case errors.Is(err, services.ErrRaffleForbidden):
			c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
		case errors.Is(err, services.ErrRaffleNotActive):
			conflict(c, "raffle is not active")
		case errors.Is(err, services.ErrPrizeTierExhausted), errors.Is(err, services.ErrRaffleExhausted):
			conflict(c, "no more entries available")
		default:
			log.Printf("DrawFromTier: %v", err)
			internal(c)
		}
		return
	}
	created(c, gin.H{"draw": draw})
}
```

- [ ] **Step 5: 在 `router.go` 加入新路由**

找到現有 raffle 路由區塊（`dashboard.POST("/raffles/:id/snapshot", ...)` 之後），加入：

```go
dashboard.POST("/raffles/:id/prize-tiers",
    middleware.RequireRole(models.RoleStreamer),
    raffleH.CreatePrizeTier)
dashboard.GET("/raffles/:id/prize-tiers",
    middleware.RequireRole(models.RoleStreamer),
    raffleH.ListPrizeTiers)
dashboard.PATCH("/raffles/:id/prize-tiers/:tier_id",
    middleware.RequireRole(models.RoleStreamer),
    raffleH.UpdatePrizeTier)
dashboard.DELETE("/raffles/:id/prize-tiers/:tier_id",
    middleware.RequireRole(models.RoleStreamer),
    raffleH.DeletePrizeTier)
dashboard.POST("/raffles/:id/prize-tiers/:tier_id/draws",
    middleware.RequireRole(models.RoleStreamer),
    raffleH.DrawFromTier)
```

- [ ] **Step 6: 執行 handler 測試，確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run "TestPrizeTierHandler" -v
```

預期：`PASS`

- [ ] **Step 7: 執行全部後端測試確認無回歸**

```bash
docker compose run --no-deps --rm app go test ./...
```

預期：全部 `PASS`

- [ ] **Step 8: Commit**

```bash
git add services/api/internal/handlers/raffle_handler.go \
        services/api/internal/handlers/raffle_handler_test.go \
        services/api/internal/router/router.go
git commit -m "feat: add prize tier handlers and routes

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Frontend — API Client

**Files:**
- Modify: `apps/dashboard/src/services/raffles.ts`

- [ ] **Step 1: 加入型別定義**

在 `raffles.ts` 現有型別定義末尾加入：

```typescript
export interface RafflePrizeTier {
  id: string
  raffle_id: string
  name: string
  prize_description: string
  winner_count: number
  drawn_count: number
  position: number
  created_at: string
  updated_at: string
}

export interface CreatePrizeTierInput {
  name: string
  prize_description: string
  winner_count: number
}

export interface UpdatePrizeTierInput {
  name?: string
  prize_description?: string
  winner_count?: number
}
```

- [ ] **Step 2: 加入 5 個 API 函式**

在現有 API 函式末尾加入：

```typescript
export async function listPrizeTiers(raffleId: string): Promise<RafflePrizeTier[]> {
  const res = await apiClient.get<ApiResponse<{ tiers: RafflePrizeTier[] }>>(
    `/dashboard/raffles/${raffleId}/prize-tiers`
  )
  return res.data.data.tiers
}

export async function createPrizeTier(
  raffleId: string,
  input: CreatePrizeTierInput
): Promise<RafflePrizeTier> {
  const res = await apiClient.post<ApiResponse<{ tier: RafflePrizeTier }>>(
    `/dashboard/raffles/${raffleId}/prize-tiers`,
    input
  )
  return res.data.data.tier
}

export async function updatePrizeTier(
  raffleId: string,
  tierId: string,
  input: UpdatePrizeTierInput
): Promise<RafflePrizeTier> {
  const res = await apiClient.patch<ApiResponse<{ tier: RafflePrizeTier }>>(
    `/dashboard/raffles/${raffleId}/prize-tiers/${tierId}`,
    input
  )
  return res.data.data.tier
}

export async function deletePrizeTier(raffleId: string, tierId: string): Promise<void> {
  await apiClient.delete(`/dashboard/raffles/${raffleId}/prize-tiers/${tierId}`)
}

export async function drawFromTier(raffleId: string, tierId: string): Promise<RaffleDraw> {
  const res = await apiClient.post<ApiResponse<{ draw: RaffleDraw }>>(
    `/dashboard/raffles/${raffleId}/prize-tiers/${tierId}/draws`
  )
  return res.data.data.draw
}
```

**注意：** `apiClient` 是現有的 axios instance，路徑是相對於 `/api/v1` base URL。請確認現有檔案裡的 base URL 設定，確保路徑一致。

- [ ] **Step 3: 確認 TypeScript 編譯無錯誤**

```bash
cd apps/dashboard
npm run type-check
```

預期：無型別錯誤。

- [ ] **Step 4: Commit**

```bash
git add apps/dashboard/src/services/raffles.ts
git commit -m "feat: add prize tier API client functions

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Frontend — Dashboard UI

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1: 加入 prize tier state 和 fetch 邏輯**

在 `RaffleDetailPage` 函式開頭的 state 宣告區，加入：

```typescript
const [tiers, setTiers] = useState<RafflePrizeTier[]>([])
const [tierDrawing, setTierDrawing] = useState<Record<string, boolean>>({})
const [tierExhausted, setTierExhausted] = useState<Record<string, boolean>>({})
const [showAddTier, setShowAddTier] = useState(false)
const [newTier, setNewTier] = useState({ name: '', prize_description: '', winner_count: 1 })
const [addingTier, setAddingTier] = useState(false)
```

加入 fetch tiers 的函式（放在 `fetchDraws` 之後）：

```typescript
const fetchTiers = useCallback(async () => {
  if (!raffleId) return
  try {
    const result = await listPrizeTiers(raffleId)
    setTiers(result)
  } catch {
    setTiers([])
  }
}, [raffleId])
```

在現有 `useEffect` 中，於 `fetchDraws()` 後加入 `fetchTiers()`：

```typescript
void fetchDraws()
void fetchTiers()
```

- [ ] **Step 2: 加入新增獎項的 handler**

```typescript
async function handleAddTier() {
  if (!raffleId || addingTier) return
  if (!newTier.name.trim() || newTier.winner_count < 1) return
  setAddingTier(true)
  try {
    await createPrizeTier(raffleId, newTier)
    await fetchTiers()
    setNewTier({ name: '', prize_description: '', winner_count: 1 })
    setShowAddTier(false)
  } finally {
    setAddingTier(false)
  }
}

async function handleDrawFromTier(tierId: string) {
  if (!raffleId || tierDrawing[tierId]) return
  setTierDrawing(prev => ({ ...prev, [tierId]: true }))
  try {
    await drawFromTier(raffleId, tierId)
    await Promise.all([fetchTiers(), fetchDraws()])
    setTierExhausted(prev => ({ ...prev, [tierId]: false }))
  } catch (error: unknown) {
    if (error && typeof error === 'object' && 'response' in error) {
      const res = (error as { response?: { status?: number } }).response
      if (res?.status === 409) {
        setTierExhausted(prev => ({ ...prev, [tierId]: true }))
      }
    }
  } finally {
    setTierDrawing(prev => ({ ...prev, [tierId]: false }))
  }
}

async function handleDeleteTier(tierId: string) {
  if (!raffleId) return
  try {
    await deletePrizeTier(raffleId, tierId)
    await fetchTiers()
  } catch {
    // tier has draws — silently ignore, UI should hide delete button anyway
  }
}
```

- [ ] **Step 3: 在 JSX 加入獎項區塊**

找到頁面中「中獎記錄」區塊的上方，插入獎項區塊：

```tsx
{/* Prize Tiers Section */}
{effectiveStatus !== 'draft' && (
  <section style={{ marginBottom: '2rem' }}>
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
      <h3 style={{ margin: 0 }}>獎項</h3>
      {effectiveStatus === 'active' && (
        <button onClick={() => setShowAddTier(v => !v)}>+ 新增獎項</button>
      )}
    </div>

    {showAddTier && (
      <div style={{ border: '1px solid #ccc', padding: '1rem', borderRadius: 8, marginBottom: '1rem' }}>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <div>
            <label>獎項名稱</label>
            <input
              value={newTier.name}
              onChange={e => setNewTier(p => ({ ...p, name: e.target.value }))}
              placeholder="例：一等獎"
            />
          </div>
          <div>
            <label>獎品描述</label>
            <input
              value={newTier.prize_description}
              onChange={e => setNewTier(p => ({ ...p, prize_description: e.target.value }))}
              placeholder="例：Switch 主機"
            />
          </div>
          <div>
            <label>抽幾人</label>
            <input
              type="number"
              min={1}
              value={newTier.winner_count}
              onChange={e => setNewTier(p => ({ ...p, winner_count: Number(e.target.value) }))}
            />
          </div>
          <button onClick={handleAddTier} disabled={addingTier}>
            {addingTier ? '新增中...' : '確認新增'}
          </button>
          <button onClick={() => setShowAddTier(false)}>取消</button>
        </div>
      </div>
    )}

    {tiers.length === 0 && !showAddTier && (
      <p style={{ color: '#888', fontSize: '0.9rem' }}>尚未設定獎項。</p>
    )}

    {tiers.map(tier => {
      const isDone = tier.drawn_count >= tier.winner_count
      const isDrawing = tierDrawing[tier.id]
      const isExhausted = tierExhausted[tier.id]
      return (
        <div
          key={tier.id}
          style={{
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            padding: '0.75rem 1rem', border: '1px solid #eee', borderRadius: 8,
            marginBottom: '0.5rem', background: isDone ? '#f5f5f5' : '#fff',
          }}
        >
          <div>
            <strong>{tier.name}</strong>
            {tier.prize_description && (
              <span style={{ color: '#666', marginLeft: '0.5rem' }}>｜{tier.prize_description}</span>
            )}
            <span style={{ marginLeft: '1rem', color: isDone ? '#888' : '#333' }}>
              已抽 {tier.drawn_count} / {tier.winner_count} 人
            </span>
          </div>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            {effectiveStatus === 'active' && (
              <>
                <button
                  onClick={() => handleDrawFromTier(tier.id)}
                  disabled={isDone || isDrawing}
                >
                  {isDrawing ? '抽取中...' : isDone ? '已完成' : '抽一人'}
                </button>
                {tier.drawn_count === 0 && (
                  <button onClick={() => handleDeleteTier(tier.id)} style={{ color: '#c00' }}>
                    刪除
                  </button>
                )}
              </>
            )}
          </div>
        </div>
      )
    })}
    {tierExhausted && Object.values(tierExhausted).some(Boolean) && (
      <p style={{ color: '#c00', fontSize: '0.875rem' }}>名單已耗盡，無法繼續抽獎。</p>
    )}
  </section>
)}
```

- [ ] **Step 4: 在中獎記錄區塊加入獎項欄位**

找到顯示每筆 draw 的地方，在 `draw.entry.twitch_login`（或類似欄位）附近加入：

```tsx
{draw.prize_tier && (
  <span style={{ background: '#e8f0fe', borderRadius: 4, padding: '0 6px', fontSize: '0.8rem', marginRight: '0.5rem' }}>
    {draw.prize_tier.name}
  </span>
)}
```

- [ ] **Step 5: 更新 import**

在 `RaffleDetailPage.tsx` 頂部的 import from `../services/raffles`，加入新函式：

```typescript
import {
  // ...現有 imports...
  listPrizeTiers,
  createPrizeTier,
  deletePrizeTier,
  drawFromTier,
  type RafflePrizeTier,
} from '../services/raffles'
```

- [ ] **Step 6: 確認 TypeScript 編譯**

```bash
cd apps/dashboard
npm run type-check
```

預期：無錯誤。

- [ ] **Step 7: 啟動 dev server 手動驗證**

```bash
make dev
```

開啟 Dashboard，進入任一 active 狀態的抽獎詳情頁，確認：
- 「獎項」區塊出現
- 可新增獎項
- 「抽一人」按鈕可運作
- 抽完後 `已抽 X / Y 人` 更新
- 中獎記錄列出獎項名稱

- [ ] **Step 8: Commit**

```bash
git add apps/dashboard/src/pages/RaffleDetailPage.tsx
git commit -m "feat: add prize tier UI to RaffleDetailPage

refs #234

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```
