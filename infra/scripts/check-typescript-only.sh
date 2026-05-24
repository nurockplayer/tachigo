#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: infra/scripts/check-typescript-only.sh [--root <repo-root>]

Fails when tracked JavaScript-like source/config/tooling/test files are present.
EOF
}

root_arg="."

while [ "$#" -gt 0 ]; do
  case "$1" in
    --root)
      if [ "$#" -lt 2 ]; then
        echo "--root requires a value" >&2
        exit 2
      fi
      root_arg="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

root_dir="$(cd "$root_arg" && pwd)"

if ! git -C "$root_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "TypeScript-only guardrail requires a git worktree: $root_dir" >&2
  exit 2
fi

is_allowed_exception() {
  local file_path="$1"

  case "$file_path" in
    plans/archive/*|plans/archived/*|docs/archive/*|docs/archived/*|docs/*/archive/*|docs/*/archived/*)
      return 0
      ;;
    dist/*|*/dist/*|build/*|*/build/*|coverage/*|*/coverage/*|generated/*|*/generated/*|.next/*|*/.next/*|storybook-static/*|*/storybook-static/*)
      return 0
      ;;
    fixtures/*|*/fixtures/*|__fixtures__/*|*/__fixtures__/*)
      return 0
      ;;
  esac

  return 1
}

violations=""

while IFS= read -r -d '' file_path; do
  if is_allowed_exception "$file_path"; then
    continue
  fi

  violations="${violations}${file_path}"$'\n'
done < <(git -C "$root_dir" ls-files -z -- '*.js' '*.jsx' '*.mjs' '*.cjs')

if [ -n "$violations" ]; then
  {
    echo "TypeScript-only guardrail failed: tracked .js/.jsx/.mjs/.cjs source/config/tooling/test files are not allowed."
    echo
    echo "Use .ts/.tsx instead. For Node-executed TypeScript scripts, run with:"
    echo "  node --experimental-strip-types --no-warnings <script.ts>"
    echo
    echo "Disallowed tracked files:"
    printf '%s' "$violations" | sed 's/^/  - /'
    echo
    echo "Allowed exceptions are limited to archived docs/plans, generated build output, and fixture data paths."
  } >&2
  exit 1
fi

echo "TypeScript-only guardrail passed"
