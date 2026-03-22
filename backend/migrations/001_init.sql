-- migrations/001_init.sql
-- Reference schema (GORM AutoMigrate handles this automatically in dev)
-- Use this for manual DB setup or production migrations.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  UNIQUE,
    email         VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    avatar_url    TEXT,
    role          VARCHAR(20)  NOT NULL DEFAULT 'viewer',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Auth providers (Twitch, Google, Web3 wallet, Email)
CREATE TABLE IF NOT EXISTS auth_providers (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(20)  NOT NULL,
    provider_id      VARCHAR(255) NOT NULL,
    access_token     TEXT,
    refresh_token    TEXT,
    token_expires_at TIMESTAMPTZ,
    metadata         JSONB,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    UNIQUE (provider, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_providers_user_id    ON auth_providers(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_providers_deleted_at ON auth_providers(deleted_at);

-- Shipping addresses
CREATE TABLE IF NOT EXISTS shipping_addresses (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_name VARCHAR(100) NOT NULL,
    phone          VARCHAR(20),
    address_line1  VARCHAR(255) NOT NULL,
    address_line2  VARCHAR(255),
    city           VARCHAR(100) NOT NULL,
    district       VARCHAR(100),
    postal_code    VARCHAR(20),
    country        VARCHAR(50)  NOT NULL DEFAULT 'TW',
    is_default     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_shipping_addresses_user_id    ON shipping_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_shipping_addresses_deleted_at ON shipping_addresses(deleted_at);

-- Refresh tokens (hashed)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Web3 nonces (one-time use, 5 min TTL)
CREATE TABLE IF NOT EXISTS web3_nonces (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nonce      VARCHAR(64) NOT NULL UNIQUE,
    address    VARCHAR(42) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_web3_nonces_address ON web3_nonces(address);
