#!/usr/bin/env bash
set -euo pipefail

: "${STAGING_API_BASE_URL:?STAGING_API_BASE_URL is required}"
: "${STAGING_AUTH_BEARER_TOKEN:?STAGING_AUTH_BEARER_TOKEN is required}"
: "${DEPLOYMENT_SHA:?DEPLOYMENT_SHA is required}"
: "${MIGRATION_STATUS:?MIGRATION_STATUS is required}"

base_url="${STAGING_API_BASE_URL%/}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

probe() {
  local label="$1"
  local path="$2"
  local output="$tmp_dir/${label}.json"
  local status

  status="$(curl -fsS -o "$output" -w "%{http_code}" "$base_url$path")"
  printf '%s status=%s path=%s\n' "$label" "$status" "$path"
}

auth_probe() {
  local label="$1"
  local path="$2"
  local output="$tmp_dir/${label}.json"
  local status

  status="$(
    curl -fsS \
      -H "Authorization: Bearer ${STAGING_AUTH_BEARER_TOKEN}" \
      -o "$output" \
      -w "%{http_code}" \
      "$base_url$path"
  )"
  printf '%s status=%s path=%s\n' "$label" "$status" "$path"
}

printf 'deployment_sha=%s\n' "$DEPLOYMENT_SHA"
printf 'migration_status=%s\n' "$MIGRATION_STATUS"

probe health /health
probe readyz /readyz
auth_probe users_me /api/v1/users/me

printf 'backend staging smoke passed\n'
