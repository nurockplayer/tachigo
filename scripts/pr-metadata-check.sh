#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/pr-metadata-check.sh --title "<pr title>" --body-file <path> [--base develop] [--head <branch>]

Checks local PR metadata before opening a PR:
  - title prefix
  - required PR body sections
  - dependency PR state
  - backend contract gating for frontend PRs
  - title prefix vs changed product surface
EOF
}

parse_repo_from_remote() {
  local remote_url
  remote_url=$(git remote get-url origin)

  case "$remote_url" in
    git@github.com:*)
      remote_url="${remote_url#git@github.com:}"
      printf '%s\n' "${remote_url%.git}"
      return 0
      ;;
    https://github.com/*)
      remote_url="${remote_url#https://github.com/}"
      printf '%s\n' "${remote_url%.git}"
      return 0
      ;;
  esac

  echo "無法從 origin remote 解析 GitHub repo：$remote_url" >&2
  return 1
}

detect_changed_files() {
  local base="$1"
  local head="$2"
  git diff-tree -r --no-commit-id --name-only "$base" "$head"
}

product_surfaces_from_files() {
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

has_checked_item() {
  local pattern="$1"
  local file="$2"
  if grep -Eq "^[[:space:]-]*\\[[xX]\\][[:space:]]+$pattern([[:space:]]|\$)" "$file"; then
    return 0
  fi
  return 1
}

main() {
  local title=""
  local body_file=""
  local base_branch="develop"
  local head_branch=""

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
  [ -f "$body_file" ] || {
    echo "找不到 body file：$body_file" >&2
    exit 2
  }

  if [ -z "$head_branch" ]; then
    head_branch=$(git branch --show-current)
  fi
  [ -n "$head_branch" ] || {
    echo "無法判斷目前 branch，請用 --head 指定。" >&2
    exit 2
  }

  local title_prefix=""
  case "$title" in
    "[backend]"*) title_prefix="[backend]" ;;
    "[frontend]"*) title_prefix="[frontend]" ;;
    "[contract]"*) title_prefix="[contract]" ;;
    "[discussion]"*) title_prefix="[discussion]" ;;
  esac

  local repo head_sha base_ref base_sha merge_base changed_files product_surfaces
  local failures=()

  [ -n "$title_prefix" ] || failures+=("PR title 必須以 [backend] / [frontend] / [contract] / [discussion] 開頭")

  repo=$(parse_repo_from_remote) || exit 2
  head_sha=$(git rev-parse "refs/heads/$head_branch^{commit}" 2>/dev/null) || {
    echo "找不到 head branch：$head_branch" >&2
    exit 2
  }
  base_ref="refs/remotes/origin/$base_branch"
  base_sha=$(git rev-parse "$base_ref^{commit}" 2>/dev/null) || {
    echo "找不到 base branch：origin/$base_branch" >&2
    exit 2
  }
  merge_base=$(git merge-base "$head_sha" "$base_sha" 2>/dev/null) || {
    echo "無法計算 merge-base：$head_branch vs $base_branch" >&2
    exit 2
  }

  changed_files=$(detect_changed_files "$merge_base" "$head_sha")
  product_surfaces=$(product_surfaces_from_files "$changed_files")

  grep -Eq '#[0-9]+' "$body_file" || failures+=("PR body 必須引用至少一個 issue / PR 編號，例如 #123")
  grep -Eq 'Source of truth[：:]' "$body_file" || failures+=("PR body 缺少 Source of truth")
  grep -Eq 'Depends on PR[：:]' "$body_file" || failures+=("PR body 缺少 Depends on PR")
  grep -Eq '本 PR 明確不做' "$body_file" || failures+=("PR body 缺少 本 PR 明確不做 區塊")

  local depends_on_raw
  depends_on_raw=$(sed -nE 's/.*Depends on PR[：:][[:space:]]*([^[:space:]].*)/\1/p' "$body_file" | head -n1 | sed 's/[[:space:]]*$//')
  [ -n "$depends_on_raw" ] || failures+=("Depends on PR 必須填 none 或 #123")

  local backend_contract_yes=0
  local backend_contract_no=0
  has_checked_item "yes" "$body_file" && backend_contract_yes=1
  has_checked_item "no" "$body_file" && backend_contract_no=1

  if [ "$backend_contract_yes" -eq 0 ] && [ "$backend_contract_no" -eq 0 ]; then
    failures+=("Backend contract already in develop 必須勾選 yes 或 no")
  fi
  if [ "$backend_contract_yes" -eq 1 ] && [ "$backend_contract_no" -eq 1 ]; then
    failures+=("Backend contract already in develop 不能同時勾 yes 與 no")
  fi

  local stacked_on_dependency=0
  local intentionally_blocked=0
  has_checked_item "stacked on dependency branch" "$body_file" && stacked_on_dependency=1
  has_checked_item "intentionally blocked until dependency merges" "$body_file" && intentionally_blocked=1

  if [ "$backend_contract_no" -eq 1 ] && [ "$stacked_on_dependency" -eq 0 ] && [ "$intentionally_blocked" -eq 0 ]; then
    failures+=("backend contract 不在 develop 時，必須勾選 stacked 或 intentionally blocked")
  fi

  if [ "$title_prefix" = "[frontend]" ] && printf '%s\n' "$changed_files" | grep -qE '^backend/'; then
    failures+=("[frontend] PR 不可修改 backend/")
  fi
  if [ "$title_prefix" = "[backend]" ] && printf '%s\n' "$changed_files" | grep -qE '^(dashboard|tachimint)/'; then
    failures+=("[backend] PR 不可修改 dashboard/ 或 tachimint/")
  fi
  if [ "$title_prefix" = "[contract]" ] && printf '%s\n' "$changed_files" | grep -qE '^(backend|dashboard|tachimint)/'; then
    failures+=("[contract] PR 不可修改 backend/、dashboard/、tachimint/")
  fi

  local depends_on_lower=""
  depends_on_lower=$(printf '%s' "$depends_on_raw" | tr '[:upper:]' '[:lower:]')

  if [ -n "$depends_on_raw" ] && [ "$depends_on_lower" != "none" ]; then
    if [[ ! "$depends_on_raw" =~ ^#([0-9]+)$ ]]; then
      failures+=("Depends on PR 只能填 none 或單一 PR 編號，例如 #123")
    else
      local dep_number="${BASH_REMATCH[1]}"
      command -v gh >/dev/null 2>&1 || {
        echo "需要先安裝 gh，才能檢查依賴 PR 狀態。" >&2
        exit 2
      }
      gh auth status >/dev/null 2>&1 || {
        echo "需要先登入 gh（gh auth login），才能檢查依賴 PR 狀態。" >&2
        exit 2
      }

      local dep_state dep_merged_at
      dep_state=$(gh pr view "$dep_number" --repo "$repo" --json state --jq '.state' 2>/dev/null || true)
      dep_merged_at=$(gh pr view "$dep_number" --repo "$repo" --json mergedAt --jq '.mergedAt // ""' 2>/dev/null || true)

      if [ -z "$dep_state" ]; then
        failures+=("無法讀取依賴 PR #$dep_number 的狀態")
      elif [ -z "$dep_merged_at" ]; then
        failures+=("依賴 PR #$dep_number 尚未 merge，請先等它進 develop 再開 PR")
      fi
    fi
  fi

  if [ "$title_prefix" = "[frontend]" ] && [ "$backend_contract_no" -eq 1 ]; then
    failures+=("[frontend] PR 不應在 backend contract 尚未進 develop 時開出")
  fi

  if [ "${#failures[@]}" -gt 0 ]; then
    echo ""
    echo "PR metadata check failed:"
    local failure
    for failure in "${failures[@]}"; do
      echo "  - $failure"
    done
    echo ""
    echo "Title   : $title"
    echo "Base    : $base_branch"
    echo "Head    : $head_branch"
    echo "Surface : $product_surfaces"
    exit 1
  fi

  echo "PR metadata check passed: title=$title_prefix base=$base_branch head=$head_branch surface=$product_surfaces"
}

main "$@"
