package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestServerAutoMigrateUsesSharedSchemaModelList(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	source := string(body)
	if !strings.Contains(source, "db.AutoMigrate(schema.AutoMigrateModels()...)") {
		t.Fatalf("server AutoMigrate must use shared schema.AutoMigrateModels list")
	}
	if strings.Contains(source, "&models.") {
		t.Fatalf("server AutoMigrate must not keep a separate hard-coded models list")
	}
}

func TestInitializeUserRoleEnumFreshDatabase(t *testing.T) {
	var statements []string

	err := initializeUserRoleEnum(func(query string) error {
		statements = append(statements, normalizeSQL(query))
		return nil
	})
	if err != nil {
		t.Fatalf("initializeUserRoleEnum returned error: %v", err)
	}

	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(statements))
	}
	if !strings.Contains(statements[0], "CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'agency', 'admin')") {
		t.Fatalf("statement should create enum, got %q", statements[0])
	}
	if !strings.Contains(statements[1], "ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'") {
		t.Fatalf("statement should alter enum, got %q", statements[1])
	}
}

func TestInitializeUserRoleEnumExistingDatabase(t *testing.T) {
	var statements []string
	callCount := 0

	err := initializeUserRoleEnum(func(query string) error {
		statements = append(statements, normalizeSQL(query))
		callCount++
		if callCount == 1 {
			return &pgconn.PgError{Code: "42710"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("initializeUserRoleEnum returned error: %v", err)
	}

	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(statements))
	}
	if !strings.Contains(statements[0], "CREATE TYPE user_role AS ENUM") {
		t.Fatalf("first statement should create enum, got %q", statements[0])
	}
	if !strings.Contains(statements[1], "ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'") {
		t.Fatalf("second statement should alter enum, got %q", statements[1])
	}
}

func TestEnsureCouponRedemptionRuntimeSchema(t *testing.T) {
	var statements []string

	err := ensureCouponRedemptionRuntimeSchema(func(query string) error {
		statements = append(statements, normalizeSQL(query))
		return nil
	})
	if err != nil {
		t.Fatalf("ensureCouponRedemptionRuntimeSchema returned error: %v", err)
	}

	joined := strings.Join(statements, " ")
	for _, want := range []string{
		"CONSTRAINT chk_coupon_redemptions_amount_gt_0",
		"CHECK (amount > 0)",
		"EXCEPTION WHEN duplicate_object THEN NULL",
		"CONSTRAINT chk_coupon_redemptions_status",
		"status IN ('pending','redeemed','compensation-needed')",
		"CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_compensation",
		"WHERE status = 'compensation-needed'",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("runtime schema SQL missing %q:\n%s", want, joined)
		}
	}
}

func normalizeSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
