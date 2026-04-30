-- migrations/012_tachi_balances.sql
-- Creates per-user Tachi balance ledger table.

CREATE TABLE IF NOT EXISTS tachi_balances (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance    NUMERIC(20,6)  NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (user_id),
    CHECK (balance >= 0)
);
