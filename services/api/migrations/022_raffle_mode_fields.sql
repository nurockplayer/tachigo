-- Raffle mode (public / subscribers_only), winner count, and entry-open flag.
-- raffle_entries gains source, eligible, and ineligible_reason for audit trails.

ALTER TABLE raffles
    ADD COLUMN IF NOT EXISTS mode         VARCHAR(50) NOT NULL DEFAULT 'public',
    ADD COLUMN IF NOT EXISTS winner_count INT         NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS entry_open   BOOLEAN     NOT NULL DEFAULT false;

ALTER TABLE raffle_entries
    ADD COLUMN IF NOT EXISTS source             VARCHAR(50),
    ADD COLUMN IF NOT EXISTS eligible           BOOLEAN     NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS ineligible_reason  VARCHAR(255);
