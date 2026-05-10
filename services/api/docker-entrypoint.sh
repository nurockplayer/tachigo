#!/bin/sh
set -eu

: "${ATLAS_DATABASE_URL:?ATLAS_DATABASE_URL is required to apply database migrations}"

atlas migrate apply --dir "file://migrations" --url "$ATLAS_DATABASE_URL"

exec "$@"
