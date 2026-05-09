-- Creates claim lifecycle tables used to track pending/broadcast/confirmed/failed states.

CREATE TABLE IF NOT EXISTS claims (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_addr   VARCHAR(42) NOT NULL,
    amount        BIGINT NOT NULL CHECK (amount > 0),
    status        VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed')),
    tx_hash       VARCHAR(66),
    error_message TEXT,
    broadcast_at  TIMESTAMPTZ,
    confirmed_at  TIMESTAMPTZ,
    failed_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claims_user_created_at
    ON claims (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_claims_status_created_at
    ON claims (status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_tx_hash_not_null
    ON claims (tx_hash)
    WHERE tx_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS claim_items (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id               UUID NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    ledger_id              UUID NOT NULL REFERENCES points_ledgers(id),
    points_transaction_id  UUID NOT NULL REFERENCES points_transactions(id),
    amount                 BIGINT NOT NULL CHECK (amount > 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (points_transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_claim_items_claim_id
    ON claim_items (claim_id);

CREATE INDEX IF NOT EXISTS idx_claim_items_ledger_id
    ON claim_items (ledger_id);
