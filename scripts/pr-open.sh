#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/pr-open.sh --title "<pr title>" --body-file <path> [--base develop] [--head <branch>] [--draft]

Runs local PR checks, then opens a PR with gh.
EOF
}

main() {
  local title=""
  local body_file=""
  local base_branch="develop"
  local head_branch=""
  local draft=0

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

  [ -n "$title" ] || {
    echo "--title 為必填" >&2
    exit 2
  }
  [ -n "$body_file" ] || {
    echo "--body-file 為必填" >&2
    exit 2
  }

  if [ -z "$head_branch" ]; then
    head_branch=$(git branch --show-current)
  fi
  [ -n "$head_branch" ] || {
    echo "無法判斷目前 branch，請用 --head 指定。" >&2
    exit 2
  }

  local root_dir
  root_dir="$(cd "$(dirname "$0")/.." && pwd)"

  "$root_dir/scripts/pr-scope-check.sh"
  "$root_dir/scripts/pr-metadata-check.sh" \
    --title "$title" \
    --body-file "$body_file" \
    --base "$base_branch" \
    --head "$head_branch"

  local cmd=(gh pr create --base "$base_branch" --head "$head_branch" --title "$title" --body-file "$body_file")
  if [ "$draft" -eq 1 ]; then
    cmd+=(--draft)
  fi

  echo "Opening PR with gh..."
  "${cmd[@]}"
}

main "$@"
