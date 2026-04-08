#!/usr/bin/env bash

set -euo pipefail

MAX_FILES=35
MAX_LINES=1800
ZERO_SHA=0000000000000000000000000000000000000000

usage() {
  cat <<'EOF'
Usage:
  scripts/pr-scope-check.sh
  scripts/pr-scope-check.sh --ref <local_ref> <local_sha> <remote> <remote_ref>

Checks the current branch diff against its likely base branch using the same
hard limits as PR Scope Police for:
  - changed files
  - diff lines
  - multiple product surfaces
EOF
}

resolve_candidate_ref() {
  local remote="$1"
  local candidate="${2:-}"

  [ -n "$candidate" ] || return 1

  case "$candidate" in
    refs/*)
      git rev-parse --verify --quiet "$candidate^{commit}" >/dev/null || return 1
      printf '%s\n' "$candidate"
      return 0
      ;;
    *)
      for ref in \
        "refs/remotes/$remote/$candidate" \
        "refs/heads/$candidate" \
        "$candidate"
      do
        git rev-parse --verify --quiet "$ref^{commit}" >/dev/null || continue
        printf '%s\n' "$ref"
        return 0
      done
      ;;
  esac

  return 1
}

detect_base_ref() {
  local remote="$1"
  local branch_name="$2"
  local remote_ref="$3"
  local configured_base merge_ref default_head candidate resolved

  configured_base=$(git config --get "branch.$branch_name.gh-merge-base" 2>/dev/null || true)
  resolved=$(resolve_candidate_ref "$remote" "$configured_base" 2>/dev/null || true)
  if [ -n "$resolved" ]; then
    printf '%s\n' "$resolved"
    return 0
  fi

  merge_ref=$(git config --get "branch.$branch_name.merge" 2>/dev/null || true)
  if [ -n "$merge_ref" ] && [ "$merge_ref" != "$remote_ref" ]; then
    resolved=$(resolve_candidate_ref "$remote" "$merge_ref" 2>/dev/null || true)
    if [ -n "$resolved" ]; then
      printf '%s\n' "$resolved"
      return 0
    fi
  fi

  default_head=$(git symbolic-ref "refs/remotes/$remote/HEAD" 2>/dev/null || true)
  for candidate in "refs/remotes/$remote/develop" "refs/remotes/$remote/main" "$default_head"; do
    resolved=$(resolve_candidate_ref "$remote" "$candidate" 2>/dev/null || true)
    if [ -n "$resolved" ] && [ "$resolved" != "$remote_ref" ]; then
      printf '%s\n' "$resolved"
      return 0
    fi
  done

  return 1
}

count_product_surfaces() {
  local files="$1"
  local count=0

  local touches_backend=0
  local touches_dashboard=0
  local touches_tachimint=0
  local touches_contracts=0

  if printf '%s\n' "$files" | grep -qE '^backend/'; then
    touches_backend=1
  fi
  if printf '%s\n' "$files" | grep -qE '^dashboard/'; then
    touches_dashboard=1
  fi
  if printf '%s\n' "$files" | grep -qE '^tachimint/'; then
    touches_tachimint=1
  fi
  if printf '%s\n' "$files" | grep -qE '^contracts/'; then
    touches_contracts=1
  fi

  count=$((touches_backend + touches_dashboard + touches_tachimint + touches_contracts))
  printf '%s\n' "$count"
}

list_product_surfaces() {
  local files="$1"
  local surfaces=()

  if printf '%s\n' "$files" | grep -qE '^backend/'; then
    surfaces+=("backend")
  fi
  if printf '%s\n' "$files" | grep -qE '^dashboard/'; then
    surfaces+=("dashboard")
  fi
  if printf '%s\n' "$files" | grep -qE '^tachimint/'; then
    surfaces+=("tachimint")
  fi
  if printf '%s\n' "$files" | grep -qE '^contracts/'; then
    surfaces+=("contracts")
  fi

  if [ "${#surfaces[@]}" -eq 0 ]; then
    printf 'none\n'
    return 0
  fi

  local joined=""
  local surface
  for surface in "${surfaces[@]}"; do
    if [ -n "$joined" ]; then
      joined="$joined, "
    fi
    joined="$joined$surface"
  done
  printf '%s\n' "$joined"
}

run_check() {
  local local_ref="$1"
  local local_sha="$2"
  local remote="$3"
  local remote_ref="$4"

  [ -n "$local_ref" ] || return 0
  [ "$local_sha" != "$ZERO_SHA" ] || return 0

  case "$local_ref" in
    refs/heads/*) ;;
    *)
      return 0
      ;;
  esac

  local branch_name="${local_ref#refs/heads/}"
  local base_ref base_sha base files changed_files log_numstat_output total_lines
  base_ref=$(detect_base_ref "$remote" "$branch_name" "$remote_ref") || {
    echo "無法判斷 branch '$branch_name' 的 base ref，略過 scope 檢查。" >&2
    return 0
  }
  base_sha=$(git rev-parse "$base_ref^{commit}" 2>/dev/null) || return 0
  base=$(git merge-base "$local_sha" "$base_sha" 2>/dev/null) || return 0

  files=$(git diff-tree -r --no-commit-id --name-only "$base" "$local_sha")
  changed_files=$(printf '%s\n' "$files" | sed '/^$/d' | wc -l | tr -d ' ')
  log_numstat_output=$(git log --format=tformat: --numstat "$base..$local_sha")
  total_lines=$(printf '%s\n' "$log_numstat_output" | awk '
    NF >= 3 {
      add = ($1 == "-" ? 0 : $1)
      del = ($2 == "-" ? 0 : $2)
      sum += add + del
    }
    END { print sum + 0 }
  ')

  local product_surface_count product_surfaces
  product_surface_count=$(count_product_surfaces "$files")
  product_surfaces=$(list_product_surfaces "$files")

  local failures=()
  if [ "$changed_files" -gt "$MAX_FILES" ]; then
    failures+=("Changed files $changed_files 超過上限 $MAX_FILES")
  fi
  if [ "$total_lines" -gt "$MAX_LINES" ]; then
    failures+=("Diff lines $total_lines 超過上限 $MAX_LINES")
  fi
  if [ "$product_surface_count" -gt 1 ]; then
    failures+=("同時修改多個 product surface：$product_surfaces")
  fi

  if [ "${#failures[@]}" -eq 0 ]; then
    echo "PR scope check passed: branch=$branch_name base=${base_ref#refs/remotes/} files=$changed_files lines=$total_lines surfaces=$product_surfaces"
    return 0
  fi

  echo ""
  echo "╔══════════════════════════════════════════════════════╗"
  echo "║  PR Scope Police — pre-push blocked                 ║"
  echo "╠══════════════════════════════════════════════════════╣"
  printf "║  Branch        : %-36s║\n" "$branch_name"
  printf "║  Base ref      : %-36s║\n" "${base_ref#refs/remotes/}"
  printf "║  Changed files : %-36s║\n" "$changed_files / $MAX_FILES"
  printf "║  Diff lines    : %-36s║\n" "$total_lines / $MAX_LINES"
  printf "║  Surfaces      : %-36s║\n" "$product_surfaces"
  echo "╠══════════════════════════════════════════════════════╣"
  local failure
  for failure in "${failures[@]}"; do
    printf "║  %-50s║\n" "$failure"
  done
  echo "╚══════════════════════════════════════════════════════╝"
  echo ""
  echo "請先拆 branch / 縮 scope，再重新 push。"

  return 1
}

main() {
  local remote="origin"
  local local_ref local_sha remote_ref

  case "${1:-}" in
    --ref)
      shift
      [ "$#" -eq 4 ] || {
        usage >&2
        exit 2
      }
      run_check "$1" "$2" "$3" "$4"
      ;;
    "")
      local_ref=$(git symbolic-ref HEAD 2>/dev/null) || {
        echo "目前不在 branch 上，無法執行 pr-scope-check。" >&2
        exit 2
      }
      local_sha=$(git rev-parse HEAD)
      remote_ref="$local_ref"
      run_check "$local_ref" "$local_sha" "$remote" "$remote_ref"
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"
