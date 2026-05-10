-- Atlas baseline reconciliation.
--
-- This migration lets Atlas take ownership of the existing migration directory
-- without replaying a full schema dump over databases that have already been
-- shaped by GORM AutoMigrate and runtime schema patches.

-- Tables that existed in GORM models/runtime but not in historical 001-019 SQL.
CREATE TABLE IF NOT EXISTS watch_time_stats (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL, channel_id VARCHAR(255) NOT NULL, total_watch_seconds BIGINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);

CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_time_user_channel ON watch_time_stats (user_id, channel_id);

CREATE TABLE IF NOT EXISTS broadcast_time_stats (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), streamer_id UUID NOT NULL, channel_id VARCHAR(255) NOT NULL, total_broadcast_seconds BIGINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);

CREATE UNIQUE INDEX IF NOT EXISTS idx_broadcast_time_streamer_channel ON broadcast_time_stats (streamer_id, channel_id);

CREATE TABLE IF NOT EXISTS broadcast_time_logs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), streamer_id UUID NOT NULL, channel_id VARCHAR(255) NOT NULL, seconds BIGINT NOT NULL, recorded_at TIMESTAMPTZ NOT NULL);

CREATE INDEX IF NOT EXISTS idx_broadcast_time_logs_streamer_id ON broadcast_time_logs (streamer_id);

CREATE INDEX IF NOT EXISTS idx_broadcast_time_logs_channel_id ON broadcast_time_logs (channel_id);

CREATE INDEX IF NOT EXISTS idx_broadcast_time_logs_recorded_at ON broadcast_time_logs (recorded_at);

CREATE TABLE IF NOT EXISTS raffles (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL, title VARCHAR(255) NOT NULL, status VARCHAR(50) NOT NULL DEFAULT 'draft', source VARCHAR(50) NOT NULL DEFAULT 'csv', scheduled_at TIMESTAMPTZ, discord_webhook_url VARCHAR(512), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);

CREATE INDEX IF NOT EXISTS idx_raffles_user_id ON raffles (user_id);

CREATE TABLE IF NOT EXISTS raffle_entries (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), raffle_id UUID NOT NULL, user_id UUID, twitch_login VARCHAR(255) NOT NULL, display_name VARCHAR(255), created_at TIMESTAMPTZ);

CREATE INDEX IF NOT EXISTS idx_raffle_entries_user_id ON raffle_entries (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_raffle_entry_twitch ON raffle_entries (raffle_id, twitch_login);

CREATE UNIQUE INDEX IF NOT EXISTS idx_entry_id_raffle ON raffle_entries (id, raffle_id);

CREATE TABLE IF NOT EXISTS raffle_draws (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), raffle_id UUID NOT NULL, entry_id UUID NOT NULL, claim_token VARCHAR(255) NOT NULL, claim_expires_at TIMESTAMPTZ, drawn_at TIMESTAMPTZ);

CREATE UNIQUE INDEX IF NOT EXISTS idx_raffle_draws_claim_token ON raffle_draws (claim_token);

CREATE UNIQUE INDEX IF NOT EXISTS idx_raffle_draw_entry ON raffle_draws (raffle_id, entry_id);

CREATE TABLE IF NOT EXISTS raffle_claims (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), draw_id UUID NOT NULL, recipient_name VARCHAR(255) NOT NULL, phone VARCHAR(50), address_line1 VARCHAR(255) NOT NULL, address_line2 VARCHAR(255), city VARCHAR(100) NOT NULL, postal_code VARCHAR(20), country VARCHAR(10) NOT NULL DEFAULT 'TW', submitted_at TIMESTAMPTZ);

CREATE UNIQUE INDEX IF NOT EXISTS idx_raffle_claims_draw_id ON raffle_claims (draw_id);

-- Runtime schema patches now owned by Atlas.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_tachi_balances_user_id'
    ) THEN
        ALTER TABLE tachi_balances
            ADD CONSTRAINT fk_tachi_balances_user_id
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

ALTER TABLE streamers
    ADD COLUMN IF NOT EXISTS agency_user_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_streamers_agency_user_id'
    ) THEN
        ALTER TABLE streamers
            ADD CONSTRAINT fk_streamers_agency_user_id
            FOREIGN KEY (agency_user_id) REFERENCES users(id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id
    ON streamers (agency_user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_user_channel
    ON streamers (user_id, channel_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_user_channel
    ON points_ledgers (user_id, channel_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_transactions_external_transaction_id
    ON points_transactions (external_transaction_id)
    WHERE external_transaction_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_tx_hash_not_null
    ON claims (tx_hash)
    WHERE tx_hash IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_coupon_redemptions_amount_gt_0'
    ) THEN
        ALTER TABLE coupon_redemptions
            ADD CONSTRAINT chk_coupon_redemptions_amount_gt_0
            CHECK (amount > 0);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_coupon_redemptions_status'
    ) THEN
        ALTER TABLE coupon_redemptions
            ADD CONSTRAINT chk_coupon_redemptions_status
            CHECK (status IN ('pending','redeemed','compensation-needed'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_compensation
    ON coupon_redemptions (status)
    WHERE status = 'compensation-needed';

-- Preserve historical claim consistency invariants before Atlas diffs from GORM.
CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_id_user_id
    ON claims (id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_id_user_id
    ON points_ledgers (id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_transactions_id_ledger_id
    ON points_transactions (id, ledger_id);

ALTER TABLE claim_items
    ADD COLUMN IF NOT EXISTS claim_user_id UUID;

DO $$
DECLARE
    missing_claim_count INTEGER;
    missing_ledger_count INTEGER;
    missing_tx_count INTEGER;
    claim_ledger_user_mismatch_count INTEGER;
    tx_ledger_mismatch_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO missing_claim_count
    FROM claim_items ci
    LEFT JOIN claims c ON c.id = ci.claim_id
    WHERE c.id IS NULL;

    SELECT COUNT(*) INTO missing_ledger_count
    FROM claim_items ci
    LEFT JOIN points_ledgers pl ON pl.id = ci.ledger_id
    WHERE pl.id IS NULL;

    SELECT COUNT(*) INTO missing_tx_count
    FROM claim_items ci
    LEFT JOIN points_transactions pt ON pt.id = ci.points_transaction_id
    WHERE pt.id IS NULL;

    SELECT COUNT(*) INTO claim_ledger_user_mismatch_count
    FROM claim_items ci
    JOIN claims c ON c.id = ci.claim_id
    JOIN points_ledgers pl ON pl.id = ci.ledger_id
    WHERE c.user_id <> pl.user_id;

    SELECT COUNT(*) INTO tx_ledger_mismatch_count
    FROM claim_items ci
    JOIN points_transactions pt ON pt.id = ci.points_transaction_id
    WHERE pt.ledger_id <> ci.ledger_id;

    IF missing_claim_count > 0
       OR missing_ledger_count > 0
       OR missing_tx_count > 0
       OR claim_ledger_user_mismatch_count > 0
       OR tx_ledger_mismatch_count > 0 THEN
        RAISE EXCEPTION
            'migration 020 blocked: inconsistent claim_items rows detected (missing_claim=%, missing_ledger=%, missing_tx=%, claim_ledger_user_mismatch=%, tx_ledger_mismatch=%). resolve data before re-running migration',
            missing_claim_count,
            missing_ledger_count,
            missing_tx_count,
            claim_ledger_user_mismatch_count,
            tx_ledger_mismatch_count;
    END IF;
END $$;

UPDATE claim_items ci
SET claim_user_id = c.user_id
FROM claims c
WHERE ci.claim_id = c.id
  AND ci.claim_user_id IS NULL;

ALTER TABLE claim_items
    ALTER COLUMN claim_user_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_claim_items_claim_user'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_claim_user
            FOREIGN KEY (claim_id, claim_user_id)
            REFERENCES claims (id, user_id)
            ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_claim_items_ledger_user'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_ledger_user
            FOREIGN KEY (ledger_id, claim_user_id)
            REFERENCES points_ledgers (id, user_id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_claim_items_tx_ledger'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_tx_ledger
            FOREIGN KEY (points_transaction_id, ledger_id)
            REFERENCES points_transactions (id, ledger_id);
    END IF;
END $$;

-- Foreign keys for tables introduced by the reconciliation migration.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffles_user') THEN
        ALTER TABLE raffles
            ADD CONSTRAINT fk_raffles_user
            FOREIGN KEY (user_id) REFERENCES users(id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffle_entries_raffle') THEN
        ALTER TABLE raffle_entries
            ADD CONSTRAINT fk_raffle_entries_raffle
            FOREIGN KEY (raffle_id) REFERENCES raffles(id)
            ON UPDATE CASCADE ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffle_entries_user') THEN
        ALTER TABLE raffle_entries
            ADD CONSTRAINT fk_raffle_entries_user
            FOREIGN KEY (user_id) REFERENCES users(id)
            ON UPDATE CASCADE ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffle_draws_raffle') THEN
        ALTER TABLE raffle_draws
            ADD CONSTRAINT fk_raffle_draws_raffle
            FOREIGN KEY (raffle_id) REFERENCES raffles(id)
            ON UPDATE CASCADE ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffle_draws_entry') THEN
        ALTER TABLE raffle_draws
            ADD CONSTRAINT fk_raffle_draws_entry
            FOREIGN KEY (entry_id, raffle_id) REFERENCES raffle_entries(id, raffle_id)
            ON UPDATE CASCADE ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_raffle_claims_draw') THEN
        ALTER TABLE raffle_claims
            ADD CONSTRAINT fk_raffle_claims_draw
            FOREIGN KEY (draw_id) REFERENCES raffle_draws(id)
            ON UPDATE CASCADE ON DELETE CASCADE;
    END IF;
END $$;
