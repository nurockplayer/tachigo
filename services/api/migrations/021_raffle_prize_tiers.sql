-- raffle_prize_tiers: sub-prize layers within a single raffle.
-- prize_tier_id on raffle_draws is nullable for backward compatibility.

CREATE TABLE raffle_prize_tiers (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    raffle_id        UUID         NOT NULL REFERENCES raffles(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    prize_description TEXT        NOT NULL DEFAULT '',
    winner_count     INT          NOT NULL CHECK (winner_count > 0),
    drawn_count      INT          NOT NULL DEFAULT 0,
    position         INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_raffle_prize_tiers_raffle_id
    ON raffle_prize_tiers (raffle_id, position);

ALTER TABLE raffle_draws
    ADD COLUMN prize_tier_id UUID REFERENCES raffle_prize_tiers(id) ON DELETE SET NULL;
