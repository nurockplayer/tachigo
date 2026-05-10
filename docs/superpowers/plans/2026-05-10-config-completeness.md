# Config 完整化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 將 `LOG_LEVEL`、`ENABLE_SWAGGER`、`ENABLE_AUTOMIGRATE`、`ENABLE_SCHEDULER`、`ALLOWED_ORIGINS`、`GIN_MODE`、`TRUSTED_PROXIES` 歸入 `ServerConfig`，並讓 main.go / router.go 從 config 讀取，消除散落的 `os.Getenv` 直讀。

**Architecture:** 在 `config.go` 新增兩個 helper（`getBoolEnv`、`getCommaEnv`）和七個 `ServerConfig` 欄位；`Load()` 依 `appEnv` 決定 `EnableSwagger` / `GinMode` 的預設值。`router.New()` 從傳入的 `InternalRouterConfig.Config` 取 GinMode / TrustedProxies / EnableSwagger 並套用。`main.go` 移除手動 `os.Getenv("ALLOWED_ORIGINS")` 段落，改傳 `cfg.Server.AllowedOrigins`。`ENABLE_AUTOMIGRATE` 與 `ENABLE_SCHEDULER` 本票只入 config struct，不改 main.go 的呼叫邏輯。

**Tech Stack:** Go 1.22+、gin、swaggo/gin-swagger、strconv、strings

**Issue:** #569

---

## 前置確認

本 branch 應從 `develop` 拉出：

```bash
git checkout develop && git pull
git checkout -b feat/config-completeness
```

所有測試指令：

```bash
# 無需 docker 的 config unit tests
cd services/api && go test ./internal/config/...

# 完整 build 驗證
docker compose run --no-deps --rm app go build ./...

# 完整測試
docker compose run --no-deps --rm app go test ./...
```

---

## 檔案地圖

| 動作 | 路徑 |
|---|---|
| Modify | `services/api/internal/config/config.go` |
| Modify | `services/api/internal/config/config_test.go` |
| Modify | `services/api/internal/router/router.go` |
| Modify | `services/api/cmd/server/main.go` |
| Modify | `services/api/.env.example` |

---

## Task 1：新增 helper 函式並補測試（TDD）

**Files:**
- Modify: `services/api/internal/config/config.go`
- Modify: `services/api/internal/config/config_test.go`

### Step 1-1：在 config_test.go 新增 helper 測試（先寫，確認紅燈）

在 `config_test.go` 末尾加入：

```go
func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		fallback bool
		want     bool
	}{
		{"uses fallback when unset", "", true, true},
		{"uses fallback when unset false", "", false, false},
		{"parses true", "true", false, true},
		{"parses 1", "1", false, true},
		{"parses false", "false", true, false},
		{"parses 0", "0", true, false},
		{"falls back on invalid", "yes", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL_ENV", tc.envValue)
			got := getBoolEnv("TEST_BOOL_ENV", tc.fallback)
			if got != tc.want {
				t.Fatalf("getBoolEnv(%q, %v) = %v, want %v", tc.envValue, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestGetCommaEnv(t *testing.T) {
	fallback := []string{"http://localhost:3000", "http://localhost:5173"}

	t.Run("uses fallback when unset", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != len(fallback) || got[0] != fallback[0] {
			t.Fatalf("expected fallback %v, got %v", fallback, got)
		}
	})

	t.Run("splits single value", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "https://example.com")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 1 || got[0] != "https://example.com" {
			t.Fatalf("expected [https://example.com], got %v", got)
		}
	})

	t.Run("splits multiple values", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "https://a.com,https://b.com")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 2 || got[0] != "https://a.com" || got[1] != "https://b.com" {
			t.Fatalf("expected [https://a.com https://b.com], got %v", got)
		}
	})

	t.Run("trims whitespace around tokens", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "10.0.0.1, 10.0.0.2 , 10.0.0.3")
		got := getCommaEnv("TEST_COMMA_ENV", fallback)
		if len(got) != 3 || got[0] != "10.0.0.1" || got[1] != "10.0.0.2" || got[2] != "10.0.0.3" {
			t.Fatalf("expected trimmed tokens, got %v", got)
		}
	})

	t.Run("returns nil fallback when unset and fallback is nil", func(t *testing.T) {
		t.Setenv("TEST_COMMA_ENV", "")
		got := getCommaEnv("TEST_COMMA_ENV", nil)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}
```

### Step 1-2：確認測試紅燈

```bash
cd services/api && go test ./internal/config/... -run "TestGetBoolEnv|TestGetCommaEnv" -v
```

預期：`FAIL` — `getBoolEnv` / `getCommaEnv` undefined。

### Step 1-3：在 config.go 新增 helper 函式及 `strings` import

在 `config.go` 的 import block 加入 `"strings"`，並在檔案末尾（`validateJWTSecret` 之後）加入：

```go
func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getCommaEnv(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
```

### Step 1-4：確認測試綠燈

```bash
cd services/api && go test ./internal/config/... -run "TestGetBoolEnv|TestGetCommaEnv" -v
```

預期：`PASS`。

### Step 1-5：Commit

```bash
git add services/api/internal/config/config.go services/api/internal/config/config_test.go
git commit -m "feat: add getBoolEnv and getCommaEnv helpers to config

refs #569"
```

---

## Task 2：擴充 ServerConfig struct 與 Load()（TDD）

**Files:**
- Modify: `services/api/internal/config/config.go`
- Modify: `services/api/internal/config/config_test.go`

### Step 2-1：在 config_test.go 新增新欄位測試（先寫，確認紅燈）

在 `config_test.go` 末尾加入：

```go
func TestLoad_ServerConfig_Defaults(t *testing.T) {
	// 清除所有相關環境變數，驗證 development 預設值
	envVars := []string{
		"APP_ENV", "LOG_LEVEL", "ENABLE_SWAGGER",
		"ENABLE_AUTOMIGRATE", "ENABLE_SCHEDULER",
		"ALLOWED_ORIGINS", "GIN_MODE", "TRUSTED_PROXIES",
	}
	for _, k := range envVars {
		t.Setenv(k, "")
	}

	cfg := Load()

	if cfg.Server.LogLevel != "info" {
		t.Errorf("LogLevel: want %q, got %q", "info", cfg.Server.LogLevel)
	}
	// APP_ENV 未設定 → development（EnvSet=false）→ EnableSwagger 預設 true
	if !cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want true when APP_ENV unset, got false")
	}
	if !cfg.Server.EnableAutoMigrate {
		t.Errorf("EnableAutoMigrate: want true, got false")
	}
	if !cfg.Server.EnableScheduler {
		t.Errorf("EnableScheduler: want true, got false")
	}
	// ALLOWED_ORIGINS 未設定 → 開發預設
	if len(cfg.Server.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins: want 2 defaults, got %v", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins[0]: want http://localhost:3000, got %q", cfg.Server.AllowedOrigins[0])
	}
	// APP_ENV 未設定 → GinMode 預設 "debug"
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode: want %q, got %q", "debug", cfg.Server.GinMode)
	}
	// TRUSTED_PROXIES 未設定 → nil
	if cfg.Server.TrustedProxies != nil {
		t.Errorf("TrustedProxies: want nil, got %v", cfg.Server.TrustedProxies)
	}
}

func TestLoad_ServerConfig_ProductionDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("ENABLE_SWAGGER", "")
	t.Setenv("GIN_MODE", "")

	cfg := Load()

	// production 且 ENABLE_SWAGGER 未設定 → false
	if cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want false in production, got true")
	}
	// production 且 GIN_MODE 未設定 → "release"
	if cfg.Server.GinMode != "release" {
		t.Errorf("GinMode: want %q in production, got %q", "release", cfg.Server.GinMode)
	}
}

func TestLoad_ServerConfig_EnvOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("ENABLE_SWAGGER", "true")
	t.Setenv("ENABLE_AUTOMIGRATE", "false")
	t.Setenv("ENABLE_SCHEDULER", "false")
	t.Setenv("ALLOWED_ORIGINS", "https://app.tachigo.io,https://admin.tachigo.io")
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1,10.0.0.2")

	cfg := Load()

	if cfg.Server.LogLevel != "warn" {
		t.Errorf("LogLevel: want %q, got %q", "warn", cfg.Server.LogLevel)
	}
	if !cfg.Server.EnableSwagger {
		t.Errorf("EnableSwagger: want true (overridden), got false")
	}
	if cfg.Server.EnableAutoMigrate {
		t.Errorf("EnableAutoMigrate: want false (overridden), got true")
	}
	if cfg.Server.EnableScheduler {
		t.Errorf("EnableScheduler: want false (overridden), got true")
	}
	if len(cfg.Server.AllowedOrigins) != 2 || cfg.Server.AllowedOrigins[1] != "https://admin.tachigo.io" {
		t.Errorf("AllowedOrigins: got %v", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode: want %q, got %q", "debug", cfg.Server.GinMode)
	}
	if len(cfg.Server.TrustedProxies) != 2 || cfg.Server.TrustedProxies[0] != "10.0.0.1" {
		t.Errorf("TrustedProxies: got %v", cfg.Server.TrustedProxies)
	}
}
```

### Step 2-2：確認測試紅燈

```bash
cd services/api && go test ./internal/config/... -run "TestLoad_ServerConfig" -v
```

預期：`FAIL` — `ServerConfig` 缺少新欄位。

### Step 2-3：更新 config.go — ServerConfig struct

將 `ServerConfig` 替換為：

```go
type ServerConfig struct {
	Port              string
	Env               string
	EnvSet            bool
	LogLevel          string   // LOG_LEVEL, 預設 "info"
	EnableSwagger     bool     // ENABLE_SWAGGER, dev=true / prod=false
	EnableAutoMigrate bool     // ENABLE_AUTOMIGRATE, 預設 true
	EnableScheduler   bool     // ENABLE_SCHEDULER, 預設 true
	AllowedOrigins    []string // ALLOWED_ORIGINS comma-split
	GinMode           string   // GIN_MODE, dev="debug" / prod="release"
	TrustedProxies    []string // TRUSTED_PROXIES comma-split, nil=信任所有
}
```

### Step 2-4：更新 config.go — Load()

在 `Load()` 函式中，於現有 `appEnv, appEnvSet := getEnvWithPresence(...)` 之後，加入以下計算（在 `return &Config{...}` 之前）：

```go
isProduction := appEnvSet && appEnv == "production"

defaultEnableSwagger := !isProduction
defaultGinMode := "debug"
if isProduction {
    defaultGinMode = "release"
}
defaultAllowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
```

然後將 `Server: ServerConfig{...}` 更新為：

```go
Server: ServerConfig{
    Port:              getEnv("PORT", "8080"),
    Env:               appEnv,
    EnvSet:            appEnvSet,
    LogLevel:          getEnv("LOG_LEVEL", "info"),
    EnableSwagger:     getBoolEnv("ENABLE_SWAGGER", defaultEnableSwagger),
    EnableAutoMigrate: getBoolEnv("ENABLE_AUTOMIGRATE", true),
    EnableScheduler:   getBoolEnv("ENABLE_SCHEDULER", true),
    AllowedOrigins:    getCommaEnv("ALLOWED_ORIGINS", defaultAllowedOrigins),
    GinMode:           getEnv("GIN_MODE", defaultGinMode),
    TrustedProxies:    getCommaEnv("TRUSTED_PROXIES", nil),
},
```

### Step 2-5：確認測試綠燈

```bash
cd services/api && go test ./internal/config/... -v
```

預期：所有測試 `PASS`。

### Step 2-6：Commit

```bash
git add services/api/internal/config/config.go services/api/internal/config/config_test.go
git commit -m "feat: extend ServerConfig with 7 new fields and update Load()

refs #569"
```

---

## Task 3：更新 router.go — gin.SetMode / SetTrustedProxies / EnableSwagger

**Files:**
- Modify: `services/api/internal/router/router.go`

### Step 3-1：更新 router.New()

在 `New()` 函式的最頂端（現有 `r := gin.New()` 之前），提前解析 `cfg` 並呼叫 `gin.SetMode`：

**目前 router.go 的 `New()` 函式開頭：**

```go
func New(
    // ... params ...
) *gin.Engine {
    r := gin.New()
    r.Use(gin.Logger(), gin.Recovery())
    r.Use(middleware.CORS(allowedOrigins))

    var cfg *config.Config
    if len(internalRouterConfig) > 0 {
        cfg = internalRouterConfig[0].Config
    }
```

**修改後：**

```go
func New(
    // ... params ...（不變）
) *gin.Engine {
    var cfg *config.Config
    if len(internalRouterConfig) > 0 {
        cfg = internalRouterConfig[0].Config
    }

    if cfg != nil && cfg.Server.GinMode != "" {
        gin.SetMode(cfg.Server.GinMode)
    }

    r := gin.New()
    r.Use(gin.Logger(), gin.Recovery())
    r.Use(middleware.CORS(allowedOrigins))

    if cfg != nil {
        if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
            // TrustedProxies 設定失敗不應讓 server 無法啟動，但必須明確警告
            // cfg.Server.TrustedProxies nil = 信任所有（gin default）
            log.Printf("warning: SetTrustedProxies: %v", err)
        }
    }
```

### Step 3-2：讓 swagger 路由受 EnableSwagger 控制

找到目前：

```go
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

替換為：

```go
enableSwagger := cfg == nil || cfg.Server.EnableSwagger
if enableSwagger {
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

> 當 `cfg == nil`（例如測試情境未傳入 config）時，預設維持原本行為（顯示 swagger）。

### Step 3-3：確認 build 通過

```bash
docker compose run --no-deps --rm app go build ./...
```

預期：無錯誤。

### Step 3-4：確認完整測試無 regression

```bash
docker compose run --no-deps --rm app go test ./...
```

預期：全部 `PASS`。

### Step 3-5：Commit

```bash
git add services/api/internal/router/router.go
git commit -m "feat: apply GinMode, TrustedProxies, EnableSwagger from config in router

refs #569"
```

---

## Task 4：更新 main.go — 移除 os.Getenv("ALLOWED_ORIGINS") 直讀

**Files:**
- Modify: `services/api/cmd/server/main.go`

### Step 4-1：移除手動 ALLOWED_ORIGINS 段落

找到（main.go L161-168）：

```go
// CORS origins from env, default to localhost for dev
originsEnv := os.Getenv("ALLOWED_ORIGINS")
allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
if originsEnv != "" {
    allowedOrigins = strings.Split(originsEnv, ",")
}

r := router.New(
    // ...
    allowedOrigins,
```

替換為（移除手動解析，直接傳 cfg.Server.AllowedOrigins）：

```go
r := router.New(
    // ...
    cfg.Server.AllowedOrigins,
```

### Step 4-2：移除 `"strings"` import（`"os"` 需確認是否仍有其他用途）

移除 import block 中的 `"strings"`。

確認 `"os"` 是否仍有其他用途：

```bash
grep -n '"os"' services/api/cmd/server/main.go
grep -n 'os\.' services/api/cmd/server/main.go
```

若 `os.Getenv("ALLOWED_ORIGINS")` 是 main.go 中唯一的 `os.` 用途，也一併移除 `"os"`。

### Step 4-3：確認 build 通過

```bash
docker compose run --no-deps --rm app go build ./...
```

預期：無錯誤。確認 `os.Getenv("ALLOWED_ORIGINS")` 不再出現：

```bash
grep "ALLOWED_ORIGINS" services/api/cmd/server/main.go
```

預期：無輸出。

### Step 4-4：確認完整測試無 regression

```bash
docker compose run --no-deps --rm app go test ./...
```

預期：全部 `PASS`。

### Step 4-5：Commit

```bash
git add services/api/cmd/server/main.go
git commit -m "refactor: use cfg.Server.AllowedOrigins, remove os.Getenv ALLOWED_ORIGINS

refs #569"
```

---

## Task 5：更新 .env.example

**Files:**
- Modify: `services/api/.env.example`

### Step 5-1：在 CORS section 後補齊新欄位

找到現有的 CORS section：

```
# ── CORS ──────────────────────────────────────────────────────────────────────
# Comma-separated list of allowed origins
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://localhost:5174
```

在 CORS section **之前**，補齊 Server 相關欄位（選擇在 `PORT` / `APP_ENV` 附近的適當位置）：

```
# ── Server ────────────────────────────────────────────────────────────────────
# Log level: debug | info | warn | error (default: info)
LOG_LEVEL=info

# Gin mode: debug | release | test (default: debug in dev, release in prod)
GIN_MODE=debug

# Enable Swagger UI (default: true in dev, false in prod)
ENABLE_SWAGGER=true

# Enable GORM AutoMigrate on startup (default: true)
# NOTE: not yet wired — field exists in config but main.go always runs AutoMigrate (scope: A5-2)
ENABLE_AUTOMIGRATE=true

# Enable background raffle scheduler (default: true)
# NOTE: not yet wired — field exists in config but main.go always starts scheduler (scope: A5-2)
ENABLE_SCHEDULER=true

# Comma-separated trusted reverse proxy IPs (empty = trust all)
TRUSTED_PROXIES=
```

> 找到合適的位置插入（靠近 PORT / APP_ENV 的 section），不要隨意置於檔案末尾。

### Step 5-2：Commit

```bash
git add services/api/.env.example
git commit -m "docs: add LOG_LEVEL, GIN_MODE, ENABLE_* env vars to .env.example

refs #569"
```

---

## Task 6：最終驗收

### Step 6-1：完成條件逐一確認

```bash
# 1. go build 通過
docker compose run --no-deps --rm app go build ./...
# 預期：無輸出（成功）

# 2. go test 通過
docker compose run --no-deps --rm app go test ./...
# 預期：全部 PASS

# 3. ALLOWED_ORIGINS 直讀已移除
grep "os.Getenv(\"ALLOWED_ORIGINS\")" services/api/cmd/server/main.go
# 預期：無輸出

# 4. .env.example 補齊
grep -E "LOG_LEVEL|ENABLE_SWAGGER|ENABLE_AUTOMIGRATE|ENABLE_SCHEDULER|GIN_MODE|TRUSTED_PROXIES" services/api/.env.example
# 預期：所有六個欄位出現

# 5. config_test.go 覆蓋新欄位
grep -E "TestLoad_ServerConfig|TestGetBoolEnv|TestGetCommaEnv" services/api/internal/config/config_test.go
# 預期：三個 test function 出現
```

---

## Task 7：開 PR

### Step 7-1：複製 PR template

```bash
cp .github/PULL_REQUEST_TEMPLATE.md /tmp/pr_body.md
```

### Step 7-2：填寫 PR body

編輯 `/tmp/pr_body.md`，重點欄位：
- **變更內容**：`ServerConfig` 新增七個欄位；新增 `getBoolEnv` / `getCommaEnv` helper；router.New 套用 GinMode / TrustedProxies / EnableSwagger；main.go 移除 `os.Getenv("ALLOWED_ORIGINS")` 直讀；`.env.example` 補齊欄位說明
- **測試方式**：`go test ./internal/config/...` 覆蓋新欄位預設值與 comma-split；`go build ./...` 通過；`grep os.Getenv...` 無輸出
- **Depends on PR**：none
- **closes #569**

### Step 7-3：發 PR

```bash
make pr-open TITLE="feat: Config 完整化 — CORS/LOG_LEVEL/ENABLE_*/GinMode/TrustedProxies 歸入 ServerConfig" BODY_FILE=/tmp/pr_body.md AUTO_READY=1
```

目標分支：`develop`

---

## 完成條件清單

- [ ] `go build ./...` 通過
- [ ] `go test ./...` 全部 PASS（無 regression）
- [ ] `main.go` 不再有 `os.Getenv("ALLOWED_ORIGINS")` 直讀
- [ ] `.env.example` 補齊 `LOG_LEVEL` / `ENABLE_SWAGGER` / `ENABLE_AUTOMIGRATE` / `ENABLE_SCHEDULER` / `GIN_MODE` / `TRUSTED_PROXIES`
- [ ] `config_test.go` 覆蓋新欄位的預設值、production 預設值、comma-split 解析、env override
