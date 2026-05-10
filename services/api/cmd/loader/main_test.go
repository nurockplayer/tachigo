package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tachigo/tachigo/internal/schema"
)

func TestAtlasModelsUseSharedAutoMigrateModels(t *testing.T) {
	got := modelTypeNames(atlasModels())
	want := modelTypeNames(schema.AutoMigrateModels())

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("atlas model list drifted from shared AutoMigrate model list\nwant: %#v\n got: %#v", want, got)
	}
}

func TestAtlasCustomPostgresSchemaPreservesRuntimeInvariants(t *testing.T) {
	sql := normalizeSQL(atlasCustomPostgresSchema())
	for _, want := range []string{
		"CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'agency', 'admin')",
		"ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
		"ALTER TABLE streamers ADD CONSTRAINT fk_streamers_agency_user_id FOREIGN KEY (agency_user_id) REFERENCES users(id)",
		"CREATE UNIQUE INDEX idx_auth_providers_provider_provider_id_active ON auth_providers (provider, provider_id) WHERE deleted_at IS NULL",
		"CREATE UNIQUE INDEX idx_auth_providers_web3_user_active ON auth_providers (user_id, provider) WHERE provider = 'web3' AND deleted_at IS NULL",
		"CREATE UNIQUE INDEX idx_watch_sessions_active_user_channel ON watch_sessions (user_id, channel_id) WHERE is_active = TRUE",
		"CREATE UNIQUE INDEX idx_points_ledgers_user_channel ON points_ledgers (user_id, channel_id)",
		"CREATE UNIQUE INDEX idx_points_transactions_external_transaction_id ON points_transactions (external_transaction_id) WHERE external_transaction_id IS NOT NULL",
		"CREATE UNIQUE INDEX idx_claims_tx_hash_not_null ON claims (tx_hash) WHERE tx_hash IS NOT NULL",
		"CREATE UNIQUE INDEX idx_claims_id_user_id ON claims (id, user_id)",
		"CREATE UNIQUE INDEX idx_points_ledgers_id_user_id ON points_ledgers (id, user_id)",
		"CREATE UNIQUE INDEX idx_points_transactions_id_ledger_id ON points_transactions (id, ledger_id)",
		"ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_claim_user FOREIGN KEY (claim_id, claim_user_id) REFERENCES claims(id, user_id) ON DELETE CASCADE",
		"ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_ledger_user FOREIGN KEY (ledger_id, claim_user_id) REFERENCES points_ledgers(id, user_id)",
		"ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_tx_ledger FOREIGN KEY (points_transaction_id, ledger_id) REFERENCES points_transactions(id, ledger_id)",
		"CREATE INDEX idx_coupon_redemptions_compensation ON coupon_redemptions (status) WHERE status = 'compensation-needed'",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("custom schema SQL missing %q:\n%s", want, sql)
		}
	}
}

func TestAtlasLoaderPreservesGormCheckConstraints(t *testing.T) {
	stmts, err := loadAtlasSchema()
	if err != nil {
		t.Fatalf("load atlas schema: %v", err)
	}

	sql := normalizeSQL(stmts)
	for _, want := range []string{
		"ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
		"ALTER TABLE streamers ADD CONSTRAINT fk_streamers_agency_user_id FOREIGN KEY (agency_user_id) REFERENCES users(id)",
		"ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_claim_user FOREIGN KEY (claim_id, claim_user_id) REFERENCES claims(id, user_id) ON DELETE CASCADE",
		"CREATE UNIQUE INDEX idx_claims_tx_hash_not_null ON claims (tx_hash) WHERE tx_hash IS NOT NULL",
		`CONSTRAINT "chk_coupon_redemptions_amount_gt_0" CHECK (amount > 0)`,
		`CONSTRAINT "chk_coupon_redemptions_status" CHECK (status IN ('pending','redeemed','compensation-needed'))`,
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("loader SQL missing %q:\n%s", want, sql)
		}
	}
}

func TestAtlasLoaderCreatesCompositeReferenceIndexesBeforeForeignKeys(t *testing.T) {
	stmts, err := loadAtlasSchema()
	if err != nil {
		t.Fatalf("load atlas schema: %v", err)
	}

	claimIndex := strings.Index(stmts, "CREATE UNIQUE INDEX idx_claims_id_user_id")
	ledgerIndex := strings.Index(stmts, "CREATE UNIQUE INDEX idx_points_ledgers_id_user_id")
	txIndex := strings.Index(stmts, "CREATE UNIQUE INDEX idx_points_transactions_id_ledger_id")
	claimFK := strings.Index(stmts, "ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_claim_user")
	ledgerFK := strings.Index(stmts, "ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_ledger_user")
	txFK := strings.Index(stmts, "ALTER TABLE claim_items ADD CONSTRAINT fk_claim_items_tx_ledger")

	for name, index := range map[string]int{
		"idx_claims_id_user_id":                claimIndex,
		"idx_points_ledgers_id_user_id":        ledgerIndex,
		"idx_points_transactions_id_ledger_id": txIndex,
		"fk_claim_items_claim_user":            claimFK,
		"fk_claim_items_ledger_user":           ledgerFK,
		"fk_claim_items_tx_ledger":             txFK,
	} {
		if index == -1 {
			t.Fatalf("loader SQL missing %s", name)
		}
	}
	if claimIndex > claimFK {
		t.Fatal("claims composite unique index must be created before claim_items claim/user FK")
	}
	if ledgerIndex > ledgerFK {
		t.Fatal("points_ledgers composite unique index must be created before claim_items ledger/user FK")
	}
	if txIndex > txFK {
		t.Fatal("points_transactions composite unique index must be created before claim_items transaction/ledger FK")
	}
}

func TestAtlasLoaderUsesPostgresDialect(t *testing.T) {
	if got := atlasDialect(); got != "postgres" {
		t.Fatalf("atlas loader dialect = %q, want postgres", got)
	}
}

func modelTypeNames(values []any) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, reflect.TypeOf(value).String())
	}
	return names
}

func normalizeSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
