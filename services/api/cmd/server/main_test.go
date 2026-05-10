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

	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	source := string(body)
	for _, forbidden := range []string{
		"AutoMigrate(",
		"initializeUserRoleEnum(",
		"ensureCouponRedemptionRuntimeSchema(",
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		"ALTER TABLE tachi_balances ADD CONSTRAINT",
		"applyStreamerAgencyMigration(db)",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("server startup must not run schema DDL %q after Atlas owns migrations", forbidden)
		}
	}
	if !strings.Contains(source, "hashLegacyRaffleClaimTokens(hashCtx, db)") {
		t.Fatalf("server startup should keep the non-schema raffle claim token data repair")
	}
	if !strings.Contains(source, "db.WithContext(ctx).Exec") {
		t.Fatalf("legacy raffle claim token repair must respect startup timeout context")
	}
}
