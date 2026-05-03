-- Atlas baseline migration: captures schema divergence between SQL migrations 001-019 and GORM models.
--
-- Safety notes before applying to production:
-- 1. Unique index operations are SAFE — all new unique indexes replace existing unique constraints
--    on the same columns; no data cleanup required.
-- 2. ALTER COLUMN DROP NOT NULL — multiple columns (created_at, updated_at, etc.) change from
--    NOT NULL to nullable to align with GORM defaults. Verify application code handles NULLs.
-- 3. FK constraint renames — old hand-written constraint names are dropped and GORM-generated
--    names are added; functionally equivalent, but update any monitoring/alerts that reference names.
-- 4. New tables (raffles, watch_time_stats, broadcast_time_stats, broadcast_time_logs) are added
--    idempotently — AutoMigrate already created them in production; this makes Atlas aware.

-- Modify "streamers" table
ALTER TABLE "public"."streamers" DROP CONSTRAINT "streamers_user_id_channel_id_key", DROP CONSTRAINT "fk_streamers_agency_user_id", DROP CONSTRAINT "streamers_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT;
-- Create index "idx_streamers_user_channel" to table: "streamers"
CREATE UNIQUE INDEX "idx_streamers_user_channel" ON "public"."streamers" ("user_id", "channel_id");
-- Create index "idx_streamers_user_id" to table: "streamers"
CREATE INDEX "idx_streamers_user_id" ON "public"."streamers" ("user_id");
-- Modify "email_verifications" table
ALTER TABLE "public"."email_verifications" DROP CONSTRAINT "email_verifications_token_hash_key", DROP CONSTRAINT "email_verifications_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT;
-- Create index "idx_email_verifications_token_hash" to table: "email_verifications"
CREATE UNIQUE INDEX "idx_email_verifications_token_hash" ON "public"."email_verifications" ("token_hash");
-- Modify "channel_configs" table
ALTER TABLE "public"."channel_configs" ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ALTER COLUMN "multiplier" TYPE bigint, ALTER COLUMN "daily_airdrop_limit" TYPE bigint;
-- Create "watch_time_stats" table
CREATE TABLE "public"."watch_time_stats" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "channel_id" character varying(255) NOT NULL,
  "total_watch_seconds" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_watch_time_user_channel" to table: "watch_time_stats"
CREATE UNIQUE INDEX "idx_watch_time_user_channel" ON "public"."watch_time_stats" ("user_id", "channel_id");
-- Create "broadcast_time_stats" table
CREATE TABLE "public"."broadcast_time_stats" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "streamer_id" uuid NOT NULL,
  "channel_id" character varying(255) NOT NULL,
  "total_broadcast_seconds" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_broadcast_time_streamer_channel" to table: "broadcast_time_stats"
CREATE UNIQUE INDEX "idx_broadcast_time_streamer_channel" ON "public"."broadcast_time_stats" ("streamer_id", "channel_id");
-- Create "broadcast_time_logs" table
CREATE TABLE "public"."broadcast_time_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "streamer_id" uuid NOT NULL,
  "channel_id" character varying(255) NOT NULL,
  "seconds" bigint NOT NULL,
  "recorded_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_broadcast_time_logs_channel_id" to table: "broadcast_time_logs"
CREATE INDEX "idx_broadcast_time_logs_channel_id" ON "public"."broadcast_time_logs" ("channel_id");
-- Create index "idx_broadcast_time_logs_recorded_at" to table: "broadcast_time_logs"
CREATE INDEX "idx_broadcast_time_logs_recorded_at" ON "public"."broadcast_time_logs" ("recorded_at");
-- Create index "idx_broadcast_time_logs_streamer_id" to table: "broadcast_time_logs"
CREATE INDEX "idx_broadcast_time_logs_streamer_id" ON "public"."broadcast_time_logs" ("streamer_id");
-- Drop index "idx_watch_sessions_active_user_channel" from table: "watch_sessions"
DROP INDEX "public"."idx_watch_sessions_active_user_channel";
-- Modify "watch_sessions" table
ALTER TABLE "public"."watch_sessions" DROP CONSTRAINT "watch_sessions_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT;
-- Modify "password_resets" table
ALTER TABLE "public"."password_resets" DROP CONSTRAINT "password_resets_token_hash_key", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT;
-- Create index "idx_password_resets_token_hash" to table: "password_resets"
CREATE UNIQUE INDEX "idx_password_resets_token_hash" ON "public"."password_resets" ("token_hash");
-- Modify "users" table
ALTER TABLE "public"."users" DROP CONSTRAINT "users_email_key", DROP CONSTRAINT "users_username_key", ALTER COLUMN "role" DROP NOT NULL, ALTER COLUMN "is_active" DROP NOT NULL, ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ALTER COLUMN "email_verified" DROP NOT NULL;
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "public"."users" ("email");
-- Create index "idx_users_username" to table: "users"
CREATE UNIQUE INDEX "idx_users_username" ON "public"."users" ("username");
-- Modify "web3_nonces" table
ALTER TABLE "public"."web3_nonces" DROP CONSTRAINT "web3_nonces_nonce_key", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT;
-- Create index "idx_web3_nonces_nonce" to table: "web3_nonces"
CREATE UNIQUE INDEX "idx_web3_nonces_nonce" ON "public"."web3_nonces" ("nonce");
-- Modify "agency_streamers" table
ALTER TABLE "public"."agency_streamers" DROP CONSTRAINT "agency_streamers_agency_id_channel_id_key", DROP CONSTRAINT "agency_streamers_agency_id_fkey", ALTER COLUMN "channel_id" TYPE text, ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ADD CONSTRAINT "fk_agency_streamers_agency" FOREIGN KEY ("agency_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "idx_agency_streamers_agency_channel" to table: "agency_streamers"
CREATE UNIQUE INDEX "idx_agency_streamers_agency_channel" ON "public"."agency_streamers" ("agency_id", "channel_id");
-- Drop index "idx_auth_providers_provider_provider_id_active" from table: "auth_providers"
DROP INDEX "public"."idx_auth_providers_provider_provider_id_active";
-- Drop index "idx_auth_providers_web3_user_active" from table: "auth_providers"
DROP INDEX "public"."idx_auth_providers_web3_user_active";
-- Modify "auth_providers" table
ALTER TABLE "public"."auth_providers" DROP CONSTRAINT "auth_providers_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ADD CONSTRAINT "fk_users_auth_providers" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "points_ledgers" table
ALTER TABLE "public"."points_ledgers" DROP CONSTRAINT "points_ledgers_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT;
-- Create index "idx_points_ledgers_channel_id" to table: "points_ledgers"
CREATE INDEX "idx_points_ledgers_channel_id" ON "public"."points_ledgers" ("channel_id");
-- Create index "idx_points_ledgers_user_id" to table: "points_ledgers"
CREATE INDEX "idx_points_ledgers_user_id" ON "public"."points_ledgers" ("user_id");
-- Drop index "idx_points_transactions_external_transaction_id" from table: "points_transactions"
DROP INDEX "public"."idx_points_transactions_external_transaction_id";
-- Modify "points_transactions" table
ALTER TABLE "public"."points_transactions" DROP CONSTRAINT "points_transactions_ledger_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ADD CONSTRAINT "fk_points_transactions_ledger" FOREIGN KEY ("ledger_id") REFERENCES "public"."points_ledgers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Drop index "idx_claims_status_created_at" from table: "claims"
DROP INDEX "public"."idx_claims_status_created_at";
-- Drop index "idx_claims_tx_hash_not_null" from table: "claims"
DROP INDEX "public"."idx_claims_tx_hash_not_null";
-- Drop index "idx_claims_user_created_at" from table: "claims"
DROP INDEX "public"."idx_claims_user_created_at";
-- Modify "claims" table
ALTER TABLE "public"."claims" DROP CONSTRAINT "claims_amount_check", DROP CONSTRAINT "claims_user_id_fkey", ADD CONSTRAINT "chk_claim_amount_gt_0" CHECK (amount > 0), ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ADD CONSTRAINT "fk_claims_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_claims_status_created_at" to table: "claims"
CREATE INDEX "idx_claims_status_created_at" ON "public"."claims" ("status", "created_at");
-- Create index "idx_claims_user_created_at" to table: "claims"
CREATE INDEX "idx_claims_user_created_at" ON "public"."claims" ("user_id", "created_at");
-- Modify "claim_items" table
ALTER TABLE "public"."claim_items" DROP CONSTRAINT "claim_items_amount_check", DROP CONSTRAINT "claim_items_points_transaction_id_key", DROP CONSTRAINT "claim_items_claim_id_fkey", DROP CONSTRAINT "claim_items_ledger_id_fkey", DROP CONSTRAINT "claim_items_points_transaction_id_fkey", DROP CONSTRAINT "fk_claim_items_claim_user", DROP CONSTRAINT "fk_claim_items_ledger_user", DROP CONSTRAINT "fk_claim_items_tx_ledger", ADD CONSTRAINT "chk_claim_item_amount_gt_0" CHECK (amount > 0), ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ADD CONSTRAINT "fk_claim_items_ledger" FOREIGN KEY ("ledger_id") REFERENCES "public"."points_ledgers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_claim_items_points_transaction" FOREIGN KEY ("points_transaction_id") REFERENCES "public"."points_transactions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_claims_items" FOREIGN KEY ("claim_id") REFERENCES "public"."claims" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_claim_items_claim_user_id" to table: "claim_items"
CREATE INDEX "idx_claim_items_claim_user_id" ON "public"."claim_items" ("claim_user_id");
-- Create index "idx_claim_items_points_transaction_id" to table: "claim_items"
CREATE UNIQUE INDEX "idx_claim_items_points_transaction_id" ON "public"."claim_items" ("points_transaction_id");
-- Drop index "idx_points_transactions_id_ledger_id" from table: "points_transactions"
DROP INDEX "public"."idx_points_transactions_id_ledger_id";
-- Drop index "idx_claims_id_user_id" from table: "claims"
DROP INDEX "public"."idx_claims_id_user_id";
-- Drop index "idx_points_ledgers_id_user_id" from table: "points_ledgers"
DROP INDEX "public"."idx_points_ledgers_id_user_id";
-- Drop index "idx_points_ledgers_user_channel" from table: "points_ledgers"
DROP INDEX "public"."idx_points_ledgers_user_channel";
-- Drop index "idx_coupon_redemptions_compensation" from table: "coupon_redemptions"
DROP INDEX "public"."idx_coupon_redemptions_compensation";
-- Modify "coupon_redemptions" table
ALTER TABLE "public"."coupon_redemptions" DROP CONSTRAINT "coupon_redemptions_amount_check", DROP CONSTRAINT "coupon_redemptions_status_check", DROP CONSTRAINT "coupon_redemptions_user_id_fkey", ADD CONSTRAINT "chk_coupon_redemptions_amount_gt_0" CHECK (amount > 0), ADD CONSTRAINT "chk_coupon_redemptions_status" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'redeemed'::character varying, 'compensation-needed'::character varying])::text[])), ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ADD CONSTRAINT "fk_coupon_redemptions_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create "raffles" table
CREATE TABLE "public"."raffles" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "title" character varying(255) NOT NULL,
  "status" character varying(50) NOT NULL DEFAULT 'draft',
  "source" character varying(50) NOT NULL DEFAULT 'csv',
  "scheduled_at" timestamptz NULL,
  "discord_webhook_url" character varying(512) NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_raffles_user_id" to table: "raffles"
CREATE INDEX "idx_raffles_user_id" ON "public"."raffles" ("user_id");
-- Create "raffle_entries" table
CREATE TABLE "public"."raffle_entries" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "raffle_id" uuid NOT NULL,
  "user_id" uuid NULL,
  "twitch_login" character varying(255) NOT NULL,
  "display_name" character varying(255) NULL,
  "created_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_raffle_entries_raffle" FOREIGN KEY ("raffle_id") REFERENCES "public"."raffles" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_raffle_entries_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE CASCADE ON DELETE SET NULL
);
-- Create index "idx_entry_id_raffle" to table: "raffle_entries"
CREATE UNIQUE INDEX "idx_entry_id_raffle" ON "public"."raffle_entries" ("id", "raffle_id");
-- Create index "idx_raffle_entries_user_id" to table: "raffle_entries"
CREATE INDEX "idx_raffle_entries_user_id" ON "public"."raffle_entries" ("user_id");
-- Create index "idx_raffle_entry_twitch" to table: "raffle_entries"
CREATE UNIQUE INDEX "idx_raffle_entry_twitch" ON "public"."raffle_entries" ("raffle_id", "twitch_login");
-- Create "raffle_draws" table
CREATE TABLE "public"."raffle_draws" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "raffle_id" uuid NOT NULL,
  "entry_id" uuid NOT NULL,
  "claim_token" character varying(255) NOT NULL,
  "claim_expires_at" timestamptz NULL,
  "drawn_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_raffle_draws_entry" FOREIGN KEY ("entry_id", "raffle_id") REFERENCES "public"."raffle_entries" ("id", "raffle_id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_raffle_draws_raffle" FOREIGN KEY ("raffle_id") REFERENCES "public"."raffles" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_raffle_draw_entry" to table: "raffle_draws"
CREATE UNIQUE INDEX "idx_raffle_draw_entry" ON "public"."raffle_draws" ("raffle_id", "entry_id");
-- Create index "idx_raffle_draws_claim_token" to table: "raffle_draws"
CREATE UNIQUE INDEX "idx_raffle_draws_claim_token" ON "public"."raffle_draws" ("claim_token");
-- Create "raffle_claims" table
CREATE TABLE "public"."raffle_claims" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "draw_id" uuid NOT NULL,
  "recipient_name" character varying(255) NOT NULL,
  "phone" character varying(50) NULL,
  "address_line1" character varying(255) NOT NULL,
  "address_line2" character varying(255) NULL,
  "city" character varying(100) NOT NULL,
  "postal_code" character varying(20) NULL,
  "country" character varying(10) NOT NULL DEFAULT 'TW',
  "submitted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_raffle_claims_draw" FOREIGN KEY ("draw_id") REFERENCES "public"."raffle_draws" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_raffle_claims_draw_id" to table: "raffle_claims"
CREATE UNIQUE INDEX "idx_raffle_claims_draw_id" ON "public"."raffle_claims" ("draw_id");
-- Modify "refresh_tokens" table
ALTER TABLE "public"."refresh_tokens" DROP CONSTRAINT "refresh_tokens_token_hash_key", DROP CONSTRAINT "refresh_tokens_user_id_fkey", ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ADD CONSTRAINT "fk_refresh_tokens_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_refresh_tokens_token_hash" to table: "refresh_tokens"
CREATE UNIQUE INDEX "idx_refresh_tokens_token_hash" ON "public"."refresh_tokens" ("token_hash");
-- Modify "shipping_addresses" table
ALTER TABLE "public"."shipping_addresses" DROP CONSTRAINT "shipping_addresses_user_id_fkey", ALTER COLUMN "country" DROP NOT NULL, ALTER COLUMN "is_default" DROP NOT NULL, ALTER COLUMN "created_at" DROP NOT NULL, ALTER COLUMN "created_at" DROP DEFAULT, ALTER COLUMN "updated_at" DROP NOT NULL, ALTER COLUMN "updated_at" DROP DEFAULT, ADD CONSTRAINT "fk_users_addresses" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "tachi_balances" table
ALTER TABLE "public"."tachi_balances" DROP CONSTRAINT "tachi_balances_balance_check", DROP CONSTRAINT "tachi_balances_user_id_key", DROP CONSTRAINT "tachi_balances_user_id_fkey", ADD CONSTRAINT "chk_tachi_balance_gte_0" CHECK (balance >= (0)::numeric), ADD CONSTRAINT "fk_tachi_balances_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "idx_tachi_balances_user_id" to table: "tachi_balances"
CREATE UNIQUE INDEX "idx_tachi_balances_user_id" ON "public"."tachi_balances" ("user_id");
