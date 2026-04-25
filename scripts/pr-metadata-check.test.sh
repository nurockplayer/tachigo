#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/.." && pwd)"
head_branch="tmp-pr-metadata-check-test"

tmpdir="$(mktemp -d)"
trap 'git branch -D "$head_branch" >/dev/null 2>&1 || true; rm -rf "$tmpdir"' EXIT

git branch -f "$head_branch" origin/develop >/dev/null 2>&1

run_case() {
  local name="$1"
  local depends_line="$2"
  local body_file="$tmpdir/$name.md"

  cat > "$body_file" <<EOF
refs #123

## Scope 對齊

- Source of truth：test
- $depends_line
- 本 PR 明確不做：
  - no-op
EOF

  "$root_dir/scripts/pr-metadata-check.sh" \
    --title "[discussion] metadata parser regression" \
    --body-file "$body_file" \
    --base develop \
    --head "$head_branch" \
    >/dev/null
}

run_case fullwidth_no_space "Depends on PR：none"
run_case halfwidth_no_space "Depends on PR:none"

echo "pr-metadata-check regression tests passed"
