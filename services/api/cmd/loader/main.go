package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"ariga.io/atlas-provider-gorm/gormschema"

	"github.com/tachigo/tachigo/internal/schema"
)

func main() {
	stmts, err := loadAtlasSchema()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	_, _ = io.WriteString(os.Stdout, stmts)
}

func loadAtlasSchema() (string, error) {
	stmts, err := gormschema.New(atlasDialect()).Load(atlasModels()...)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		atlasCustomPostgresTypes(),
		stmts,
		atlasCustomPostgresConstraints(),
		atlasCustomPostgresIndexes(),
	}, "\n\n"), nil
}

func atlasDialect() string {
	return "postgres"
}

func atlasModels() []any {
	return schema.AutoMigrateModels()
}

func atlasCustomPostgresSchema() string {
	return strings.Join([]string{
		atlasCustomPostgresTypes(),
		atlasCustomPostgresConstraints(),
		atlasCustomPostgresIndexes(),
	}, "\n\n")
}

func atlasCustomPostgresTypes() string {
	return `CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'agency', 'admin');`
}

func atlasCustomPostgresConstraints() string {
	return `ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;`
}

func atlasCustomPostgresIndexes() string {
	return strings.Join([]string{
		`CREATE UNIQUE INDEX idx_auth_providers_provider_provider_id_active
ON auth_providers (provider, provider_id)
WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX idx_auth_providers_web3_user_active
ON auth_providers (user_id, provider)
WHERE provider = 'web3' AND deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX idx_watch_sessions_active_user_channel
ON watch_sessions (user_id, channel_id)
WHERE is_active = TRUE;`,
		`CREATE UNIQUE INDEX idx_points_ledgers_user_channel
ON points_ledgers (user_id, channel_id);`,
		`CREATE UNIQUE INDEX idx_points_transactions_external_transaction_id
ON points_transactions (external_transaction_id)
WHERE external_transaction_id IS NOT NULL;`,
		`CREATE INDEX idx_coupon_redemptions_compensation
ON coupon_redemptions (status)
WHERE status = 'compensation-needed';`,
	}, "\n\n")
}
