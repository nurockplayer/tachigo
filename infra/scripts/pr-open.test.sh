#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/infra/scripts" "$tmp_dir/fakebin"
cp "$root_dir/infra/scripts/pr-open.sh" "$tmp_dir/infra/scripts/pr-open.sh"

cat > "$tmp_dir/infra/scripts/pr-metadata-check.sh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$PR_METADATA_CHECK_LOG"
STUB
chmod +x "$tmp_dir/infra/scripts/pr-metadata-check.sh"

cat > "$tmp_dir/fakebin/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "auth" ] && [ "${2:-}" = "status" ]; then
  exit 0
fi

if [ "${1:-}" = "pr" ] && [ "${2:-}" = "create" ]; then
  printf '%s\n' "$*" > "$GH_PR_CREATE_LOG"
  exit 0
fi

echo "unexpected gh invocation: $*" >&2
exit 1
STUB
chmod +x "$tmp_dir/fakebin/gh"

(
  cd "$tmp_dir"
  git init -q
  git checkout -q -b test/pr-open
  touch body.md

  PATH="$tmp_dir/fakebin:$PATH" \
    GH_PR_CREATE_LOG="$tmp_dir/gh-pr-create.log" \
    PR_METADATA_CHECK_LOG="$tmp_dir/pr-metadata-check.log" \
    "$tmp_dir/infra/scripts/pr-open.sh" \
      --title "[chore] Test auto-ready PR" \
      --body-file body.md \
      --auto-ready
)

grep -q -- '--draft' "$tmp_dir/gh-pr-create.log"
grep -q -- '--label auto-ready' "$tmp_dir/gh-pr-create.log"
grep -q -- '--title \[chore\] Test auto-ready PR' "$tmp_dir/pr-metadata-check.log"

echo "pr-open regression tests passed"
