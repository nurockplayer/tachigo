#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/commit-message-check.sh <commit-message-file>

Validates a git commit message against the repo commit policy.
EOF
}

main() {
  local message_file="${1:-}"
  [ -n "$message_file" ] || { usage >&2; exit 2; }
  [ -f "$message_file" ] || { echo "找不到 commit message file：$message_file" >&2; exit 2; }

  local subject
  subject="$(sed -n '1p' "$message_file")"

  if [[ "$subject" =~ ^Merge[[:space:]] ]] || [[ "$subject" =~ ^Revert[[:space:]] ]]; then
    exit 0
  fi

  local allowed_types='feat|fix|docs|chore|refactor|test'
  if ! [[ "$subject" =~ ^(${allowed_types}):[[:space:]].+ ]]; then
    cat >&2 <<'EOF'
commit subject 必須符合 `<type>: <short description>`，type 限定為：
feat / fix / docs / chore / refactor / test
EOF
    exit 1
  fi

  if ! grep -Eq '^(refs|closes) #[0-9]+$' "$message_file"; then
    cat >&2 <<'EOF'
commit message 必須包含一行 `refs #<issue號碼>` 或 `closes #<issue號碼>`。
EOF
    exit 1
  fi
}

main "$@"
