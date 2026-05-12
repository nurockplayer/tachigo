#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

codex_home="$tmp_dir/codex"
sessions_dir="$codex_home/sessions/2026/05/10"
mkdir -p "$sessions_dir" "$tmp_dir/fakebin"

session_id="019e07df-3d46-7913-9f70-28c1488562e7"
session_title="檢查專案測試完整性"
session_file="$sessions_dir/rollout-2026-05-08T22-55-31-$session_id.jsonl"
finder_session_id="019e10ea-df65-7501-9e29-de343fbda8b4"
finder_session_title="找出 PR #566 的 session"
finder_session_file="$sessions_dir/rollout-2026-05-10T17-04-48-$finder_session_id.jsonl"
branch="test/auth-persistence-failure-coverage"
commit="418ad9175bcf5c5c2ea5edcad53b253ab44596f8"
pr_url="https://github.com/nurockplayer/tachigo/pull/566"
pr_created_at="2026-05-10T04:48:25Z"

cat > "$codex_home/session_index.jsonl" <<EOF
{"id":"$session_id","thread_name":"$session_title","updated_at":"2026-05-10T04:48:50Z"}
{"id":"$finder_session_id","thread_name":"$finder_session_title","updated_at":"2026-05-10T08:05:15Z"}
EOF

cat > "$session_file" <<EOF
{"timestamp":"2026-05-10T04:39:18Z","type":"event_msg","payload":{"type":"agent_message","message":"建立 $branch worktree"}}
{"timestamp":"2026-05-10T04:47:03Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"git commit -m fix"}}
{"timestamp":"2026-05-10T04:48:22Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"make pr-open TITLE=\"[backend] fix: propagate auth persistence errors\""}}
{"timestamp":"2026-05-10T04:48:50Z","type":"event_msg","payload":{"type":"agent_message","message":"PR $pr_url commit $commit branch $branch ::git-create-pr"}}
EOF

cat > "$finder_session_file" <<EOF
{"timestamp":"2026-05-10T08:04:48Z","type":"session_meta","payload":{"id":"$finder_session_id"}}
{"timestamp":"2026-05-10T08:05:15Z","type":"event_msg","payload":{"type":"agent_message","message":"找出 $pr_url $commit $branch git commit git push make pr-open ::git-create-pr"}}
EOF

cat > "$tmp_dir/fakebin/gh" <<EOF
#!/usr/bin/env bash
set -euo pipefail

if [ "\${1:-}" = "pr" ] && [ "\${2:-}" = "view" ]; then
  printf '%s\t%s\t%s\t%s\t%s\n' "$branch" "$commit" "[backend] fix: propagate auth persistence errors" "$pr_url" "$pr_created_at"
  exit 0
fi

echo "unexpected gh invocation: \$*" >&2
exit 1
EOF
chmod +x "$tmp_dir/fakebin/gh"

index_file="$tmp_dir/tachigo-session-index.tsv"

PATH="$tmp_dir/fakebin:$PATH" \
CODEX_HOME="$codex_home" \
  "$root_dir/infra/scripts/session-index.sh" add \
    --pr 566 \
    --issue 420 \
    --worktree /tmp/worktree \
    --index-file "$index_file" \
    > "$tmp_dir/add.out"

grep -q "PR #566 indexed" "$tmp_dir/add.out"
grep -q "$session_id" "$index_file"
grep -q "$session_title" "$index_file"
grep -q "$commit" "$index_file"

CODEX_HOME="$codex_home" \
  "$root_dir/infra/scripts/session-index.sh" find \
    --pr 566 \
    --index-file "$index_file" \
    > "$tmp_dir/find-index.out"

grep -q "Session title: $session_title" "$tmp_dir/find-index.out"
grep -q "Session id: $session_id" "$tmp_dir/find-index.out"
grep -q "Left nav search: $session_title" "$tmp_dir/find-index.out"

rm -f "$index_file"

PATH="$tmp_dir/fakebin:$PATH" \
CODEX_HOME="$codex_home" \
  "$root_dir/infra/scripts/session-index.sh" find \
    --pr 566 \
    --index-file "$index_file" \
    > "$tmp_dir/find-fallback.out"

grep -q "No index entry found" "$tmp_dir/find-fallback.out"
grep -q "Session title: $session_title" "$tmp_dir/find-fallback.out"
grep -q "Session id: $session_id" "$tmp_dir/find-fallback.out"

echo "session-index regression tests passed"
