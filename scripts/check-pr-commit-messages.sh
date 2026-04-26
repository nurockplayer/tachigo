#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/check-pr-commit-messages.sh <base-ref> <head-ref>

Validates all non-merge commits in a PR range.
EOF
}

main() {
  local base_ref="${1:-}"
  local head_ref="${2:-}"
  [ -n "$base_ref" ] && [ -n "$head_ref" ] || { usage >&2; exit 2; }

  local base_sha head_sha merge_base
  base_sha="$(git rev-parse "${base_ref}^{commit}" 2>/dev/null)" || {
    echo "找不到 base ref：$base_ref" >&2
    exit 2
  }
  head_sha="$(git rev-parse "${head_ref}^{commit}" 2>/dev/null)" || {
    echo "找不到 head ref：$head_ref" >&2
    exit 2
  }
  merge_base="$(git merge-base "$base_sha" "$head_sha" 2>/dev/null)" || {
    echo "無法計算 merge-base：$base_ref vs $head_ref" >&2
    exit 2
  }

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  git rev-list --reverse --no-merges "${merge_base}..${head_sha}" > "$tmpdir/commits.txt"
  if [ ! -s "$tmpdir/commits.txt" ]; then
    echo "No non-merge commits to validate in ${merge_base}..${head_sha}"
    exit 0
  fi

  local root_dir
  root_dir="$(cd "$(dirname "$0")/.." && pwd)"

  local sha failed=0
  while IFS= read -r sha; do
    [ -n "$sha" ] || continue
    git show -s --format=%B "$sha" > "$tmpdir/$sha.txt"
    if ! "$root_dir/scripts/commit-message-check.sh" "$tmpdir/$sha.txt"; then
      echo "" >&2
      echo "Commit $sha 驗證失敗" >&2
      failed=1
    fi
  done < "$tmpdir/commits.txt"

  exit "$failed"
}

main "$@"
