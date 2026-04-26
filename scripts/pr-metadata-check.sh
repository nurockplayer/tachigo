#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/pr-metadata-check.sh --title "<pr title>" --body-file <path> [--base develop] [--head <branch>] [--repo owner/name]

Checks local PR metadata against the current PR Scope Police rules before opening a PR.
EOF
}

parse_repo_from_remote() {
  local remote_url
  remote_url=$(git remote get-url origin)

  case "$remote_url" in
    git@github.com:*)
      remote_url="${remote_url#git@github.com:}"
      printf '%s\n' "${remote_url%.git}"
      ;;
    https://github.com/*)
      remote_url="${remote_url#https://github.com/}"
      printf '%s\n' "${remote_url%.git}"
      ;;
    *)
      echo "無法從 origin remote 解析 GitHub repo：$remote_url" >&2
      return 1
      ;;
  esac
}

is_docs_or_template_path() {
  local name="$1"

  case "$name" in
    docs/*|plans/*|.github/ISSUE_TEMPLATE/*|.github/PULL_REQUEST_TEMPLATE.md|.gitignore|.gitattributes)
      return 0
      ;;
    *.md)
      [ "${name#*/}" = "$name" ]
      return
      ;;
  esac

  return 1
}

checked_item() {
  local section="$1"
  local label="$2"
  local file="$3"

  awk -v section="$section" -v label="$label" '
    BEGIN {
      in_section = 0
      found = 0
      pattern = "^[[:space:]]*-[[:space:]]*\\[[xX]\\][[:space:]]+" label "([[:space:]]|$)"
    }
    $0 ~ "^[[:space:]]*[-]*[[:space:]]*" section "[：:][[:space:]]*$" {
      in_section = 1
      next
    }
    in_section && $0 ~ /^##[[:space:]]/ { exit }
    in_section && $0 ~ /^[[:space:]]*$/ { exit }
    in_section && $0 ~ pattern {
      found = 1
      exit
    }
    END {
      exit(found ? 0 : 1)
    }
  ' "$file"
}

main() {
  local title=""
  local body_file=""
  local base_branch="develop"
  local head_branch=""
  local repo=""

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
      --repo)
        repo="${2:-}"
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

  [ -n "$title" ] || { echo "--title 為必填" >&2; exit 2; }
  [ -n "$body_file" ] || { echo "--body-file 為必填" >&2; exit 2; }
  [ -f "$body_file" ] || { echo "找不到 body file：$body_file" >&2; exit 2; }

  if [ -z "$head_branch" ]; then
    head_branch=$(git branch --show-current)
  fi
  [ -n "$head_branch" ] || { echo "無法判斷目前 branch，請用 --head 指定。" >&2; exit 2; }

  local allowed_prefixes="[backend] [frontend] [contract] [discussion] [release] [infra] [chore]"
  local title_prefix=""
  local prefix
  for prefix in $allowed_prefixes; do
    case "$title" in
      "$prefix"*) title_prefix="$prefix"; break ;;
    esac
  done

  local failures=()
  local warn_surfaces=()
  [ -n "$title_prefix" ] || failures+=("PR title 必須以其中一個 prefix 開頭：$allowed_prefixes")

  local base_ref=""
  local head_ref="refs/heads/$head_branch"
  local base_sha head_sha merge_base
  local base_candidates=(
    "refs/remotes/origin/$base_branch"
    "refs/remotes/$base_branch"
  )
  local candidate
  for candidate in "${base_candidates[@]}"; do
    base_sha=$(git rev-parse "$candidate^{commit}" 2>/dev/null || true)
    if [ -n "$base_sha" ]; then
      base_ref="$candidate"
      break
    fi
  done
  [ -n "$base_ref" ] || {
    echo "找不到 base branch。已嘗試：${base_candidates[*]}" >&2
    exit 2
  }
  head_sha=$(git rev-parse "$head_ref^{commit}" 2>/dev/null) || {
    echo "找不到 head branch：$head_branch" >&2
    exit 2
  }
  merge_base=$(git merge-base "$base_sha" "$head_sha" 2>/dev/null) || {
    echo "無法計算 merge-base：$base_branch vs $head_branch" >&2
    exit 2
  }

  local changed_files
  changed_files=$(git diff --name-only "$merge_base" "$head_sha") || {
    echo "無法計算 diff：$merge_base vs $head_sha" >&2
    exit 2
  }

  local touches_backend=0 touches_dashboard=0 touches_tachimint=0 touches_contracts=0 docs_only=1
  local file
  while IFS= read -r file; do
    [ -n "$file" ] || continue
    case "$file" in
      backend/*) touches_backend=1 ;;
      dashboard/*) touches_dashboard=1 ;;
      tachimint/*) touches_tachimint=1 ;;
      contracts/*) touches_contracts=1 ;;
    esac
    if ! is_docs_or_template_path "$file"; then
      docs_only=0
    fi
  done <<< "$changed_files"

  local product_surface_count=0
  [ "$touches_backend" -eq 1 ] && product_surface_count=$((product_surface_count + 1))
  [ "$touches_dashboard" -eq 1 ] && product_surface_count=$((product_surface_count + 1))
  [ "$touches_tachimint" -eq 1 ] && product_surface_count=$((product_surface_count + 1))
  [ "$touches_contracts" -eq 1 ] && product_surface_count=$((product_surface_count + 1))

  local is_release_promotion=0
  if [ "$base_branch" = "main" ] && [ "$head_branch" = "develop" ]; then
    is_release_promotion=1
  fi

  local is_infra_or_chore=0
  if [ "$title_prefix" = "[infra]" ] || [ "$title_prefix" = "[chore]" ]; then
    is_infra_or_chore=1
  fi

  if [ "$is_release_promotion" -eq 1 ] && [ "$title_prefix" != "[release]" ]; then
    failures+=("develop -> main release PR 必須使用 [release] title prefix")
  fi
  if [ "$is_release_promotion" -eq 0 ] && [ "$title_prefix" = "[release]" ]; then
    failures+=("[release] 只能用於 develop -> main release promotion PR")
  fi

  local depends_on_raw=""
  depends_on_raw=$(grep -iE '^[[:space:]-]*Depends on PR[：:][[:space:]]*' "$body_file" | head -n1 | sed -E 's/^[^：:]*[：:][[:space:]]*//' | sed 's/[[:space:]]*$//')

  if [ "$is_infra_or_chore" -eq 0 ] && [ "$is_release_promotion" -eq 0 ]; then
    grep -Eq '#[0-9]+' "$body_file" || failures+=("PR body 必須引用至少一個 issue 或 PR 編號，例如 #123")
  fi

  if [ "$is_infra_or_chore" -eq 0 ]; then
    grep -Eq '(^|[[:space:]-])Source of truth[：:]' "$body_file" || failures+=("PR body 必須包含 Source of truth")
    grep -Eq '本 PR 明確不做' "$body_file" || failures+=("PR body 必須包含 本 PR 明確不做")
    grep -Eq '(^|[[:space:]-])Depends on PR[：:]' "$body_file" || failures+=("PR body 必須包含 Depends on PR")

    if [ -n "$depends_on_raw" ]; then
      if [ "$(printf '%s' "$depends_on_raw" | tr '[:upper:]' '[:lower:]')" != "none" ] && [[ ! "$depends_on_raw" =~ ^#[0-9]+$ ]]; then
        failures+=("Depends on PR 必須是 none 或單一 PR 編號，例如 #123")
      fi
    else
      failures+=("Depends on PR 必須填 none 或 #123")
    fi
  fi

  local backend_contract_yes=0 backend_contract_no=0
  checked_item "Backend contract already in develop" "yes" "$body_file" && backend_contract_yes=1
  checked_item "Backend contract already in develop" "no" "$body_file" && backend_contract_no=1

  local stacked_on_dependency=0 intentionally_blocked=0
  checked_item "If no, this PR is" "stacked on dependency branch" "$body_file" && stacked_on_dependency=1
  checked_item "If no, this PR is" "intentionally blocked until dependency merges" "$body_file" && intentionally_blocked=1

  if [ "$is_infra_or_chore" -eq 0 ] && [ "$docs_only" -eq 0 ]; then
    if [ "$backend_contract_yes" -eq 0 ] && [ "$backend_contract_no" -eq 0 ]; then
      failures+=("產品程式碼 PR 必須標記 Backend contract already in develop")
    fi
    if [ "$backend_contract_yes" -eq 1 ] && [ "$backend_contract_no" -eq 1 ]; then
      failures+=("Backend contract already in develop 不可同時勾 yes 與 no")
    fi
    if [ "$backend_contract_no" -eq 1 ] && [ "$stacked_on_dependency" -eq 0 ] && [ "$intentionally_blocked" -eq 0 ]; then
      failures+=("Backend contract 不在 develop 時，必須標記 stacked 或 intentionally blocked")
    fi
    if [ "$is_release_promotion" -eq 1 ] && [ "$backend_contract_yes" -eq 0 ]; then
      failures+=("develop -> main release PR 必須標記 Backend contract already in develop 為 yes")
    fi
  fi

  if [ "$title_prefix" = "[backend]" ] && { [ "$touches_dashboard" -eq 1 ] || [ "$touches_tachimint" -eq 1 ]; }; then
    failures+=("[backend] PR 不可修改 dashboard/ 或 tachimint/")
  fi
  if [ "$title_prefix" = "[frontend]" ] && [ "$touches_backend" -eq 1 ]; then
    failures+=("[frontend] PR 不可修改 backend/")
  fi
  if [ "$title_prefix" = "[contract]" ] && { [ "$touches_backend" -eq 1 ] || [ "$touches_dashboard" -eq 1 ] || [ "$touches_tachimint" -eq 1 ]; }; then
    failures+=("[contract] PR 不可修改 backend/、dashboard/ 或 tachimint/")
  fi
  if [ "$is_release_promotion" -eq 0 ] && [ "$product_surface_count" -gt 1 ]; then
    failures+=("PR 不可同時修改多個 product surface")
  fi

  # Warn when [infra]/[chore] touches product surface code.
  # Common cause: conflict-resolution commits sneak in code changes.
  # Reviewer will block if PR body promises "不動程式碼" but diff says otherwise.
  if [ "$is_infra_or_chore" -eq 1 ] && [ "$docs_only" -eq 0 ]; then
    [ "$touches_backend" -eq 1 ]   && warn_surfaces+=("backend/")
    [ "$touches_dashboard" -eq 1 ] && warn_surfaces+=("dashboard/")
    [ "$touches_tachimint" -eq 1 ] && warn_surfaces+=("tachimint/")
    [ "$touches_contracts" -eq 1 ] && warn_surfaces+=("contracts/")
  fi

  if [ "${#warn_surfaces[@]}" -gt 0 ]; then
    local _ws
    _ws=$(IFS=,; printf '%s' "${warn_surfaces[*]}")
    failures+=("$title_prefix PR 改動了 product surface 程式碼（${_ws}）—— 若 PR body「本 PR 明確不做」承諾了不動程式碼，請先更新再 push；若屬刻意改動，請改用對應的 [backend]/[frontend] prefix")
  fi

  if [ "$title_prefix" = "[frontend]" ] && [[ "$depends_on_raw" =~ ^#([0-9]+)$ ]] && [ "$backend_contract_no" -eq 1 ]; then
    command -v gh >/dev/null 2>&1 || { echo "需要 gh 才能檢查 dependency PR 狀態。" >&2; exit 2; }
    gh auth status >/dev/null 2>&1 || { echo "gh 尚未認證；請先處理 gh auth，再重跑。" >&2; exit 2; }
    [ -n "$repo" ] || repo=$(parse_repo_from_remote)

    local dep_number="${BASH_REMATCH[1]}"
    local dep_state
    dep_state=$(gh pr view "$dep_number" --repo "$repo" --json state --jq '.state' 2>/dev/null || true)
    if [ -z "$dep_state" ]; then
      failures+=("無法讀取 dependency PR #$dep_number")
    elif [ "$dep_state" = "MERGED" ]; then
      failures+=("[frontend] Backend contract PR #$dep_number is already MERGED into develop; please check 'Backend contract already in develop: yes'")
    elif [ "$dep_state" = "CLOSED" ]; then
      failures+=("[frontend] Dependency PR #$dep_number is CLOSED and not merged; please rebase or update dependency")
    elif [ "$dep_state" = "OPEN" ]; then
      echo "[info] Dependency PR #$dep_number state: $dep_state (stacked or blocked, OK)" >&2
    else
      failures+=("未知 dependency PR 狀態：$dep_state (#$dep_number)")
    fi
  fi

  if [ "${#failures[@]}" -gt 0 ]; then
    echo "PR metadata check failed:"
    local failure
    for failure in "${failures[@]}"; do
      echo "  - $failure"
    done
    exit 1
  fi

  echo "PR metadata check passed: title=$title_prefix base=$base_branch head=$head_branch docs_only=$docs_only"
}

main "$@"
