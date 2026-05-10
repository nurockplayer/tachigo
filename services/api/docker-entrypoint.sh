#!/bin/sh
set -eu

case "${1:-}" in
  air|/tachigo|tachigo)
    : "${ATLAS_DATABASE_URL:?ATLAS_DATABASE_URL is required to apply database migrations}"
    atlas migrate apply --dir "file://migrations" --url "$ATLAS_DATABASE_URL"
    ;;
esac

exec "$@"
