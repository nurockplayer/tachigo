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

cat > "$tmp_dir/infra/scripts/session-index.sh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$SESSION_INDEX_LOG"
exit "${SESSION_INDEX_EXIT:-0}"
STUB
chmod +x "$tmp_dir/infra/scripts/session-index.sh"

cat > "$tmp_dir/fakebin/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "auth" ] && [ "${2:-}" = "status" ]; then
  exit 0
fi

if [ "${1:-}" = "pr" ] && [ "${2:-}" = "create" ]; then
  printf '%s\n' "$*" > "$GH_PR_CREATE_LOG"
  printf '%s\n' "https://github.com/nurockplayer/tachigo/pull/566"
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
  printf '%s\n' "refs #420" > body.md

  PATH="$tmp_dir/fakebin:$PATH" \
    GH_PR_CREATE_LOG="$tmp_dir/gh-pr-create.log" \
    PR_METADATA_CHECK_LOG="$tmp_dir/pr-metadata-check.log" \
    SESSION_INDEX_LOG="$tmp_dir/session-index.log" \
    "$tmp_dir/infra/scripts/pr-open.sh" \
      --title "[chore] Test auto-ready PR" \
      --body-file body.md \
      --auto-ready
)

grep -q -- '--draft' "$tmp_dir/gh-pr-create.log"
grep -q -- '--label auto-ready' "$tmp_dir/gh-pr-create.log"
grep -q -- '--title \[chore\] Test auto-ready PR' "$tmp_dir/pr-metadata-check.log"
grep -q -- 'add --pr 566' "$tmp_dir/session-index.log"
grep -q -- '--issue 420' "$tmp_dir/session-index.log"
grep -q -- '--worktree '"$tmp_dir" "$tmp_dir/session-index.log"

(
  cd "$tmp_dir"

  PATH="$tmp_dir/fakebin:$PATH" \
    GH_PR_CREATE_LOG="$tmp_dir/gh-pr-create-default.log" \
    PR_METADATA_CHECK_LOG="$tmp_dir/pr-metadata-check-default.log" \
    SESSION_INDEX_LOG="$tmp_dir/session-index-default.log" \
    "$tmp_dir/infra/scripts/pr-open.sh" \
      --title "[chore] Test default PR" \
      --body-file body.md
)

! grep -q -- '--draft' "$tmp_dir/gh-pr-create-default.log"
! grep -q -- '--label auto-ready' "$tmp_dir/gh-pr-create-default.log"
grep -q -- '--title \[chore\] Test default PR' "$tmp_dir/pr-metadata-check-default.log"
grep -q -- 'add --pr 566' "$tmp_dir/session-index-default.log"
grep -q -- '--issue 420' "$tmp_dir/session-index-default.log"
grep -q -- '--worktree '"$tmp_dir" "$tmp_dir/session-index-default.log"

(
  cd "$tmp_dir"

  PATH="$tmp_dir/fakebin:$PATH" \
    GH_PR_CREATE_LOG="$tmp_dir/gh-pr-create-index-fail.log" \
    PR_METADATA_CHECK_LOG="$tmp_dir/pr-metadata-check-index-fail.log" \
    SESSION_INDEX_LOG="$tmp_dir/session-index-fail.log" \
    SESSION_INDEX_EXIT=1 \
    "$tmp_dir/infra/scripts/pr-open.sh" \
      --title "[chore] Test index failure is non-fatal" \
      --body-file body.md \
      2>"$tmp_dir/index-fail.stderr"
)

grep -q -- 'warning: failed to update local Codex session index' "$tmp_dir/index-fail.stderr"

echo "pr-open regression tests passed"
