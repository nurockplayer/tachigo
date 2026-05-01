-- migrations/019_coupon_redemptions.sql
-- Tracks the lifecycle of each coupon redemption attempt for compensation.

CREATE TABLE IF NOT EXISTS coupon_redemptions (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    coupon_id     VARCHAR(255) NOT NULL,
    amount        BIGINT       NOT NULL CHECK (amount > 0),
    tx_hash       VARCHAR(255) NOT NULL,
    status        VARCHAR(50)  NOT NULL
                  CHECK (status IN ('pending', 'redeemed', 'compensation-needed')),
    voucher_code  VARCHAR(255),
    error_message TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_user_id
    ON coupon_redemptions (user_id);

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_compensation
    ON coupon_redemptions (status)
    WHERE status = 'compensation-needed';
