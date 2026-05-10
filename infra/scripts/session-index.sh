#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  infra/scripts/session-index.sh add --pr <number> [--issue <number>] [--worktree <path>] [--index-file <path>]
  infra/scripts/session-index.sh find --pr <number> [--index-file <path>]

Maintains a private local Codex session index for mapping PRs back to the
Codex IDE session title/id that produced them.
EOF
}

codex_home() {
  printf '%s\n' "${CODEX_HOME:-$HOME/.codex}"
}

default_index_file() {
  printf '%s/tachigo-session-index.tsv\n' "$(codex_home)"
}

clean_tsv_field() {
  printf '%s' "${1:-}" | tr '\t\r\n' '   '
}

pr_metadata() {
  local pr="$1"

  command -v gh >/dev/null 2>&1 || return 1
  gh pr view "$pr" \
    --json headRefName,commits,title,url,createdAt \
    --jq '[.headRefName, (.commits[-1].oid // ""), .title, .url, .createdAt] | @tsv'
}

session_title_for_id() {
  local id="$1"
  local index_json
  index_json="$(codex_home)/session_index.jsonl"

  [ -f "$index_json" ] || return 1
  grep -F "\"id\":\"$id\"" "$index_json" \
    | tail -n 1 \
    | sed -E 's/.*"thread_name":"([^"]*)".*/\1/'
}

session_id_from_file() {
  local file="$1"
  local base
  base="$(basename "$file")"

  printf '%s\n' "$base" \
    | sed -E 's/^.*-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\.jsonl$/\1/'
}

search_files() {
  local needle="$1"
  shift

  [ -n "$needle" ] || return 0
  if command -v rg >/dev/null 2>&1; then
    rg -l -F "$needle" "$@" 2>/dev/null || true
  else
    grep -R -l -F "$needle" "$@" 2>/dev/null || true
  fi
}

session_started_at() {
  local file="$1"

  sed -n -E '/"timestamp":/ { s/.*"timestamp":"([^"]+)".*/\1/; p; q; }' "$file"
}

score_session_file() {
  local file="$1"
  local branch="$2"
  local commit="$3"
  local pr_url="$4"
  local pr_created_at="$5"
  local score=0
  local started_at=""

  [ -n "$commit" ] && grep -q -F "$commit" "$file" && score=$((score + 20))
  [ -n "$branch" ] && grep -q -F "$branch" "$file" && score=$((score + 10))
  [ -n "$pr_url" ] && grep -q -F "$pr_url" "$file" && score=$((score + 10))
  grep -q -F "git commit" "$file" && score=$((score + 5))
  grep -q -F "git push" "$file" && score=$((score + 5))
  grep -q -F "make pr-open" "$file" && score=$((score + 8))
  grep -q -F "::git-create-pr" "$file" && score=$((score + 12))
  grep -q -F "::git-create-branch" "$file" && score=$((score + 4))

  if [ -n "$pr_created_at" ]; then
    started_at="$(session_started_at "$file" || true)"
    if [ -n "$started_at" ] && [[ "$started_at" > "$pr_created_at" ]]; then
      score=$((score - 1000))
    fi
  fi

  printf '%s\t%s\n' "$score" "$file"
}

detect_session() {
  local branch="$1"
  local commit="$2"
  local pr_url="$3"
  local pr_created_at="$4"
  local home
  local roots=()
  local candidates
  local best
  local score
  local file
  local id
  local title

  home="$(codex_home)"
  [ -d "$home/sessions" ] && roots+=("$home/sessions")
  [ -d "$home/archived_sessions" ] && roots+=("$home/archived_sessions")
  [ "${#roots[@]}" -gt 0 ] || return 1

  candidates="$(
    {
      search_files "$commit" "${roots[@]}"
      search_files "$branch" "${roots[@]}"
      search_files "$pr_url" "${roots[@]}"
    } | sort -u
  )"
  [ -n "$candidates" ] || return 1

  best="$(
    printf '%s\n' "$candidates" | while IFS= read -r file; do
      [ -n "$file" ] || continue
      score_session_file "$file" "$branch" "$commit" "$pr_url" "$pr_created_at"
    done | sort -nr | head -n 1
  )"

  [ -n "$best" ] || return 1
  score="${best%%	*}"
  file="${best#*	}"
  [ "$score" -gt 0 ] || return 1

  id="$(session_id_from_file "$file")"
  title="$(session_title_for_id "$id" || true)"
  [ -n "$title" ] || title="$(basename "$file")"

  printf '%s\t%s\t%s\t%s\n' "$id" "$title" "$file" "$score"
}

print_entry() {
  local line="$1"
  local pr issue branch commit session_id session_title session_file worktree pr_url pr_created_at indexed_at

  IFS=$'\t' read -r pr issue branch commit session_id session_title session_file worktree pr_url pr_created_at indexed_at <<EOF
$line
EOF

  printf 'PR #%s\n' "$pr"
  [ -n "$issue" ] && printf 'Issue: #%s\n' "$issue"
  printf 'Session title: %s\n' "$session_title"
  printf 'Session id: %s\n' "$session_id"
  printf 'Left nav search: %s\n' "$session_title"
  printf 'Branch: %s\n' "$branch"
  printf 'Commit: %s\n' "$commit"
  printf 'Worktree: %s\n' "$worktree"
  printf 'Session file: %s\n' "$session_file"
  printf 'PR URL: %s\n' "$pr_url"
  [ -n "$pr_created_at" ] && printf 'PR created at: %s\n' "$pr_created_at"
  printf 'Indexed at: %s\n' "$indexed_at"
}

find_index_line() {
  local pr="$1"
  local index_file="$2"

  [ -f "$index_file" ] || return 1
  awk -F '\t' -v pr="$pr" '$1 == pr { line = $0 } END { if (line != "") print line; else exit 1 }' "$index_file"
}

cmd_add() {
  local pr=""
  local issue=""
  local worktree=""
  local index_file=""
  local branch=""
  local commit=""
  local title=""
  local pr_url=""
  local pr_created_at=""
  local metadata=""
  local detected=""
  local session_id=""
  local session_title=""
  local session_file=""
  local score=""
  local tmp_file
  local indexed_at

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --pr) pr="${2:-}"; shift 2 ;;
      --issue) issue="${2:-}"; shift 2 ;;
      --worktree) worktree="${2:-}"; shift 2 ;;
      --index-file) index_file="${2:-}"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) usage >&2; echo "unknown argument: $1" >&2; exit 2 ;;
    esac
  done

  [ -n "$pr" ] || { echo "--pr is required" >&2; exit 2; }
  [ -n "$index_file" ] || index_file="$(default_index_file)"
  [ -n "$worktree" ] || worktree="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

  metadata="$(pr_metadata "$pr" || true)"
  if [ -n "$metadata" ]; then
    IFS=$'\t' read -r branch commit title pr_url pr_created_at <<EOF
$metadata
EOF
  fi

  detected="$(detect_session "$branch" "$commit" "$pr_url" "$pr_created_at" || true)"
  if [ -n "$detected" ]; then
    IFS=$'\t' read -r session_id session_title session_file score <<EOF
$detected
EOF
  fi

  indexed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  mkdir -p "$(dirname "$index_file")"
  tmp_file="$(mktemp)"
  if [ -f "$index_file" ]; then
    awk -F '\t' -v pr="$pr" '$1 != pr { print }' "$index_file" > "$tmp_file"
  fi
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$(clean_tsv_field "$pr")" \
    "$(clean_tsv_field "$issue")" \
    "$(clean_tsv_field "$branch")" \
    "$(clean_tsv_field "$commit")" \
    "$(clean_tsv_field "$session_id")" \
    "$(clean_tsv_field "$session_title")" \
    "$(clean_tsv_field "$session_file")" \
    "$(clean_tsv_field "$worktree")" \
    "$(clean_tsv_field "$pr_url")" \
    "$(clean_tsv_field "$pr_created_at")" \
    "$indexed_at" >> "$tmp_file"
  mv "$tmp_file" "$index_file"

  echo "PR #$pr indexed at $index_file"
  if [ -n "$session_title" ]; then
    echo "Left nav search: $session_title"
  else
    echo "Warning: no matching Codex session detected; try running session-find later." >&2
  fi
}

cmd_find() {
  local pr=""
  local index_file=""
  local line=""
  local branch=""
  local commit=""
  local title=""
  local pr_url=""
  local pr_created_at=""
  local metadata=""
  local detected=""
  local session_id=""
  local session_title=""
  local session_file=""
  local score=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --pr) pr="${2:-}"; shift 2 ;;
      --index-file) index_file="${2:-}"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) usage >&2; echo "unknown argument: $1" >&2; exit 2 ;;
    esac
  done

  [ -n "$pr" ] || { echo "--pr is required" >&2; exit 2; }
  [ -n "$index_file" ] || index_file="$(default_index_file)"

  line="$(find_index_line "$pr" "$index_file" || true)"
  if [ -n "$line" ]; then
    print_entry "$line"
    return 0
  fi

  echo "No index entry found for PR #$pr; searching Codex sessions by branch/commit..."
  metadata="$(pr_metadata "$pr" || true)"
  if [ -z "$metadata" ]; then
    echo "Unable to load PR metadata with gh; cannot fallback search." >&2
    exit 1
  fi
  IFS=$'\t' read -r branch commit title pr_url pr_created_at <<EOF
$metadata
EOF

  detected="$(detect_session "$branch" "$commit" "$pr_url" "$pr_created_at" || true)"
  if [ -z "$detected" ]; then
    echo "No matching Codex session found." >&2
    echo "Search keys: PR #$pr, branch '$branch', commit '$commit'"
    exit 1
  fi

  IFS=$'\t' read -r session_id session_title session_file score <<EOF
$detected
EOF

  printf 'PR #%s\n' "$pr"
  printf 'Session title: %s\n' "$session_title"
  printf 'Session id: %s\n' "$session_id"
  printf 'Left nav search: %s\n' "$session_title"
  printf 'Branch: %s\n' "$branch"
  printf 'Commit: %s\n' "$commit"
  printf 'Session file: %s\n' "$session_file"
  printf 'PR URL: %s\n' "$pr_url"
  printf 'PR created at: %s\n' "$pr_created_at"
  printf 'Fallback score: %s\n' "$score"
}

main() {
  local command="${1:-}"

  case "$command" in
    add)
      shift
      cmd_add "$@"
      ;;
    find)
      shift
      cmd_find "$@"
      ;;
    -h|--help|"")
      usage
      ;;
    *)
      usage >&2
      echo "unknown command: $command" >&2
      exit 2
      ;;
  esac
}

main "$@"
