-- Drop the global unique constraint added in 001_init.sql.
-- Soft-deleted rows (deleted_at IS NOT NULL) must not block re-binding
-- the same wallet, so we replace it with partial unique indexes.
ALTER TABLE auth_providers
    -- atlas:nolint
    DROP CONSTRAINT IF EXISTS auth_providers_provider_provider_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS
    idx_auth_providers_provider_provider_id_active
ON auth_providers(provider, provider_id)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS
    idx_auth_providers_web3_user_active
ON auth_providers(user_id, provider)
WHERE provider = 'web3' AND deleted_at IS NULL;
