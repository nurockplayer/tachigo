#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  infra/scripts/pr-open.sh --title "<pr title>" --body-file <path> [--base develop] [--head <branch>] [--draft] [--auto-ready]

Runs local PR metadata checks, then opens a PR with gh.
EOF
}

extract_issue_number() {
  local body_file="$1"

  grep -Eio '(refs|closes|fixes|resolves)[[:space:]]+#[0-9]+' "$body_file" 2>/dev/null \
    | head -n 1 \
    | grep -Eo '[0-9]+' \
    || true
}

extract_pr_number() {
  sed -n -E 's#.*github\.com/.*/pull/([0-9]+).*#\1#p' | tail -n 1
}

update_session_index() {
  local root_dir="$1"
  local pr_number="$2"
  local issue_number="$3"
  local session_index="$root_dir/infra/scripts/session-index.sh"
  local cmd=("$session_index" add --pr "$pr_number" --worktree "$root_dir")

  [ -x "$session_index" ] || return 0
  if [ -n "$issue_number" ]; then
    cmd+=(--issue "$issue_number")
  fi

  if "${cmd[@]}" >/dev/null; then
    echo "Updated local Codex session index for PR #$pr_number."
  else
    echo "warning: failed to update local Codex session index for PR #$pr_number" >&2
  fi
}

main() {
  local title=""
  local body_file=""
  local base_branch="develop"
  local head_branch=""
  local draft=0
  local auto_ready=0
  local pr_output=""
  local pr_number=""
  local issue_number=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --title)
        title="${2:-}"
        shift 2
        ;;
      --body-file)
        body_file="${2:-}"
        shift 2
        ;;
      --base)
        base_branch="${2:-}"
        shift 2
        ;;
      --head)
        head_branch="${2:-}"
        shift 2
        ;;
      --draft)
        draft=1
        shift
        ;;
      --auto-ready)
        auto_ready=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        usage >&2
        echo "未知參數：$1" >&2
        exit 2
        ;;
    esac
  done

  [ -n "$title" ] || { echo "--title 為必填" >&2; exit 2; }
  [ -n "$body_file" ] || { echo "--body-file 為必填" >&2; exit 2; }
  [ -f "$body_file" ] || { echo "找不到 body file：$body_file" >&2; exit 2; }

  if [ -z "$head_branch" ]; then
    head_branch=$(git branch --show-current)
  fi
  [ -n "$head_branch" ] || { echo "無法判斷目前 branch，請用 --head 指定。" >&2; exit 2; }

  command -v gh >/dev/null 2>&1 || { echo "需要 gh 才能開 PR。" >&2; exit 2; }
  gh auth status >/dev/null 2>&1 || { echo "gh 尚未認證；請先處理 gh auth，再重跑。" >&2; exit 2; }

  local root_dir
  root_dir="$(cd "$(dirname "$0")/../.." && pwd)"

  local metadata_cmd=(
    "$root_dir/infra/scripts/pr-metadata-check.sh"
    --title "$title"
    --body-file "$body_file"
    --base "$base_branch"
    --head "$head_branch"
  )
  if [ "$auto_ready" -eq 1 ]; then
    metadata_cmd+=(--auto-ready)
  fi
  "${metadata_cmd[@]}"

  local cmd=(gh pr create --base "$base_branch" --head "$head_branch" --title "$title" --body-file "$body_file")
  if [ "$draft" -eq 1 ] || [ "$auto_ready" -eq 1 ]; then
    cmd+=(--draft)
  fi
  if [ "$auto_ready" -eq 1 ]; then
    cmd+=(--label auto-ready)
  fi

  echo "Opening PR with gh..."
  pr_output="$("${cmd[@]}")"
  printf '%s\n' "$pr_output"

  pr_number="$(printf '%s\n' "$pr_output" | extract_pr_number)"
  if [ -n "$pr_number" ]; then
    issue_number="$(extract_issue_number "$body_file")"
    update_session_index "$root_dir" "$pr_number" "$issue_number"
  fi
}

main "$@"
