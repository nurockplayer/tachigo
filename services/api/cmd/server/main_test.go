package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestServerStartupDoesNotRunSchemaDDL(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	dir := filepath.Dir(file)
	var source strings.Builder
	for _, name := range []string{"main.go", "bootstrap.go", "wiring.go"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source.Write(body)
	}
	for _, forbidden := range []string{
		"AutoMigrate(",
		"initializeUserRoleEnum(",
		"ensureCouponRedemptionRuntimeSchema(",
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		"ALTER TABLE tachi_balances ADD CONSTRAINT",
		"applyStreamerAgencyMigration(db)",
	} {
		if strings.Contains(source.String(), forbidden) {
			t.Fatalf("server startup must not run schema DDL %q after Atlas owns migrations", forbidden)
		}
	}
	bootstrapBody, err := os.ReadFile(filepath.Join(dir, "bootstrap.go"))
	if err != nil {
		t.Fatalf("read bootstrap.go: %v", err)
	}
	bootstrapSource := string(bootstrapBody)
	if !strings.Contains(bootstrapSource, "hashLegacyRaffleClaimTokens(hashCtx, db)") {
		t.Fatalf("bootstrap should keep the non-schema raffle claim token data repair")
	}
	if !strings.Contains(bootstrapSource, "db.WithContext(ctx).Exec") {
		t.Fatalf("legacy raffle claim token repair must respect startup timeout context")
	}
}

func TestServerStartupIsSplitByResponsibility(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	dir := filepath.Dir(file)

	mainBody, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if lines := strings.Count(string(mainBody), "\n"); lines > 50 {
		t.Fatalf("main.go should stay at 50 lines or fewer after bootstrap/wiring split, got %d", lines)
	}
	for _, want := range []string{
		"bootstrap(cfg)",
		"wire(db, cfg, serverCtx)",
	} {
		if !strings.Contains(string(mainBody), want) {
			t.Fatalf("main.go should delegate startup with %q", want)
		}
	}

	bootstrapBody, err := os.ReadFile(filepath.Join(dir, "bootstrap.go"))
	if err != nil {
		t.Fatalf("read bootstrap.go: %v", err)
	}
	bootstrapSource := string(bootstrapBody)
	for _, want := range []string{
		"func bootstrap(cfg *config.Config) *gorm.DB",
		"database.Connect(cfg.Database.DSN)",
		"hashLegacyRaffleClaimTokens(hashCtx, db)",
	} {
		if !strings.Contains(bootstrapSource, want) {
			t.Fatalf("bootstrap.go should own %q", want)
		}
	}

	wiringBody, err := os.ReadFile(filepath.Join(dir, "wiring.go"))
	if err != nil {
		t.Fatalf("read wiring.go: %v", err)
	}
	wiringSource := string(wiringBody)
	for _, want := range []string{
		"func wire(db *gorm.DB, cfg *config.Config, ctx context.Context) *gin.Engine",
		"services.NewAuthService(db, cfg)",
		"context.WithTimeout(ctx, 10*time.Second)",
		"services.NewRaffleScheduler(raffleSvc).Start(ctx)",
		"router.New(",
	} {
		if !strings.Contains(wiringSource, want) {
			t.Fatalf("wiring.go should own %q", want)
		}
	}
	if strings.Contains(wiringSource, "context.WithTimeout(context.Background(), 10*time.Second)") {
		t.Fatalf("Sepolia RPC dial timeout should inherit the server context")
	}
}
